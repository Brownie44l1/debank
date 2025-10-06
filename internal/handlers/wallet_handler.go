package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/service"
	"github.com/gin-gonic/gin"
)

// ==============================================
// SERVICE INTERFACE (for testing)
// ==============================================

type WalletService interface {
	Deposit(ctx context.Context, req models.DepositRequest) (*models.TransactionResponse, error)
	Withdraw(ctx context.Context, req models.WithdrawRequest) (*models.TransactionResponse, error)
	Transfer(ctx context.Context, req models.TransferRequest) (*models.TransferResponse, error)
	GetBalance(ctx context.Context, userID int) (*models.BalanceResponse, error)
	GetTransactionHistory(ctx context.Context, userID, page, perPage int) (*models.TransactionHistoryResponse, error)
}

// ==============================================
// HANDLER (HTTP Layer ONLY)
// ==============================================

type WalletHandler struct {
	service WalletService
}

func NewWalletHandler(service WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

// ==============================================
// ENDPOINTS
// ==============================================

// Deposit handles POST /api/v1/deposit
func (h *WalletHandler) Deposit(c *gin.Context) {
	var req models.DepositRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	resp, err := h.service.Deposit(c.Request.Context(), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, resp)
}

// Withdraw handles POST /api/v1/withdraw
func (h *WalletHandler) Withdraw(c *gin.Context) {
	var req models.WithdrawRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	resp, err := h.service.Withdraw(c.Request.Context(), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, resp)
}

// Transfer handles POST /api/v1/transfer
func (h *WalletHandler) Transfer(c *gin.Context) {
	var req models.TransferRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	resp, err := h.service.Transfer(c.Request.Context(), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, resp)
}

// GetBalance handles GET /api/v1/balance/:user_id
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, err := parseUserID(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid user_id", err)
		return
	}

	resp, err := h.service.GetBalance(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, resp)
}

// GetTransactionHistory handles GET /api/v1/transactions/:user_id
func (h *WalletHandler) GetTransactionHistory(c *gin.Context) {
	userID, err := parseUserID(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid user_id", err)
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	resp, err := h.service.GetTransactionHistory(c.Request.Context(), userID, page, perPage)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, resp)
}

// ==============================================
// ROUTE REGISTRATION
// ==============================================

func (h *WalletHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		v1.POST("/deposit", h.Deposit)
		v1.POST("/withdraw", h.Withdraw)
		v1.POST("/transfer", h.Transfer)
		v1.GET("/balance/:user_id", h.GetBalance)
		v1.GET("/transactions/:user_id", h.GetTransactionHistory)
	}
}

// ==============================================
// HELPER FUNCTIONS
// ==============================================

// parseUserID extracts and validates user_id from URL parameter
func parseUserID(c *gin.Context) (int, error) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, errors.New("user_id must be a number")
	}
	if userID <= 0 {
		return 0, errors.New("user_id must be positive")
	}
	return userID, nil
}

// respondSuccess sends a successful JSON response
func respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// respondError sends an error JSON response
func respondError(c *gin.Context, statusCode int, message string, err error) {
	c.JSON(statusCode, gin.H{
		"error":   message,
		"message": err.Error(),
	})
}

// respondServiceError maps service errors to appropriate HTTP status codes and responses
func respondServiceError(c *gin.Context, err error) {
	statusCode, message := mapServiceError(err)
	c.JSON(statusCode, gin.H{
		"error":   message,
		"message": err.Error(),
	})
}

// mapServiceError maps service errors to HTTP status codes and user-friendly messages
func mapServiceError(err error) (int, string) {
	switch {
	// Validation errors (400 Bad Request)
	case errors.Is(err, service.ErrInvalidAmount):
		return http.StatusBadRequest, "Invalid amount"
	case errors.Is(err, service.ErrAmountTooSmall):
		return http.StatusBadRequest, "Amount too small"
	case errors.Is(err, service.ErrAmountTooLarge):
		return http.StatusBadRequest, "Amount too large"
	case errors.Is(err, service.ErrInvalidIdempotencyKey):
		return http.StatusBadRequest, "Idempotency key required"
	case errors.Is(err, service.ErrSameAccount):
		return http.StatusBadRequest, "Cannot transfer to same account"

	// Not found errors (404 Not Found)
	case errors.Is(err, service.ErrAccountNotFound):
		return http.StatusNotFound, "Account not found"

	// Business logic errors (422 Unprocessable Entity)
	case errors.Is(err, service.ErrInsufficientBalance):
		return http.StatusUnprocessableEntity, "Insufficient balance"

	// System errors (500 Internal Server Error)
	case errors.Is(err, service.ErrNegativeBalance):
		return http.StatusInternalServerError, "System error: negative balance detected"

	// Default (500 Internal Server Error)
	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}