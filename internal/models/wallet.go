package models

import "time"

// Account represents a wallet account
type Account struct {
	ID         int64     `db:"id"`
	ExternalID *string   `db:"external_id"`
	Name       string    `db:"name"`
	Type       string    `db:"type"`
	Balance    int64     `db:"balance"`
	Currency   string    `db:"currency"`
	UserID     *int      `db:"user_id"`
	CreatedAt  time.Time `db:"created_at"`
}

// Transaction represents a logical transaction
type Transaction struct {
	ID             int64     `db:"id"`
	IdempotencyKey *string   `db:"idempotency_key"`
	Kind           string    `db:"kind"`
	Status         string    `db:"status"`
	Reference      *string   `db:"reference"`
	Metadata       []byte    `db:"metadata"` // JSONB stored as bytes
	CreatedAt      time.Time `db:"created_at"`
}

// Posting represents a double-entry posting
type Posting struct {
	ID            int64     `db:"id"`
	TransactionID int64     `db:"transaction_id"`
	AccountID     int64     `db:"account_id"`
	Amount        int64     `db:"amount"`
	Currency      string    `db:"currency"`
	CreatedAt     time.Time `db:"created_at"`
}

// DepositRequest represents a deposit operation request
type DepositRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"` // Amount in kobo
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// WithdrawRequest represents a withdrawal operation request
type WithdrawRequest struct {
	UserID         int    `json:"user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"` // Amount in kobo
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// TransactionResponse represents the API response for operations
type TransactionResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Status        string `json:"status"`
	Balance       int64  `json:"balance"` // New balance in kobo
	Message       string `json:"message"`
}