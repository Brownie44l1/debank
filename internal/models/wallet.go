package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// ACCOUNT MODEL (Wallet/Bank Account)
// ==============================================

type Account struct {
	ID            int64            `db:"id"`
	AccountNumber pgtype.Text      `db:"account_number"` // For user accounts
	ExternalID    pgtype.Text      `db:"external_id"`    // For system accounts
	Name          string           `db:"name"`
	Type          string           `db:"type"`     // 'user', 'system', 'reserve', 'fee'
	Balance       int64            `db:"balance"`  // In kobo
	Currency      string           `db:"currency"` // 'NGN'
	UserID        pgtype.Int4      `db:"user_id"`  // NULL for system accounts
	BankCode      pgtype.Text      `db:"bank_code"`
	BankName      pgtype.Text      `db:"bank_name"`
	IsActive      bool             `db:"is_active"`
	FrozenAt      pgtype.Timestamp `db:"frozen_at"`
	FrozenReason  pgtype.Text      `db:"frozen_reason"`
	CreatedAt     time.Time        `db:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at"`
}

func (a *Account) IsUserAccount() bool {
	return a.Type == AccountTypeUser
}

func (a *Account) IsSystemAccount() bool {
	return a.Type == AccountTypeSystem || a.Type == AccountTypeReserve || a.Type == AccountTypeFee
}

func (a *Account) IsFrozen() bool {
	return a.FrozenAt.Valid
}

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
// RESPONSE DTOs
// ==============================================

type BalanceResponse struct {
	UserID        int     `json:"user_id"`
	AccountNumber string  `json:"account_number"`
	Balance       int64   `json:"balance"`     // In kobo
	BalanceNGN    float64 `json:"balance_ngn"` // In Naira for convenience
	Currency      string  `json:"currency"`
}

type TransactionResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	Balance       int64  `json:"balance"` // New balance in kobo
	Message       string `json:"message"`
}

type TransferResponse struct {
	TransactionID    int64  `json:"transaction_id"`
	Reference        string `json:"reference"`
	Status           string `json:"status"`
	SenderBalance    int64  `json:"sender_balance"`
	RecipientBalance int64  `json:"recipient_balance,omitempty"`
	Message          string `json:"message"`
}

type TransactionHistoryResponse struct {
	UserID       int                      `json:"user_id"`
	Transactions []TransactionHistoryItem `json:"transactions"`
	Total        int                      `json:"total"`
	Page         int                      `json:"page,omitempty"`
	PerPage      int                      `json:"per_page,omitempty"`
}

// ==============================================
// REQUEST DTOs
// ==============================================

type DepositRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

type WithdrawRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

type TransferRequest struct {
	FromUserID     int    `json:"from_user_id" binding:"required"`
	ToIdentifier   string `json:"to_identifier" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	Fee            int64  `json:"fee,omitempty"`
	Pin            string `json:"pin" binding:"required,len=4"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Description    string `json:"description,omitempty"`
}