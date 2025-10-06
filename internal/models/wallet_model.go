package models

import "time"

// ==============================================
// DATABASE MODELS (Maps to DB tables)
// ==============================================

// Account represents a wallet or system account
type Account struct {
	ID         int64     `db:"id"`
	ExternalID *string   `db:"external_id"`
	Name       string    `db:"name"`
	Type       string    `db:"type"`     // 'user', 'system', 'reserve', 'fee'
	Balance    int64     `db:"balance"`  // In kobo
	Currency   string    `db:"currency"` // 'NGN'
	UserID     *int      `db:"user_id"`  // NULL for system accounts
	CreatedAt  time.Time `db:"created_at"`
}

// Transaction represents a logical transaction
type Transaction struct {
	ID             int64     `db:"id"`
	IdempotencyKey string    `db:"idempotency_key"`
	Kind           string    `db:"kind"`      // 'deposit', 'withdrawal', 'p2p'
	Status         string    `db:"status"`    // 'pending', 'posted', 'failed'
	Reference      *string   `db:"reference"` // Optional user reference
	Metadata       []byte    `db:"metadata"`  // JSONB stored as bytes
	CreatedAt      time.Time `db:"created_at"`
}

// Posting represents a debit or credit entry
type Posting struct {
	ID            int64     `db:"id"`
	TransactionID int64     `db:"transaction_id"`
	AccountID     int64     `db:"account_id"`
	Amount        int64     `db:"amount"`   // Positive=credit, Negative=debit
	Currency      string    `db:"currency"` // 'NGN'
	CreatedAt     time.Time `db:"created_at"`
}

// TransactionHistoryItem represents a transaction in user's history
type TransactionHistoryItem struct {
	ID           int64     `db:"id" json:"id"`
	Type         string    `db:"kind" json:"type"`          // 'deposit', 'withdrawal', 'p2p'
	Status       string    `db:"status" json:"status"`      // 'posted', 'failed'
	Reference    *string   `db:"reference" json:"reference,omitempty"`
	Amount       int64     `db:"amount" json:"amount"`      // In kobo
	Direction    string    `db:"direction" json:"direction"` // 'credit' or 'debit'
	Counterparty *string   `db:"counterparty" json:"counterparty,omitempty"` // Who sent/received
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ==============================================
// REQUEST DTOs (API Input)
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

// TransferRequest for P2P transfers
type TransferRequest struct {
	FromUserID     int    `json:"from_user_id" binding:"required"`
	ToUserID       int    `json:"to_user_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	Fee            int64  `json:"fee,omitempty"` // Optional fee
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// ==============================================
// RESPONSE DTOs (API Output)
// ==============================================

// TransactionResponse returned after transaction operations
type TransactionResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Status        string `json:"status"`
	Balance       int64  `json:"balance"`   // New balance in kobo
	Reference     string `json:"reference,omitempty"`
	Message       string `json:"message"`
}

// BalanceResponse for balance queries
type BalanceResponse struct {
	UserID    int     `json:"user_id"`
	Balance   int64   `json:"balance"`    // In kobo
	BalanceNGN float64 `json:"balance_ngn"` // In Naira for convenience
	Currency  string  `json:"currency"`
}

// TransactionHistoryResponse for history queries
type TransactionHistoryResponse struct {
	UserID       int                      `json:"user_id"`
	Transactions []TransactionHistoryItem `json:"transactions"`
	Total        int                      `json:"total"`
	Page         int                      `json:"page,omitempty"`
	PerPage      int                      `json:"per_page,omitempty"`
}

// TransferResponse for P2P transfers
type TransferResponse struct {
	TransactionID    int64  `json:"transaction_id"`
	Status           string `json:"status"`
	SenderBalance    int64  `json:"sender_balance"`
	RecipientBalance int64  `json:"recipient_balance,omitempty"` // Optional
	Message          string `json:"message"`
}