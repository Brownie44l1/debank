package models

import (
	"database/sql"
	"time"
)

// ==============================================
// ACCOUNT MODEL (Wallet/Bank Account)
// ==============================================

// Account represents a wallet or system account
type Account struct {
	ID            int64          `db:"id"`
	AccountNumber sql.NullString `db:"account_number"` // For user accounts
	ExternalID    sql.NullString `db:"external_id"`    // For system accounts
	Name          string         `db:"name"`
	Type          string         `db:"type"`     // 'user', 'system', 'reserve', 'fee'
	Balance       int64          `db:"balance"`  // In kobo
	Currency      string         `db:"currency"` // 'NGN'
	UserID        sql.NullInt32  `db:"user_id"`  // NULL for system accounts
	BankCode      sql.NullString `db:"bank_code"` // For interbank transfers
	BankName      sql.NullString `db:"bank_name"`
	IsActive      bool           `db:"is_active"`
	FrozenAt      sql.NullTime   `db:"frozen_at"`
	FrozenReason  sql.NullString `db:"frozen_reason"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
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

// ==============================================
// RESPONSE DTOs (Keep your existing structure)
// ==============================================

// BalanceResponse for balance queries
type BalanceResponse struct {
	UserID        int     `json:"user_id"`
	AccountNumber string  `json:"account_number"`
	Balance       int64   `json:"balance"`        // In kobo
	BalanceNGN    float64 `json:"balance_ngn"`    // In Naira for convenience
	Currency      string  `json:"currency"`
}

// TransactionResponse returned after transaction operations
type TransactionResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	Balance       int64  `json:"balance"`   // New balance in kobo
	Message       string `json:"message"`
}

// TransferResponse for P2P transfers
type TransferResponse struct {
	TransactionID    int64  `json:"transaction_id"`
	Reference        string `json:"reference"`
	Status           string `json:"status"`
	SenderBalance    int64  `json:"sender_balance"`
	RecipientBalance int64  `json:"recipient_balance,omitempty"` // Optional
	Message          string `json:"message"`
}

// TransactionHistoryResponse for history queries
type TransactionHistoryResponse struct {
	UserID       int                      `json:"user_id"`
	Transactions []TransactionHistoryItem `json:"transactions"`
	Total        int                      `json:"total"`
	Page         int                      `json:"page,omitempty"`
	PerPage      int                      `json:"per_page,omitempty"`
}

// ==============================================
// REQUEST DTOs (Keep your existing structure)
// ==============================================

// DepositRequest for depositing money
type DepositRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// WithdrawRequest for withdrawing money
type WithdrawRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// TransferRequest for P2P transfers (enhanced to support username/phone)
type TransferRequest struct {
	FromUserID     int    `json:"from_user_id" binding:"required"`
	ToIdentifier   string `json:"to_identifier" binding:"required"` // username, phone, or account number
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	Fee            int64  `json:"fee,omitempty"` // Optional fee
	Pin            string `json:"pin" binding:"required,len=4"` // Transaction PIN
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Description    string `json:"description,omitempty"`
}