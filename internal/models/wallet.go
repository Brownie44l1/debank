package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// ACCOUNT MODEL (Database Only)
// ==============================================

// Account represents a wallet or system account
type Account struct {
	ID            int64              `db:"id"`
	AccountNumber pgtype.Text        `db:"account_number"` // For user accounts
	ExternalID    pgtype.Text        `db:"external_id"`    // For system accounts
	Name          string             `db:"name"`
	Type          string             `db:"type"`     // 'user', 'system', 'reserve', 'fee'
	Balance       int64              `db:"balance"`  // In kobo
	Currency      string             `db:"currency"` // 'NGN'
	UserID        pgtype.Int4        `db:"user_id"`  // NULL for system accounts
	BankCode      pgtype.Text        `db:"bank_code"` // For interbank transfers
	BankName      pgtype.Text        `db:"bank_name"`
	IsActive      bool               `db:"is_active"`
	FrozenAt      pgtype.Timestamptz `db:"frozen_at"`
	FrozenReason  pgtype.Text        `db:"frozen_reason"`
	CreatedAt     time.Time          `db:"created_at"`
	UpdatedAt     time.Time          `db:"updated_at"`
}

// IsUserAccount checks if this is a user wallet account
func (a *Account) IsUserAccount() bool {
	return a.Type == AccountTypeUser
}

// IsSystemAccount checks if this is a system account
func (a *Account) IsSystemAccount() bool {
	return a.Type == AccountTypeSystem || a.Type == AccountTypeReserve || a.Type == AccountTypeFee
}

// IsFrozen checks if account is currently frozen
func (a *Account) IsFrozen() bool {
	return a.FrozenAt.Valid
}

// GetBalanceNGN returns balance in Naira (kobo / 100)
func (a *Account) GetBalanceNGN() float64 {
	return float64(a.Balance) / 100.0
}

// ==============================================
// ACCOUNT TYPE CONSTANTS
// ==============================================
const (
	AccountTypeUser    = "user"
	AccountTypeSystem  = "system"
	AccountTypeReserve = "reserve"
	AccountTypeFee     = "fee"
)
 type TransactionPIN struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Description    string `json:"description,omitempty"`
}