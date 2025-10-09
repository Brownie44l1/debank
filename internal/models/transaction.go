package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// TRANSACTION MODELS (Database Only)
// ==============================================

// Transaction represents a logical transaction
type Transaction struct {
	ID              int64              `db:"id"`
	IdempotencyKey  string             `db:"idempotency_key"`
	Reference       string             `db:"reference"`
	Kind            string             `db:"kind"`   // 'p2p', 'deposit', 'withdrawal', 'fee', 'interbank', 'refund'
	Status          string             `db:"status"` // 'pending', 'posted', 'failed', 'reversed'
	Amount          int64              `db:"amount"` // In kobo
	Currency        string             `db:"currency"`
	FromAccountID   pgtype.Int8        `db:"from_account_id"`
	ToAccountID     pgtype.Int8        `db:"to_account_id"`
	FromIdentifier  pgtype.Text        `db:"from_identifier"` // username/phone used
	ToIdentifier    pgtype.Text        `db:"to_identifier"`   // username/phone used
	Description     pgtype.Text        `db:"description"`
	Metadata        pgtype.Text        `db:"metadata"` // JSON string
	CreatedAt       time.Time          `db:"created_at"`
	PostedAt        pgtype.Timestamptz `db:"posted_at"`
	FailedAt        pgtype.Timestamptz `db:"failed_at"`
	FailureReason   pgtype.Text        `db:"failure_reason"`
}

// IsPending checks if transaction is still pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// IsPosted checks if transaction is successfully posted
func (t *Transaction) IsPosted() bool {
	return t.Status == TransactionStatusPosted
}

// IsFailed checks if transaction has failed
func (t *Transaction) IsFailed() bool {
	return t.Status == TransactionStatusFailed
}

// Posting represents a debit or credit entry (double-entry bookkeeping)
type Posting struct {
	ID            int64     `db:"id"`
	TransactionID int64     `db:"transaction_id"`
	AccountID     int64     `db:"account_id"`
	Amount        int64     `db:"amount"`   // Positive=credit, Negative=debit
	Currency      string    `db:"currency"` // 'NGN'
	CreatedAt     time.Time `db:"created_at"`
}

// IsCredit checks if this posting is a credit (positive amount)
func (p *Posting) IsCredit() bool {
	return p.Amount > 0
}

// IsDebit checks if this posting is a debit (negative amount)
func (p *Posting) IsDebit() bool {
	return p.Amount < 0
}

// ==============================================
// TRANSACTION CONSTANTS
// ==============================================

// Transaction Kinds
const (
	TransactionKindP2P       = "p2p"
	TransactionKindDeposit   = "deposit"
	TransactionKindWithdraw  = "withdrawal"
	TransactionKindFee       = "fee"
	TransactionKindInterbank = "interbank"
	TransactionKindRefund    = "refund"
)

// Transaction Statuses
const (
	TransactionStatusPending  = "pending"
	TransactionStatusPosted   = "posted"
	TransactionStatusFailed   = "failed"
	TransactionStatusReversed = "reversed"
)

// ==============================================
// TRANSACTION HISTORY (for user-facing display)
// ==============================================

// TransactionHistoryItem represents a transaction in user's history
type TransactionHistoryItem struct {
	ID           int64      `db:"id" json:"id"`
	Reference    string     `db:"reference" json:"reference"`
	Type         string     `db:"kind" json:"type"`             // 'p2p', 'deposit', etc.
	Status       string     `db:"status" json:"status"`         // 'posted', 'failed'
	Amount       int64      `db:"amount" json:"amount"`         // In kobo
	Description  *string    `db:"description" json:"description,omitempty"`
	Direction    string     `json:"direction"`                  // 'credit' or 'debit' (computed)
	Counterparty *string    `json:"counterparty,omitempty"`     // Who sent/received (computed)
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}