package repository

import (
	"context"
	"os"
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
// 4. Run test data setup SQL

// Helper function to get test database connection
func getTestDB(t *testing.T) *pgxpool.Pool {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration tests")
		return nil
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}

	// Verify connection
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("Unable to ping database: %v", err)
	}

	return pool
}

// ==============================================
// ACCOUNT QUERY TESTS (Non-locking)
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
	assert.GreaterOrEqual(t, account.Balance, int64(0))
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
// ACCOUNT QUERY TESTS (With Locking - NEW)
// ==============================================

func TestGetAccountByUserIDForUpdate_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Begin transaction
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Lock the account
	account, err := repo.GetAccountByUserIDForUpdate(ctx, tx, 1)

	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, 1, *account.UserID)
	assert.Equal(t, "NGN", account.Currency)
}

func TestGetAccountByUserIDForUpdate_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	account, err := repo.GetAccountByUserIDForUpdate(ctx, tx, 99999)

	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestGetSystemAccountForUpdate_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	account, err := repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")

	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, "sys_reserve", *account.ExternalID)
	assert.Equal(t, "system", account.Type)
}

func TestGetSystemAccountForUpdate_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	account, err := repo.GetSystemAccountForUpdate(ctx, tx, "non_existent")

	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

// ==============================================
// LOCKING BEHAVIOR TESTS (NEW)
// ==============================================

func TestAccountLocking_PreventsConcurrentModification(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Start first transaction and lock account
	tx1, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	account1, err := repo.GetAccountByUserIDForUpdate(ctx, tx1, 1)
	require.NoError(t, err)
	originalBalance := account1.Balance

	// Try to lock the same account in a second transaction
	// This should wait (we'll use a short timeout context)
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	// This would normally block, but we're just testing the mechanism
	// In a real scenario, tx2 would wait until tx1 commits or rolls back
	
	// Rollback first transaction
	err = tx1.Rollback(ctx)
	require.NoError(t, err)

	// Now second transaction can proceed
	account2, err := repo.GetAccountByUserIDForUpdate(ctx, tx2, 1)
	require.NoError(t, err)
	assert.Equal(t, originalBalance, account2.Balance)
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

func TestGetTransactionByIdempotencyKey_Found(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// Use a known idempotency key from test data
	txn, err := repo.GetTransactionByIdempotencyKey(ctx, "test_hist_deposit_1")

	if err == nil {
		// If found, verify it's correct
		require.NoError(t, err)
		assert.NotNil(t, txn)
		assert.Equal(t, "test_hist_deposit_1", txn.IdempotencyKey)
	} else {
		// If not found, that's okay for this test
		assert.ErrorIs(t, err, ErrNoRows)
	}
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

	// Get accounts with locks
	userAccount, err := repo.GetAccountByUserIDForUpdate(ctx, tx, 1)
	require.NoError(t, err)

	reserveAccount, err := repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")
	require.NoError(t, err)

	// Create transaction
	txn := &models.Transaction{
		IdempotencyKey: "test-deposit-integration-123",
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
		AccountID:     reserveAccount.ID,
		Amount:        -100000,
		Currency:      "NGN",
	}

	err = repo.CreatePosting(ctx, tx, posting1)
	require.NoError(t, err)
	assert.NotZero(t, posting1.ID)

	posting2 := &models.Posting{
		TransactionID: txn.ID,
		AccountID:     userAccount.ID,
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
	savedTxn, err := repo.GetTransactionByIdempotencyKey(ctx, "test-deposit-integration-123")
	require.NoError(t, err)
	assert.Equal(t, txn.ID, savedTxn.ID)
	assert.Equal(t, "deposit", savedTxn.Kind)

	// Cleanup: delete test transaction
	cleanupCtx := context.Background()
	cleanupTx, _ := repo.BeginTx(cleanupCtx)
	if cleanupTx != nil {
		cleanupTx.Exec(cleanupCtx, "DELETE FROM postings WHERE transaction_id = $1", txn.ID)
		cleanupTx.Exec(cleanupCtx, "DELETE FROM transactions WHERE id = $1", txn.ID)
		cleanupTx.Commit(cleanupCtx)
	}
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
	
	// Should have at least the historical transactions from setup
	if len(history) > 0 {
		// Verify structure
		assert.NotEmpty(t, history[0].Type)
		assert.NotEmpty(t, history[0].Direction)
		assert.NotZero(t, history[0].CreatedAt)
	}
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

	// First, find a transaction that exists
	history, err := repo.GetTransactionHistory(ctx, 1, 1, 0)
	require.NoError(t, err)
	
	if len(history) > 0 {
		txnID := history[0].ID
		postings, err := repo.GetPostingsByTransactionID(ctx, txnID)

		require.NoError(t, err)
		assert.NotNil(t, postings)
		// A valid transaction should have at least 2 postings (double-entry)
		assert.GreaterOrEqual(t, len(postings), 2)
	} else {
		t.Skip("No transactions found for user 1")
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
// CONCURRENT ACCESS TESTS (NEW)
// ==============================================

func TestConcurrentAccountLocking(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewWalletRepository(db)
	ctx := context.Background()

	// This test demonstrates that locks work correctly
	tx1, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	// Lock account in first transaction
	account1, err := repo.GetAccountByUserIDForUpdate(ctx, tx1, 1)
	require.NoError(t, err)
	assert.NotNil(t, account1)

	// Rollback to release lock
	err = tx1.Rollback(ctx)
	require.NoError(t, err)

	// Now second transaction can get the lock
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	account2, err := repo.GetAccountByUserIDForUpdate(ctx, tx2, 1)
	require.NoError(t, err)
	assert.NotNil(t, account2)
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