package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientBalance     = errors.New("insufficient balance")
	ErrAccountNotFound         = errors.New("account not found")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
)

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

// GetAccountByUserID retrieves a user's account
func (r *WalletRepository) GetAccountByUserID(ctx context.Context, userID int) (int64, int64, error) {
	query := `SELECT id, balance FROM accounts WHERE user_id = $1`

	var accountID, balance int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&accountID, &balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, fmt.Errorf("failed to get account: %w", err)
	}

	return accountID, balance, nil
}

// GetSystemAccount retrieves a system account by external_id
func (r *WalletRepository) GetSystemAccount(ctx context.Context, externalID string) (int64, error) {
	query := `SELECT id FROM accounts WHERE external_id = $1 AND type = 'system'`

	var accountID int64
	err := r.db.QueryRow(ctx, query, externalID).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrAccountNotFound
		}
		return 0, fmt.Errorf("failed to get system account: %w", err)
	}

	return accountID, nil
}

// CheckIdempotency checks if an idempotency key already exists
// Returns (transactionID, alreadyExists, error)
func (r *WalletRepository) CheckIdempotency(ctx context.Context, idempotencyKey string) (int64, bool, error) {
	query := `SELECT id FROM transactions WHERE idempotency_key = $1`

	var txnID int64
	err := r.db.QueryRow(ctx, query, idempotencyKey).Scan(&txnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil // Key doesn't exist, safe to proceed
		}
		return 0, false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return txnID, true, nil // Key exists, return existing transaction ID
}

// Deposit adds money to a user's account from the reserve
func (r *WalletRepository) Deposit(ctx context.Context, userID int, amount int64, idempotencyKey, reference string) (int64, int64, error) {
	log.Printf("DEBUG: Starting deposit - userID=%d, amount=%d, idempotencyKey=%s", userID, amount, idempotencyKey)

	// Start a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to begin transaction: %v", err)
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Check idempotency
	log.Printf("DEBUG: Checking idempotency key: %s", idempotencyKey)
	var existingTxnID int64
	err = tx.QueryRow(ctx, `SELECT id FROM transactions WHERE idempotency_key = $1`, idempotencyKey).Scan(&existingTxnID)
	// Right before line 92 in your Deposit function
	/* log.Printf("DEBUG: Checking accounts table structure")
	var colExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'accounts' 
			AND column_name = 'user_id'
		)
	`).Scan(&colExists)
	log.Printf("DEBUG: user_id column exists: %v", colExists) */
	if err == nil {
		log.Printf("DEBUG: Idempotency key already exists, returning existing transaction: %d", existingTxnID)
		// Idempotency key already exists - return existing transaction
		var balance int64
		_ = tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE user_id = $1`, userID).Scan(&balance)
		return existingTxnID, balance, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("ERROR: Failed to check idempotency: %v", err)
		return 0, 0, fmt.Errorf("failed to check idempotency: %w", err)
	}
	log.Printf("DEBUG: Idempotency key is new, proceeding...")

	// 2. Get user account ID
	log.Printf("DEBUG: Getting user account for user_id=%d", userID)
	var userAccountID int64
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE user_id = $1`, userID).Scan(&userAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("ERROR: Account not found for user_id=%d", userID)
			return 0, 0, ErrAccountNotFound
		}
		log.Printf("ERROR: Failed to get user account: %v", err)
		return 0, 0, fmt.Errorf("failed to get user account: %w", err)
	}
	log.Printf("DEBUG: Found user account_id=%d", userAccountID)

	// 3. Get reserve account ID
	log.Printf("DEBUG: Getting reserve account")
	var reserveAccountID int64
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE external_id = 'sys_reserve'`).Scan(&reserveAccountID)
	if err != nil {
		log.Printf("ERROR: Failed to get reserve account: %v", err)
		return 0, 0, fmt.Errorf("failed to get reserve account: %w", err)
	}
	log.Printf("DEBUG: Found reserve account_id=%d", reserveAccountID)

	// 4. Create transaction record
	log.Printf("DEBUG: Creating transaction record")
	var txnID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (idempotency_key, kind, status, reference) 
		 VALUES ($1, 'deposit', 'posted', $2) 
		 RETURNING id`,
		idempotencyKey, reference,
	).Scan(&txnID)
	if err != nil {
		log.Printf("ERROR: Failed to create transaction: %v", err)
		return 0, 0, fmt.Errorf("failed to create transaction: %w", err)
	}
	log.Printf("DEBUG: Created transaction_id=%d", txnID)

	// 5. Create postings (double-entry)
	// Note: The database trigger will automatically update account balances

	log.Printf("DEBUG: Creating reserve posting (debit)")
	// Debit reserve account (negative amount)
	_, err = tx.Exec(ctx,
		`INSERT INTO postings (transaction_id, account_id, amount, currency) 
		 VALUES ($1, $2, $3, 'NGN')`,
		txnID, reserveAccountID, -amount,
	)
	if err != nil {
		log.Printf("ERROR: Failed to create reserve posting: %v", err)
		return 0, 0, fmt.Errorf("failed to create reserve posting: %w", err)
	}

	log.Printf("DEBUG: Creating user posting (credit)")
	// Credit user account (positive amount)
	_, err = tx.Exec(ctx,
		`INSERT INTO postings (transaction_id, account_id, amount, currency) 
		 VALUES ($1, $2, $3, 'NGN')`,
		txnID, userAccountID, amount,
	)
	if err != nil {
		log.Printf("ERROR: Failed to create user posting: %v", err)
		return 0, 0, fmt.Errorf("failed to create user posting: %w", err)
	}

	// 6. Get the updated balance (trigger already updated it)
	log.Printf("DEBUG: Getting updated balance")
	var newBalance int64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1`,
		userAccountID,
	).Scan(&newBalance)
	if err != nil {
		log.Printf("ERROR: Failed to get updated balance: %v", err)
		return 0, 0, fmt.Errorf("failed to get updated balance: %w", err)
	}
	log.Printf("DEBUG: New balance=%d", newBalance)

	// Commit transaction
	log.Printf("DEBUG: Committing transaction")
	if err := tx.Commit(ctx); err != nil {
		log.Printf("ERROR: Failed to commit transaction: %v", err)
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("DEBUG: Deposit successful - txnID=%d, newBalance=%d", txnID, newBalance)
	return txnID, newBalance, nil
}

// Withdraw removes money from a user's account to the reserve
func (r *WalletRepository) Withdraw(ctx context.Context, userID int, amount int64, idempotencyKey, reference string) (int64, int64, error) {
	// Start a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Check idempotency
	var existingTxnID int64
	err = tx.QueryRow(ctx, `SELECT id FROM transactions WHERE idempotency_key = $1`, idempotencyKey).Scan(&existingTxnID)
	if err == nil {
		// Idempotency key already exists - return existing transaction
		var balance int64
		_ = tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE user_id = $1`, userID).Scan(&balance)
		return existingTxnID, balance, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// 2. Get user account ID and current balance
	var userAccountID, currentBalance int64
	err = tx.QueryRow(ctx, `SELECT id, balance FROM accounts WHERE user_id = $1`, userID).Scan(&userAccountID, &currentBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, fmt.Errorf("failed to get user account: %w", err)
	}

	// 3. Check sufficient balance
	if currentBalance < amount {
		return 0, currentBalance, ErrInsufficientBalance
	}

	// 4. Get reserve account ID
	var reserveAccountID int64
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE external_id = 'sys_reserve'`).Scan(&reserveAccountID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get reserve account: %w", err)
	}

	// 5. Create transaction record
	var txnID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (idempotency_key, kind, status, reference) 
		 VALUES ($1, 'withdrawal', 'posted', $2) 
		 RETURNING id`,
		idempotencyKey, reference,
	).Scan(&txnID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create transaction: %w", err)
	}

	// 6. Create postings (double-entry)
	// Note: The database trigger will automatically update account balances

	// Debit user account (negative amount)
	_, err = tx.Exec(ctx,
		`INSERT INTO postings (transaction_id, account_id, amount, currency) 
		 VALUES ($1, $2, $3, 'NGN')`,
		txnID, userAccountID, -amount,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create user posting: %w", err)
	}

	// Credit reserve account (positive amount)
	_, err = tx.Exec(ctx,
		`INSERT INTO postings (transaction_id, account_id, amount, currency) 
		 VALUES ($1, $2, $3, 'NGN')`,
		txnID, reserveAccountID, amount,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create reserve posting: %w", err)
	}

	// 7. Get the updated balance (trigger already updated it)
	var newBalance int64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1`,
		userAccountID,
	).Scan(&newBalance)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get updated balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return txnID, newBalance, nil
}

// GetTransactionHistory retrieves all transactions for a user
/* func (r *WalletRepository) GetTransactionHistory(ctx context.Context, userID int) ([]models.TransactionHistoryItem, error) {
	// First, get the user's account ID
	var accountID int64
	err := r.db.QueryRow(ctx, `SELECT id FROM accounts WHERE user_id = $1`, userID).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Get all transactions for this user, ordered by most recent first
	query := `
		SELECT 
			t.id,
			t.kind,
			t.status,
			t.reference,
			t.created_at,
			COALESCE(SUM(p.amount), 0) as amount
		FROM transactions t
		INNER JOIN postings p ON t.id = p.transaction_id
		WHERE p.account_id = $1
		GROUP BY t.id, t.kind, t.status, t.reference, t.created_at
		ORDER BY t.created_at DESC, t.id DESC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.TransactionHistoryItem
	for rows.Next() {
		var txn models.TransactionHistoryItem
		err := rows.Scan(
			&txn.ID,
			&txn.Type,
			&txn.Status,
			&txn.Reference,
			&txn.CreatedAt,
			&txn.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
} */
