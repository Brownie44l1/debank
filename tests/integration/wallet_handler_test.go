package integration
/* 
import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ==============================================
// MOCK SERVICE
// ==============================================

type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) Deposit(ctx context.Context, req models.DepositRequest) (*models.TransactionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionResponse), args.Error(1)
}

func (m *MockWalletService) Withdraw(ctx context.Context, req models.WithdrawRequest) (*models.TransactionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionResponse), args.Error(1)
}

func (m *MockWalletService) Transfer(ctx context.Context, req models.TransferRequest) (*models.TransferResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransferResponse), args.Error(1)
}

func (m *MockWalletService) GetBalance(ctx context.Context, userID int) (*models.BalanceResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BalanceResponse), args.Error(1)
}

func (m *MockWalletService) GetTransactionHistory(ctx context.Context, userID, page, perPage int) (*models.TransactionHistoryResponse, error) {
	args := m.Called(ctx, userID, page, perPage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionHistoryResponse), args.Error(1)
}

// ==============================================
// TEST SETUP
// ==============================================

func setupTest() (*gin.Engine, *MockWalletService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockService := new(MockWalletService)
	handler := NewWalletHandler(mockService)
	handler.RegisterRoutes(router)

	return router, mockService
}

// ==============================================
// DEPOSIT TESTS
// ==============================================

func TestDeposit_Success(t *testing.T) {
	router, mockService := setupTest()

	depositReq := models.DepositRequest{
		UserID:         1,
		Amount:         50000,
		IdempotencyKey: "dep_123",
		Reference:      "ref_123",
	}

	expectedResp := &models.TransactionResponse{
		TransactionID: 1,
		Status:        "posted",
		Balance:       50000,
		Reference:     "ref_123",
		Message:       "Successfully deposited ₦500.00",
	}

	mockService.On("Deposit", mock.Anything, depositReq).Return(expectedResp, nil)

	body, _ := json.Marshal(depositReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/deposit", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.TransactionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp.TransactionID, resp.TransactionID)
	assert.Equal(t, expectedResp.Balance, resp.Balance)
	mockService.AssertExpectations(t)
}

func TestDeposit_InvalidJSON(t *testing.T) {
	router, _ := setupTest()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/deposit", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request")
}

func TestDeposit_AmountTooSmall(t *testing.T) {
	router, mockService := setupTest()

	depositReq := models.DepositRequest{
		UserID:         1,
		Amount:         5000,
		IdempotencyKey: "dep_123",
		Reference:      "ref_123",
	}

	mockService.On("Deposit", mock.Anything, depositReq).Return(nil, service.ErrAmountTooSmall)

	body, _ := json.Marshal(depositReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/deposit", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Amount too small")
	mockService.AssertExpectations(t)
}

func TestDeposit_MissingIdempotencyKey(t *testing.T) {
	router, _ := setupTest()

	depositReq := models.DepositRequest{
		UserID:         1,
		Amount:         50000,
		IdempotencyKey: "", // Empty idempotency key
		Reference:      "ref_123",
	}

	// Don't set up mock expectation - Gin validation should reject before service is called
	// Remove this line: mockService.On("Deposit", mock.Anything, depositReq).Return(nil, service.ErrInvalidIdempotencyKey)

	body, _ := json.Marshal(depositReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/deposit", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Fix: Check the response body, not the request body
	assert.Contains(t, w.Body.String(), "IdempotencyKey")
	// Don't assert expectations since service should never be called
	// mockService.AssertExpectations(t) - Remove this
}

// ==============================================
// WITHDRAW TESTS
// ==============================================

func TestWithdraw_Success(t *testing.T) {
	router, mockService := setupTest()

	withdrawReq := models.WithdrawRequest{
		UserID:         1,
		Amount:         20000,
		IdempotencyKey: "wth_123",
		Reference:      "ref_123",
	}

	expectedResp := &models.TransactionResponse{
		TransactionID: 2,
		Status:        "posted",
		Balance:       30000,
		Reference:     "ref_123",
		Message:       "Successfully withdrew ₦200.00",
	}

	mockService.On("Withdraw", mock.Anything, withdrawReq).Return(expectedResp, nil)

	body, _ := json.Marshal(withdrawReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.TransactionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp.TransactionID, resp.TransactionID)
	assert.Equal(t, expectedResp.Balance, resp.Balance)
	mockService.AssertExpectations(t)
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	router, mockService := setupTest()

	withdrawReq := models.WithdrawRequest{
		UserID:         1,
		Amount:         100000,
		IdempotencyKey: "wth_123",
		Reference:      "ref_123",
	}

	mockService.On("Withdraw", mock.Anything, withdrawReq).Return(nil, service.ErrInsufficientBalance)

	body, _ := json.Marshal(withdrawReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient balance")
	mockService.AssertExpectations(t)
}

// ==============================================
// TRANSFER TESTS
// ==============================================

func TestTransfer_Success(t *testing.T) {
	router, mockService := setupTest()

	transferReq := models.TransferRequest{
		FromUserID:     1,
		ToUserID:       2,
		Amount:         30000,
		Fee:            5000,
		IdempotencyKey: "txf_123",
		Reference:      "ref_123",
	}

	expectedResp := &models.TransferResponse{
		TransactionID:    3,
		Status:           "posted",
		SenderBalance:    15000,
		RecipientBalance: 30000,
		Message:          "Successfully transferred ₦300.00",
	}

	mockService.On("Transfer", mock.Anything, transferReq).Return(expectedResp, nil)

	body, _ := json.Marshal(transferReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/transfer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.TransferResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp.TransactionID, resp.TransactionID)
	assert.Equal(t, expectedResp.SenderBalance, resp.SenderBalance)
	assert.Equal(t, expectedResp.RecipientBalance, resp.RecipientBalance)
	mockService.AssertExpectations(t)
}

// ==============================================
// GET BALANCE TESTS
// ==============================================

func TestGetBalance_Success(t *testing.T) {
	router, mockService := setupTest()

	expectedResp := &models.BalanceResponse{
		UserID:     1,
		Balance:    50000,
		BalanceNGN: 500.00,
		Currency:   "NGN",
	}

	mockService.On("GetBalance", mock.Anything, 1).Return(expectedResp, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/balance/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp.Balance, resp.Balance)
	mockService.AssertExpectations(t)
} */