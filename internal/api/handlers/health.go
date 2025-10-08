package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET /health - returns API health status
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "debank-wallet-api",
		"version": "v1.0.0",
	})
}

// Readiness handles GET /ready - can be extended to check dependencies
func (h *HealthHandler) Readiness(c *gin.Context) {
	// In future, you can add checks for:
	// - Database connectivity
	// - External service availability
	// - Queue health, etc.
	
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"api": "ok",
			// "database": "ok",
			// "cache": "ok",
		},
	})
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.Health)
	router.GET("/ready", h.Readiness)
}