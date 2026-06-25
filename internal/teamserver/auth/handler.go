package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fmt"

	"c2/internal/teamserver/database"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	DB          *sql.DB
	Secret      string
	RateLimiter *RateLimiter
	JWT         struct {
		AccessExpiry  time.Duration
		RefreshExpiry time.Duration
	}
}

// LoginRequest is the JSON body for the login endpoint.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the JSON response for successful login.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "username and password are required"},
		})
		return
	}

	// Rate limiting: max 5 attempts per IP per minute
	if h.RateLimiter != nil && !h.RateLimiter.Allow(c.ClientIP()) {
		resetAfter := h.RateLimiter.ResetAfter(c.ClientIP())
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"code":    "RATE_LIMITED",
				"message": fmt.Sprintf("Too many login attempts. Try again in %.0f seconds.", resetAfter.Seconds()),
			},
		})
		return
	}

	// Query operator
	var op database.Operator
	var lastLogin *string
	err := h.DB.QueryRow(
		"SELECT id, username, password_hash, role, last_login, active FROM operators WHERE username = ?",
		req.Username,
	).Scan(&op.ID, &op.Username, &op.PasswordHash, &op.Role, &lastLogin, &op.Active)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid credentials"},
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Database error"},
		})
		return
	}

	if !op.Active {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{"code": "FORBIDDEN", "message": "Account is disabled"},
		})
		return
	}

	// Verify password
	if !CheckPassword(op.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid credentials"},
		})
		return
	}

	// Generate tokens
	accessToken, err := GenerateAccessToken(op.ID, op.Username, op.Role, h.Secret, h.JWT.AccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Failed to generate access token"},
		})
		return
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Failed to generate refresh token"},
		})
		return
	}

	// Store refresh token
	refreshID := uuid.New().String()
	expiresAt := time.Now().Add(h.JWT.RefreshExpiry).UTC().Format(time.RFC3339)
	_, err = h.DB.Exec(
		"INSERT INTO refresh_tokens (id, operator_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		refreshID, op.ID, HashToken(refreshToken), expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Failed to store refresh token"},
		})
		return
	}

	// Update last_login
	now := time.Now().UTC().Format(time.RFC3339)
	h.DB.Exec("UPDATE operators SET last_login = ? WHERE id = ?", now, op.ID)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.JWT.AccessExpiry.Seconds()),
	})
}

// RefreshRequest is the JSON body for the refresh endpoint.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse is the JSON response for token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "refresh_token is required"},
		})
		return
	}

	tokenHash := HashToken(req.RefreshToken)

	var rt database.RefreshToken
	err := h.DB.QueryRow(
		"SELECT id, operator_id, token_hash, expires_at, revoked FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&rt.ID, &rt.OperatorID, &rt.TokenHash, &rt.ExpiresAt, &rt.Revoked)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid refresh token"},
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Database error"},
		})
		return
	}

	if rt.Revoked {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Token has been revoked"},
		})
		return
	}

	// Check expiry
	expiresAt, err := time.Parse(time.RFC3339, rt.ExpiresAt)
	if err != nil || time.Now().After(expiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Token has expired"},
		})
		return
	}

	// Get operator info
	var op database.Operator
	err = h.DB.QueryRow(
		"SELECT id, username, role, active FROM operators WHERE id = ?",
		rt.OperatorID,
	).Scan(&op.ID, &op.Username, &op.Role, &op.Active)
	if err != nil || !op.Active {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Operator not found or disabled"},
		})
		return
	}

	// Generate new access token
	accessToken, err := GenerateAccessToken(op.ID, op.Username, op.Role, h.Secret, h.JWT.AccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL", "message": "Failed to generate access token"},
		})
		return
	}

	c.JSON(http.StatusOK, RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(h.JWT.AccessExpiry.Seconds()),
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "refresh_token is required"},
		})
		return
	}

	tokenHash := HashToken(req.RefreshToken)
	h.DB.Exec("UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?", tokenHash)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	claims := GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Not authenticated"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       claims.OperatorID,
		"username": claims.Username,
		"role":     claims.Role,
	})
}
