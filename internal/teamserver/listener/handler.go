package listener

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"c2/internal/teamserver/database"
)

// Handler provides HTTP handlers for listener management.
type Handler struct {
	manager *Manager
	ctx     context.Context
}

// NewHandler creates a new listener HTTP handler.
func NewHandler(manager *Manager, ctx context.Context) *Handler {
	return &Handler{manager: manager, ctx: ctx}
}

// SetContext updates the context (called from Run after the real context is available).
func (h *Handler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// CreateListenerRequest is the JSON body for creating a listener.
type CreateListenerRequest struct {
	Name     string `json:"name" binding:"required"`
	Protocol string `json:"protocol" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
}

// CreateListener handles POST /api/v1/listeners
func (h *Handler) CreateListener(c *gin.Context) {
	var req CreateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "name, protocol, host, and port are required"},
		})
		return
	}

	if req.Port < 1 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "port must be between 1 and 65535"},
		})
		return
	}

	l, err := h.manager.Create(req.Name, req.Protocol, req.Host, req.Port)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{"code": "CONFLICT", "message": err.Error()},
		})
		return
	}

	dl, _ := h.manager.Get((*l).ID())
	c.JSON(http.StatusCreated, dl)
}

// ListListeners handles GET /api/v1/listeners
func (h *Handler) ListListeners(c *gin.Context) {
	listeners := h.manager.List()
	if listeners == nil {
		listeners = []database.Listener{}
	}
	c.JSON(http.StatusOK, listeners)
}

// GetListener handles GET /api/v1/listeners/:id
func (h *Handler) GetListener(c *gin.Context) {
	id := c.Param("id")
	l, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, l)
}

// DeleteListener handles DELETE /api/v1/listeners/:id
func (h *Handler) DeleteListener(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.Delete(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{"code": "CONFLICT", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Listener deleted"})
}

// StartListener handles POST /api/v1/listeners/:id/start
func (h *Handler) StartListener(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.Start(h.ctx, id); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{"code": "CONFLICT", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Listener started"})
}

// StopListener handles POST /api/v1/listeners/:id/stop
func (h *Handler) StopListener(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.Stop(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{"code": "CONFLICT", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Listener stopped"})
}
