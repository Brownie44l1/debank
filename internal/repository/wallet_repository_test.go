package repository

import (
	"context"
	"testing"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: These are integration tests that require a real database
// To run them, you need:
// 1. A running PostgreSQL database
// 2. Database migrations applied
// 3. Set DATABASE_URL environment variable

// Helper function to get test database connection
func getTestDB(t *testing.T) *pgxpool.Pool {
	// This would connect to your test database
	// For now, we'll skip if no database is available
	t.Skip("Integration tests require database connection")
	return nil
}

// ==============================================
// ACCOUNT QUERY TESTS
// ==============================================

func TestGetAccountByUserID_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Assuming user 1 exists in test database
	account, err := repo.GetAccountByUserID(ctx, 1)

	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, 1, *account.UserID)
	assert.Equal(t, "NGN", account.Currency)
}

func TestGetAccountByUserID_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	account, err := repo.GetAccountByUserID(ctx, 99999)

	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestGetSystemAccount_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	account, err := repo.GetSystemAccount(ctx, "sys_reserve")

	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, "sys_reserve", *account.ExternalID)
	assert.Equal(t, "system", account.Type)
}

func TestGetSystemAccount_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	account, err := repo.GetSystemAccount(ctx, "non_existent")

	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

// ==============================================
// TRANSACTION QUERY TESTS
// ==============================================

func TestGetTransactionByIdempotencyKey_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	txn, err := repo.GetTransactionByIdempotencyKey(ctx, "non-existent-key")

	assert.Error(t, err)
	assert.Nil(t, txn)
	assert.ErrorIs(t, err, ErrNoRows)
}

// ==============================================
// FULL TRANSACTION TESTS
// ==============================================

func TestCreateTransaction_FullFlow(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Begin transaction
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Create transaction
	txn := &models.Transaction{
		IdempotencyKey: "test-deposit-123",
		Kind:           "deposit",
		Status:         "posted",
		Reference:      strPtr("Test reference"),
	}

	err = repo.CreateTransaction(ctx, tx, txn)
	require.NoError(t, err)
	assert.NotZero(t, txn.ID)

	// Create postings
	posting1 := &models.Posting{
		TransactionID: txn.ID,
		AccountID:     1, // System reserve
		Amount:        -100000,
		Currency:      "NGN",
	}

	err = repo.CreatePosting(ctx, tx, posting1)
	require.NoError(t, err)
	assert.NotZero(t, posting1.ID)

	posting2 := &models.Posting{
		TransactionID: txn.ID,
		AccountID:     2, // User account
		Amount:        100000,
		Currency:      "NGN",
	}

	err = repo.CreatePosting(ctx, tx, posting2)
	require.NoError(t, err)
	assert.NotZero(t, posting2.ID)

	// Commit
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify transaction was created
	savedTxn, err := repo.GetTransactionByIdempotencyKey(ctx, "test-deposit-123")
	require.NoError(t, err)
	assert.Equal(t, txn.ID, savedTxn.ID)
	assert.Equal(t, "deposit", savedTxn.Kind)
}

// ==============================================
// TRANSACTION HISTORY TESTS
// ==============================================

func TestGetTransactionHistory_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Assuming user 1 has transactions
	history, err := repo.GetTransactionHistory(ctx, 1, 10, 0)

	require.NoError(t, err)
	assert.NotNil(t, history)
	// Can't assert exact length without knowing test data
}

func TestGetTransactionHistory_Pagination(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Get first page
	page1, err := repo.GetTransactionHistory(ctx, 1, 5, 0)
	require.NoError(t, err)

	// Get second page
	page2, err := repo.GetTransactionHistory(ctx, 1, 5, 5)
	require.NoError(t, err)

	// Verify they're different (if there are enough transactions)
	if len(page1) > 0 && len(page2) > 0 {
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	}
}

func TestCountTransactionHistory_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	count, err := repo.CountTransactionHistory(ctx, 1)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}

func TestCountTransactionHistory_NonExistentUser(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	count, err := repo.CountTransactionHistory(ctx, 99999)

	// Should return error because account doesn't exist
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	assert.Equal(t, 0, count)
}

// ==============================================
// GET POSTINGS TESTS
// ==============================================

func TestGetPostingsByTransactionID_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Assuming transaction 1 exists
	postings, err := repo.GetPostingsByTransactionID(ctx, 1)

	require.NoError(t, err)
	assert.NotNil(t, postings)
	// A valid transaction should have at least 2 postings (double-entry)
	if len(postings) > 0 {
		assert.GreaterOrEqual(t, len(postings), 2)
	}
}

func TestGetPostingsByTransactionID_NonExistent(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	postings, err := repo.GetPostingsByTransactionID(ctx, 99999)

	require.NoError(t, err)
	assert.Empty(t, postings)
}

// ==============================================
// HELPER FUNCTIONS
// ==============================================

func strPtr(s string) *string {
	return &s
}

// ==============================================
// UNIT TESTS (No Database Required)
// ==============================================

func TestNewWalletRepository(t *testing.T) {
	// This test doesn't require a database
	repo := NewWalletRepository(nil)
	assert.NotNil(t, repo)
}

func TestErrorConstants(t *testing.T) {
	// Verify error messages
	assert.Equal(t, "account not found", ErrAccountNotFound.Error())
	assert.Equal(t, "no rows found", ErrNoRows.Error())
}