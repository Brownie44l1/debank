package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/services"
)

type WalletHandler struct {
	service *services.WalletService
}

func NewWalletHandler(service *services.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

// Deposit handles POST /api/v1/deposit
func (h *WalletHandler) Deposit(c *gin.Context) {
	var req models.DepositRequest

	// 1. Parse and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"message": err.Error(),
		})
		return
	}

	// 2. Call service layer
	resp, err := h.service.Deposit(c.Request.Context(), req)
	if err != nil {
		statusCode := h.getStatusCode(err)
		c.JSON(statusCode, gin.H{
			"error":   "deposit failed",
			"message": err.Error(),
		})
		return
	}

	// 3. Return success response
	c.JSON(http.StatusOK, resp)
}

// Withdraw handles POST /api/v1/withdraw
func (h *WalletHandler) Withdraw(c *gin.Context) {
	var req models.WithdrawRequest

	// 1. Parse and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"message": err.Error(),
		})
		return
	}

	// 2. Call service layer
	resp, err := h.service.Withdraw(c.Request.Context(), req)
	if err != nil {
		statusCode := h.getStatusCode(err)
		c.JSON(statusCode, gin.H{
			"error":   "withdrawal failed",
			"message": err.Error(),
		})
		return
	}

	// 3. Return success response
	c.JSON(http.StatusOK, resp)
}

// GetBalance handles GET /api/v1/balance/:user_id
func (h *WalletHandler) GetBalance(c *gin.Context) {
	// 1. Parse user_id from URL parameter
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid user_id",
			"message": "user_id must be a number",
		})
		return
	}

	// 2. Call service layer
	balance, err := h.service.GetBalance(c.Request.Context(), userID)
	if err != nil {
		statusCode := h.getStatusCode(err)
		c.JSON(statusCode, gin.H{
			"error":   "failed to get balance",
			"message": err.Error(),
		})
		return
	}

	// 3. Return balance
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"balance": balance,
		"balance_ngn": float64(balance) / 100, // Convert kobo to naira for readability
	})
}

// getStatusCode maps service errors to HTTP status codes
func (h *WalletHandler) getStatusCode(err error) int {
	switch {
	case errors.Is(err, services.ErrInvalidAmount),
		errors.Is(err, services.ErrAmountTooSmall),
		errors.Is(err, services.ErrAmountTooLarge),
		errors.Is(err, services.ErrInvalidIdempotencyKey):
		return http.StatusBadRequest
	case err.Error() == "account not found":
		return http.StatusNotFound
	case err.Error() == "insufficient balance for withdrawal":
		return http.StatusUnprocessableEntity
	case err.Error() == "duplicate transaction detected":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// RegisterRoutes registers all wallet routes
func (h *WalletHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		v1.POST("/deposit", h.Deposit)
		v1.POST("/withdraw", h.Withdraw)
		v1.GET("/balance/:user_id", h.GetBalance)
	}
}