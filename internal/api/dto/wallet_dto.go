package dto

// ==============================================
// WALLET REQUEST DTOs
// ==============================================

// DepositRequest for depositing money
type DepositRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// WithdrawRequest for withdrawing money
type WithdrawRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	Pin            string `json:"pin" binding:"required,len=4,numeric"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Reference      string `json:"reference,omitempty"`
}

// TransferRequest for P2P transfers
type TransferRequest struct {
	ToIdentifier   string `json:"to_identifier" binding:"required"` // @username, phone, or account_number
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	Pin            string `json:"pin" binding:"required,len=4,numeric"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Description    string `json:"description,omitempty"`
}

// ==============================================
// WALLET RESPONSE DTOs
// ==============================================

// BalanceResponse for balance queries
type BalanceResponse struct {
	UserID        int     `json:"user_id"`
	AccountNumber string  `json:"account_number"`
	Balance       int64   `json:"balance"`     // In kobo
	BalanceNGN    float64 `json:"balance_ngn"` // In Naira
	Currency      string  `json:"currency"`
}

// TransactionResponse returned after transaction operations
type TransactionResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	Balance       int64  `json:"balance"` // New balance in kobo
	Message       string `json:"message"`
}

// TransferResponse for P2P transfers
type TransferResponse struct {
	TransactionID    int64  `json:"transaction_id"`
	Reference        string `json:"reference"`
	Status           string `json:"status"`
	SenderBalance    int64  `json:"sender_balance"`
	RecipientBalance int64  `json:"recipient_balance,omitempty"`
	Message          string `json:"message"`
}

// TransactionHistoryResponse for history queries
type TransactionHistoryResponse struct {
	UserID       int                       `json:"user_id"`
	Transactions []TransactionHistoryItem  `json:"transactions"`
	Total        int                       `json:"total"`
	Page         int                       `json:"page,omitempty"`
	PerPage      int                       `json:"per_page,omitempty"`
}

// TransactionHistoryItem represents a single transaction in history
type TransactionHistoryItem struct {
	ID           int64   `json:"id"`
	Reference    string  `json:"reference"`
	Type         string  `json:"type"`   // 'p2p', 'deposit', 'withdrawal'
	Status       string  `json:"status"` // 'posted', 'failed'
	Amount       int64   `json:"amount"` // In kobo
	AmountNGN    float64 `json:"amount_ngn"` // In Naira for convenience
	Description  *string `json:"description,omitempty"`
	Direction    string  `json:"direction"`             // 'credit' or 'debit'
	Counterparty *string `json:"counterparty,omitempty"` // Who sent/received
	CreatedAt    string  `json:"created_at"`            // ISO 8601
}