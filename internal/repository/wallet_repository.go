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