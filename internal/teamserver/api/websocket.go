package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"c2/internal/teamserver/agent"
	"c2/internal/teamserver/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSHandler handles WebSocket connections for real-time agent interaction.
type WSHandler struct {
	agentMgr *agent.Manager
	jwtSecret string
}

// NewWSHandler creates a new WebSocket handler.
func NewWSHandler(agentMgr *agent.Manager, jwtSecret string) *WSHandler {
	return &WSHandler{
		agentMgr:  agentMgr,
		jwtSecret: jwtSecret,
	}
}

// HandleAgentWS handles WebSocket connections at /ws/agents/:id
func (h *WSHandler) HandleAgentWS(c *gin.Context) {
	agentID := c.Param("id")

	// Authenticate via query parameter token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	claims, err := auth.ValidateAccessToken(token, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	_ = claims // Token validated

	// Verify agent exists
	_, ok := h.agentMgr.Get(agentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[INFO] WebSocket connected for agent %s", agentID)

	// Subscribe to agent events
	eventCh := h.agentMgr.Notifier().Subscribe(agentID)
	defer h.agentMgr.Notifier().Unsubscribe(agentID, eventCh)

	// Read pump (client → server)
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Parse client message
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			// Handle ping
			if t, ok := msg["type"].(string); ok && t == "ping" {
				conn.WriteJSON(map[string]string{"type": "pong"})
			}
		}
	}()

	// Write pump (server → client): forward agent events
	for event := range eventCh {
		wsMsg := map[string]interface{}{
			"type":     string(event.Type),
			"agent_id": event.AgentID,
			"data":     event.Data,
		}
		if err := conn.WriteJSON(wsMsg); err != nil {
			log.Printf("[DEBUG] WebSocket write error for agent %s: %v", agentID, err)
			return
		}
	}
}
