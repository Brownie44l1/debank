package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ==============================================
// ERRORS
// ==============================================

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrNoRows          = errors.New("no rows found")
)

// ==============================================
// REPOSITORY (Data Access ONLY)
// ==============================================

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

// ==============================================
// TRANSACTION MANAGEMENT
// ==============================================

// BeginTx starts a new database transaction
func (r *WalletRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// ==============================================
// ACCOUNT QUERIES (WITHOUT LOCKING - for reads)
// ==============================================

// GetAccountByID retrieves an account by its ID (no lock)
func (r *WalletRepository) GetAccountByID(ctx context.Context, accountID int64) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE id = $1
	`

	var acc models.Account
	err := r.db.QueryRow(ctx, query, accountID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &acc, nil
}

// GetAccountByUserID retrieves a user's wallet account (no lock)
func (r *WalletRepository) GetAccountByUserID(ctx context.Context, userID int) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE user_id = $1
	`

	var acc models.Account
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &acc, nil
}

// GetSystemAccount retrieves a system account by external_id (no lock)
func (r *WalletRepository) GetSystemAccount(ctx context.Context, externalID string) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE external_id = $1 AND type = 'system'
	`

	var acc models.Account
	err := r.db.QueryRow(ctx, query, externalID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get system account: %w", err)
	}

	return &acc, nil
}

// ==============================================
// ACCOUNT QUERIES (WITH LOCKING - for updates)
// ==============================================

// GetAccountByUserIDForUpdate retrieves and locks a user's account for update
// This prevents concurrent modifications to the same account
func (r *WalletRepository) GetAccountByUserIDForUpdate(ctx context.Context, tx pgx.Tx, userID int) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE user_id = $1
		FOR UPDATE
	`

	var acc models.Account
	err := tx.QueryRow(ctx, query, userID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to lock account: %w", err)
	}

	return &acc, nil
}

// GetAccountByIDForUpdate retrieves and locks an account by ID
func (r *WalletRepository) GetAccountByIDForUpdate(ctx context.Context, tx pgx.Tx, accountID int64) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var acc models.Account
	err := tx.QueryRow(ctx, query, accountID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to lock account: %w", err)
	}

	return &acc, nil
}

// GetSystemAccountForUpdate retrieves and locks a system account
func (r *WalletRepository) GetSystemAccountForUpdate(ctx context.Context, tx pgx.Tx, externalID string) (*models.Account, error) {
	query := `
		SELECT id, external_id, name, type, balance, currency, user_id, created_at
		FROM accounts
		WHERE external_id = $1 AND type = 'system'
		FOR UPDATE
	`

	var acc models.Account
	err := tx.QueryRow(ctx, query, externalID).Scan(
		&acc.ID,
		&acc.ExternalID,
		&acc.Name,
		&acc.Type,
		&acc.Balance,
		&acc.Currency,
		&acc.UserID,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to lock system account: %w", err)
	}

	return &acc, nil
}

// ==============================================
// TRANSACTION QUERIES
// ==============================================

// GetTransactionByID retrieves a transaction by ID
func (r *WalletRepository) GetTransactionByID(ctx context.Context, txnID int64) (*models.Transaction, error) {
	query := `
		SELECT id, idempotency_key, kind, status, reference, metadata, created_at
		FROM transactions
		WHERE id = $1
	`

	var txn models.Transaction
	err := r.db.QueryRow(ctx, query, txnID).Scan(
		&txn.ID,
		&txn.IdempotencyKey,
		&txn.Kind,
		&txn.Status,
		&txn.Reference,
		&txn.Metadata,
		&txn.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &txn, nil
}

// GetTransactionByIdempotencyKey checks if idempotency key exists
func (r *WalletRepository) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error) {
	query := `
		SELECT id, idempotency_key, kind, status, reference, metadata, created_at
		FROM transactions
		WHERE idempotency_key = $1
	`

	var txn models.Transaction
	err := r.db.QueryRow(ctx, query, key).Scan(
		&txn.ID,
		&txn.IdempotencyKey,
		&txn.Kind,
		&txn.Status,
		&txn.Reference,
		&txn.Metadata,
		&txn.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows // Not an error - key doesn't exist
		}
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return &txn, nil
}

// CreateTransaction creates a new transaction record within a transaction
func (r *WalletRepository) CreateTransaction(ctx context.Context, tx pgx.Tx, txn *models.Transaction) error {
	query := `
		INSERT INTO transactions (idempotency_key, kind, status, reference, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := tx.QueryRow(ctx, query,
		txn.IdempotencyKey,
		txn.Kind,
		txn.Status,
		txn.Reference,
		txn.Metadata,
	).Scan(&txn.ID, &txn.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// ==============================================
// POSTING QUERIES
// ==============================================

// CreatePosting creates a new posting (debit or credit) within a transaction
func (r *WalletRepository) CreatePosting(ctx context.Context, tx pgx.Tx, posting *models.Posting) error {
	query := `
		INSERT INTO postings (transaction_id, account_id, amount, currency)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := tx.QueryRow(ctx, query,
		posting.TransactionID,
		posting.AccountID,
		posting.Amount,
		posting.Currency,
	).Scan(&posting.ID, &posting.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create posting: %w", err)
	}

	return nil
}

// GetPostingsByTransactionID retrieves all postings for a transaction
func (r *WalletRepository) GetPostingsByTransactionID(ctx context.Context, txnID int64) ([]models.Posting, error) {
	query := `
		SELECT id, transaction_id, account_id, amount, currency, created_at
		FROM postings
		WHERE transaction_id = $1
		ORDER BY id
	`

	rows, err := r.db.Query(ctx, query, txnID)
	if err != nil {
		return nil, fmt.Errorf("failed to query postings: %w", err)
	}
	defer rows.Close()

	var postings []models.Posting
	for rows.Next() {
		var p models.Posting
		err := rows.Scan(&p.ID, &p.TransactionID, &p.AccountID, &p.Amount, &p.Currency, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan posting: %w", err)
		}
		postings = append(postings, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating postings: %w", err)
	}

	return postings, nil
}

// ==============================================
// TRANSACTION HISTORY
// ==============================================

// GetTransactionHistory retrieves transaction history for a user with pagination
func (r *WalletRepository) GetTransactionHistory(ctx context.Context, userID int, limit, offset int) ([]models.TransactionHistoryItem, error) {
	// First, get the user's account ID
	account, err := r.GetAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			t.id,
			t.kind,
			t.status,
			t.reference,
			p.amount,
			CASE 
				WHEN p.amount > 0 THEN 'credit'
				ELSE 'debit'
			END as direction,
			other_acc.name as counterparty,
			t.created_at
		FROM postings p
		JOIN transactions t ON t.id = p.transaction_id
		LEFT JOIN postings other_p ON other_p.transaction_id = t.id 
			AND other_p.account_id != p.account_id
			AND SIGN(other_p.amount) != SIGN(p.amount)
		LEFT JOIN accounts other_acc ON other_acc.id = other_p.account_id
		WHERE p.account_id = $1
			AND t.status = 'posted'
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, account.ID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction history: %w", err)
	}
	defer rows.Close()

	var history []models.TransactionHistoryItem
	for rows.Next() {
		var item models.TransactionHistoryItem
		err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Status,
			&item.Reference,
			&item.Amount,
			&item.Direction,
			&item.Counterparty,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction history: %w", err)
		}
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction history: %w", err)
	}

	return history, nil
}

// CountTransactionHistory returns total number of transactions for a user
func (r *WalletRepository) CountTransactionHistory(ctx context.Context, userID int) (int, error) {
	account, err := r.GetAccountByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	query := `
		SELECT COUNT(DISTINCT t.id)
		FROM postings p
		JOIN transactions t ON t.id = p.transaction_id
		WHERE p.account_id = $1
			AND t.status = 'posted'
	`

	var count int
	err = r.db.QueryRow(ctx, query, account.ID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}