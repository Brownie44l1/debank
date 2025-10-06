package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==============================================
// MOCK REPOSITORY
// ==============================================

type MockWalletRepository struct {
	BeginTxFunc                     func(ctx context.Context) (pgx.Tx, error)
	GetAccountByUserIDFunc          func(ctx context.Context, userID int) (*models.Account, error)
	GetSystemAccountFunc            func(ctx context.Context, externalID string) (*models.Account, error)
	GetTransactionByIdempotencyKeyFunc func(ctx context.Context, key string) (*models.Transaction, error)
	CreateTransactionFunc           func(ctx context.Context, tx pgx.Tx, txn *models.Transaction) error
	CreatePostingFunc               func(ctx context.Context, tx pgx.Tx, posting *models.Posting) error
	GetTransactionHistoryFunc       func(ctx context.Context, userID int, limit, offset int) ([]models.TransactionHistoryItem, error)
	CountTransactionHistoryFunc     func(ctx context.Context, userID int) (int, error)
}

func (m *MockWalletRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc(ctx)
	}
	return &MockTx{}, nil
}

func (m *MockWalletRepository) GetAccountByUserID(ctx context.Context, userID int) (*models.Account, error) {
	if m.GetAccountByUserIDFunc != nil {
		return m.GetAccountByUserIDFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWalletRepository) GetSystemAccount(ctx context.Context, externalID string) (*models.Account, error) {
	if m.GetSystemAccountFunc != nil {
		return m.GetSystemAccountFunc(ctx, externalID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWalletRepository) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error) {
	if m.GetTransactionByIdempotencyKeyFunc != nil {
		return m.GetTransactionByIdempotencyKeyFunc(ctx, key)
	}
	return nil, errors.New("no rows found")
}

func (m *MockWalletRepository) CreateTransaction(ctx context.Context, tx pgx.Tx, txn *models.Transaction) error {
	if m.CreateTransactionFunc != nil {
		return m.CreateTransactionFunc(ctx, tx, txn)
	}
	txn.ID = 12345 // Simulate auto-increment
	return nil
}

func (m *MockWalletRepository) CreatePosting(ctx context.Context, tx pgx.Tx, posting *models.Posting) error {
	if m.CreatePostingFunc != nil {
		return m.CreatePostingFunc(ctx, tx, posting)
	}
	return nil
}

func (m *MockWalletRepository) GetTransactionHistory(ctx context.Context, userID int, limit, offset int) ([]models.TransactionHistoryItem, error) {
	if m.GetTransactionHistoryFunc != nil {
		return m.GetTransactionHistoryFunc(ctx, userID, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWalletRepository) CountTransactionHistory(ctx context.Context, userID int) (int, error) {
	if m.CountTransactionHistoryFunc != nil {
		return m.CountTransactionHistoryFunc(ctx, userID)
	}
	return 0, errors.New("not implemented")
}

// Mock transaction
type MockTx struct {
	CommitFunc   func(ctx context.Context) error
	RollbackFunc func(ctx context.Context) error
}

func (m *MockTx) Commit(ctx context.Context) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx)
	}
	return nil
}

func (m *MockTx) Rollback(ctx context.Context) error {
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx)
	}
	return nil
}

// Implement other pgx.Tx methods as no-ops
func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *MockTx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}
func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *MockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *MockTx) Conn() *pgx.Conn { return nil }

// ==============================================
// DEPOSIT TESTS
// ==============================================

func TestDeposit_Success(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	userID := 1
	initialBalance := int64(50000) // ₦500
	depositAmount := int64(100000) // ₦1000
	finalBalance := initialBalance + depositAmount

	// Mock implementations
	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found") // Not a duplicate
	}

	callCount := 0
	repo.GetAccountByUserIDFunc = func(ctx context.Context, uid int) (*models.Account, error) {
		callCount++
		balance := initialBalance
		if callCount > 1 { // After transaction
			balance = finalBalance
		}
		return &models.Account{
			ID:       100,
			UserID:   &uid,
			Balance:  balance,
			Currency: "NGN",
		}, nil
	}

	repo.GetSystemAccountFunc = func(ctx context.Context, externalID string) (*models.Account, error) {
		return &models.Account{
			ID:       999,
			Type:     "system",
			Balance:  1000000000,
			Currency: "NGN",
		}, nil
	}

	req := models.DepositRequest{
		UserID:         userID,
		Amount:         depositAmount,
		IdempotencyKey: "dep_123",
		Reference:      "test_deposit",
	}

	resp, err := service.Deposit(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "posted", resp.Status)
	assert.Equal(t, finalBalance, resp.Balance)
	assert.Equal(t, "test_deposit", resp.Reference)
	assert.Contains(t, resp.Message, "Successfully deposited")
}

func TestDeposit_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	tests := []struct {
		name    string
		req     models.DepositRequest
		wantErr error
	}{
		{
			name: "missing idempotency key",
			req: models.DepositRequest{
				UserID: 1,
				Amount: 100000,
			},
			wantErr: ErrInvalidIdempotencyKey,
		},
		{
			name: "zero amount",
			req: models.DepositRequest{
				UserID:         1,
				Amount:         0,
				IdempotencyKey: "dep_zero",
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "negative amount",
			req: models.DepositRequest{
				UserID:         1,
				Amount:         -1000,
				IdempotencyKey: "dep_neg",
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "amount below minimum",
			req: models.DepositRequest{
				UserID:         1,
				Amount:         5000, // ₦50, below ₦100 minimum
				IdempotencyKey: "dep_low",
			},
			wantErr: ErrAmountTooSmall,
		},
		{
			name: "amount exceeds maximum",
			req: models.DepositRequest{
				UserID:         1,
				Amount:         200000000, // ₦2M, above ₦1M max
				IdempotencyKey: "dep_high",
			},
			wantErr: ErrAmountTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Deposit(ctx, tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDeposit_Idempotency(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	existingTxnID := int64(9999)
	currentBalance := int64(500000)

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return &models.Transaction{
			ID:             existingTxnID,
			IdempotencyKey: key,
			Status:         "posted",
		}, nil
	}

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return &models.Account{
			ID:       100,
			UserID:   &userID,
			Balance:  currentBalance,
			Currency: "NGN",
		}, nil
	}

	req := models.DepositRequest{
		UserID:         1,
		Amount:         100000,
		IdempotencyKey: "dep_duplicate",
		Reference:      "original_ref",
	}

	resp, err := service.Deposit(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, existingTxnID, resp.TransactionID)
	assert.Equal(t, "posted", resp.Status)
	assert.Equal(t, currentBalance, resp.Balance)
	assert.Contains(t, resp.Message, "already processed")
}

func TestDeposit_AccountNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return nil, errors.New("account not found")
	}

	req := models.DepositRequest{
		UserID:         999,
		Amount:         100000,
		IdempotencyKey: "dep_nouser",
	}

	_, err := service.Deposit(ctx, req)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

// ==============================================
// WITHDRAW TESTS
// ==============================================

func TestWithdraw_Success(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	userID := 1
	initialBalance := int64(500000)  // ₦5000
	withdrawAmount := int64(100000)  // ₦1000
	finalBalance := initialBalance - withdrawAmount

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	callCount := 0
	repo.GetAccountByUserIDFunc = func(ctx context.Context, uid int) (*models.Account, error) {
		callCount++
		balance := initialBalance
		if callCount > 2 { // After balance check and transaction
			balance = finalBalance
		}
		return &models.Account{
			ID:       100,
			UserID:   &uid,
			Balance:  balance,
			Currency: "NGN",
		}, nil
	}

	repo.GetSystemAccountFunc = func(ctx context.Context, externalID string) (*models.Account, error) {
		return &models.Account{
			ID:       999,
			Type:     "system",
			Balance:  1000000000,
			Currency: "NGN",
		}, nil
	}

	req := models.WithdrawRequest{
		UserID:         userID,
		Amount:         withdrawAmount,
		IdempotencyKey: "wd_123",
		Reference:      "test_withdraw",
	}

	resp, err := service.Withdraw(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "posted", resp.Status)
	assert.Equal(t, finalBalance, resp.Balance)
	assert.Contains(t, resp.Message, "Successfully withdrew")
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return &models.Account{
			ID:       100,
			UserID:   &userID,
			Balance:  50000, // ₦500
			Currency: "NGN",
		}, nil
	}

	req := models.WithdrawRequest{
		UserID:         1,
		Amount:         100000, // ₦1000 - more than balance
		IdempotencyKey: "wd_insufficient",
	}

	_, err := service.Withdraw(ctx, req)
	assert.ErrorIs(t, err, ErrInsufficientBalance)
}

// ==============================================
// TRANSFER TESTS
// ==============================================

func TestTransfer_Success(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	senderID := 1
	recipientID := 2
	senderInitialBalance := int64(500000) // ₦5000
	recipientInitialBalance := int64(100000) // ₦1000
	transferAmount := int64(100000) // ₦1000
	fee := int64(5000) // ₦50

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	accountCallCount := 0
	repo.GetAccountByUserIDFunc = func(ctx context.Context, uid int) (*models.Account, error) {
		accountCallCount++
		
		if uid == senderID {
			balance := senderInitialBalance
			if accountCallCount > 3 { // After transaction
				balance = senderInitialBalance - transferAmount - fee
			}
			return &models.Account{
				ID:       100,
				UserID:   &uid,
				Balance:  balance,
				Currency: "NGN",
			}, nil
		}
		
		if uid == recipientID {
			balance := recipientInitialBalance
			if accountCallCount > 3 { // After transaction
				balance = recipientInitialBalance + transferAmount
			}
			return &models.Account{
				ID:       200,
				UserID:   &uid,
				Balance:  balance,
				Currency: "NGN",
			}, nil
		}
		
		return nil, errors.New("account not found")
	}

	repo.GetSystemAccountFunc = func(ctx context.Context, externalID string) (*models.Account, error) {
		return &models.Account{
			ID:       998,
			Type:     "system",
			Balance:  0,
			Currency: "NGN",
		}, nil
	}

	req := models.TransferRequest{
		FromUserID:     senderID,
		ToUserID:       recipientID,
		Amount:         transferAmount,
		Fee:            fee,
		IdempotencyKey: "txf_123",
		Reference:      "test_transfer",
	}

	resp, err := service.Transfer(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "posted", resp.Status)
	assert.Equal(t, senderInitialBalance-transferAmount-fee, resp.SenderBalance)
	assert.Equal(t, recipientInitialBalance+transferAmount, resp.RecipientBalance)
	assert.Contains(t, resp.Message, "Successfully transferred")
}

func TestTransfer_SameAccount(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	req := models.TransferRequest{
		FromUserID:     1,
		ToUserID:       1, // Same as FromUserID
		Amount:         100000,
		IdempotencyKey: "txf_same",
	}

	_, err := service.Transfer(ctx, req)
	assert.ErrorIs(t, err, ErrSameAccount)
}

func TestTransfer_InsufficientBalanceWithFee(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return &models.Account{
			ID:       100,
			UserID:   &userID,
			Balance:  100000, // ₦1000
			Currency: "NGN",
		}, nil
	}

	req := models.TransferRequest{
		FromUserID:     1,
		ToUserID:       2,
		Amount:         95000,  // ₦950
		Fee:            10000,  // ₦100
		IdempotencyKey: "txf_insufficient",
	}

	// Total needed: 95000 + 10000 = 105000, but balance is only 100000
	_, err := service.Transfer(ctx, req)
	assert.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestTransfer_NegativeFee(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	req := models.TransferRequest{
		FromUserID:     1,
		ToUserID:       2,
		Amount:         100000,
		Fee:            -1000, // Negative fee
		IdempotencyKey: "txf_negfee",
	}

	_, err := service.Transfer(ctx, req)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

// ==============================================
// GET BALANCE TESTS
// ==============================================

func TestGetBalance_Success(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	userID := 1
	balance := int64(123456) // ₦1234.56

	repo.GetAccountByUserIDFunc = func(ctx context.Context, uid int) (*models.Account, error) {
		return &models.Account{
			ID:       100,
			UserID:   &uid,
			Balance:  balance,
			Currency: "NGN",
		}, nil
	}

	resp, err := service.GetBalance(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, balance, resp.Balance)
	assert.Equal(t, 1234.56, resp.BalanceNGN)
	assert.Equal(t, "NGN", resp.Currency)
}

func TestGetBalance_AccountNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return nil, errors.New("account not found")
	}

	_, err := service.GetBalance(ctx, 999)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

// ==============================================
// TRANSACTION HISTORY TESTS
// ==============================================

func TestGetTransactionHistory_Success(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	userID := 1
	now := time.Now()

	repo.GetTransactionHistoryFunc = func(ctx context.Context, uid int, limit, offset int) ([]models.TransactionHistoryItem, error) {
		return []models.TransactionHistoryItem{
			{
				ID:        1,
				Type:      "deposit",
				Status:    "posted",
				Amount:    100000,
				Direction: "credit",
				CreatedAt: now,
			},
			{
				ID:        2,
				Type:      "withdrawal",
				Status:    "posted",
				Amount:    50000,
				Direction: "debit",
				CreatedAt: now.Add(-1 * time.Hour),
			},
		}, nil
	}

	repo.CountTransactionHistoryFunc = func(ctx context.Context, uid int) (int, error) {
		return 2, nil
	}

	resp, err := service.GetTransactionHistory(ctx, userID, 1, 20)

	require.NoError(t, err)
	assert.Equal(t, userID, resp.UserID)
	assert.Len(t, resp.Transactions, 2)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PerPage)
}

func TestGetTransactionHistory_Pagination(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	tests := []struct {
		name           string
		inputPage      int
		inputPerPage   int
		expectedPage   int
		expectedPerPage int
		expectedOffset int
	}{
		{
			name:           "default values for invalid page",
			inputPage:      0,
			inputPerPage:   0,
			expectedPage:   1,
			expectedPerPage: 20,
			expectedOffset: 0,
		},
		{
			name:           "page 2 with 10 per page",
			inputPage:      2,
			inputPerPage:   10,
			expectedPage:   2,
			expectedPerPage: 10,
			expectedOffset: 10,
		},
		{
			name:           "exceeds max per page",
			inputPage:      1,
			inputPerPage:   150,
			expectedPage:   1,
			expectedPerPage: 20,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedLimit, capturedOffset int
			
			repo.GetTransactionHistoryFunc = func(ctx context.Context, uid int, limit, offset int) ([]models.TransactionHistoryItem, error) {
				capturedLimit = limit
				capturedOffset = offset
				return []models.TransactionHistoryItem{}, nil
			}

			repo.CountTransactionHistoryFunc = func(ctx context.Context, uid int) (int, error) {
				return 0, nil
			}

			resp, err := service.GetTransactionHistory(ctx, 1, tt.inputPage, tt.inputPerPage)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPage, resp.Page)
			assert.Equal(t, tt.expectedPerPage, resp.PerPage)
			assert.Equal(t, tt.expectedPerPage, capturedLimit)
			assert.Equal(t, tt.expectedOffset, capturedOffset)
		})
	}
}

// ==============================================
// EDGE CASES & ERROR SCENARIOS
// ==============================================

func TestTransactionCommitFailure(t *testing.T) {
	ctx := context.Background()
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	repo.GetTransactionByIdempotencyKeyFunc = func(ctx context.Context, key string) (*models.Transaction, error) {
		return nil, errors.New("no rows found")
	}

	repo.GetAccountByUserIDFunc = func(ctx context.Context, userID int) (*models.Account, error) {
		return &models.Account{
			ID:       100,
			UserID:   &userID,
			Balance:  500000,
			Currency: "NGN",
		}, nil
	}

	repo.GetSystemAccountFunc = func(ctx context.Context, externalID string) (*models.Account, error) {
		return &models.Account{
			ID:       999,
			Type:     "system",
			Balance:  1000000000,
			Currency: "NGN",
		}, nil
	}

	// Mock transaction that fails on commit
	repo.BeginTxFunc = func(ctx context.Context) (pgx.Tx, error) {
		return &MockTx{
			CommitFunc: func(ctx context.Context) error {
				return errors.New("commit failed")
			},
		}, nil
	}

	req := models.DepositRequest{
		UserID:         1,
		Amount:         100000,
		IdempotencyKey: "dep_commitfail",
	}

	_, err := service.Deposit(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit")
}

func TestValidateAmounts_BoundaryValues(t *testing.T) {
	repo := &MockWalletRepository{}
	service := NewWalletService(repo)

	tests := []struct {
		name      string
		amount    int64
		shouldErr bool
	}{
		{"exactly minimum", MinDepositAmount, false},
		{"one below minimum", MinDepositAmount - 1, true},
		{"exactly maximum", MaxTransactionAmount, false},
		{"one above maximum", MaxTransactionAmount + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateDepositAmount(tt.amount)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}