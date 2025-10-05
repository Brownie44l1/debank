package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/repository"
)

// Business logic constants
const (
	MinDepositAmount     = 10000     // ₦100.00 minimum deposit
	MinWithdrawAmount    = 10000     // ₦100.00 minimum withdrawal
	MaxTransactionAmount = 100000000 // ₦1,000,000.00 maximum per transaction
)

var (
	ErrInvalidAmount         = errors.New("invalid transaction amount")
	ErrAmountTooSmall        = errors.New("amount is below minimum")
	ErrAmountTooLarge        = errors.New("amount exceeds maximum")
	ErrInvalidIdempotencyKey = errors.New("idempotency key is required")
	ErrNegativeBalance       = errors.New("balance integrity error: negative balance detected")
)

type WalletService struct {
	repo *repository.WalletRepository
}

func NewWalletService(repo *repository.WalletRepository) *WalletService {
	return &WalletService{repo: repo}
}

// Deposit processes a deposit request
func (s *WalletService) Deposit(ctx context.Context, req models.DepositRequest) (*models.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[DEPOSIT] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s", 
		req.UserID, req.Amount, req.IdempotencyKey)

	// 1. Validate idempotency key FIRST (before any other validation)
	if req.IdempotencyKey == "" {
		log.Printf("[DEPOSIT] Failed - Missing idempotency key")
		return nil, ErrInvalidIdempotencyKey
	}

	// 2. Check if this request was already processed (idempotency check)
	existingTxnID, exists, err := s.repo.CheckIdempotency(ctx, req.IdempotencyKey)
	if err != nil {
		log.Printf("[DEPOSIT] Error checking idempotency: %v", err)
		return nil, s.translateError(err)
	}
	
	if exists {
		// Request already processed - return existing result
		log.Printf("[DEPOSIT] Idempotent request detected - Returning existing transaction: %d", existingTxnID)
		_, balance, err := s.repo.GetAccountByUserID(ctx, req.UserID)
		if err != nil {
			return nil, s.translateError(err)
		}
		
		return &models.TransactionResponse{
			TransactionID: existingTxnID,
			Status:        "posted",
			Balance:       balance,
			Reference:     req.Reference,
			Message:       "Transaction already processed (idempotent)",
		}, nil
	}

	// 3. Validate amount (after idempotency check)
	if err := s.validateDepositAmount(req.Amount); err != nil {
		log.Printf("[DEPOSIT] Failed validation - %v", err)
		return nil, err
	}

	// 4. Process deposit through repository
	txnID, newBalance, err := s.repo.Deposit(ctx, req.UserID, req.Amount, req.IdempotencyKey, req.Reference)
	if err != nil {
		log.Printf("[DEPOSIT] Repository error - UserID: %d, Error: %v", req.UserID, err)
		return nil, s.translateError(err)
	}

	// 5. Sanity check: balance should never be negative
	if newBalance < 0 {
		log.Printf("[DEPOSIT] CRITICAL - Negative balance detected! UserID: %d, Balance: %d", req.UserID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[DEPOSIT] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	// 6. Build response
	message := fmt.Sprintf("Successfully deposited ₦%.2f", float64(req.Amount)/100)
	if req.Reference != "" {
		message += fmt.Sprintf(" (Ref: %s)", req.Reference)
	}

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       message,
	}, nil
}

// Withdraw processes a withdrawal request
func (s *WalletService) Withdraw(ctx context.Context, req models.WithdrawRequest) (*models.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[WITHDRAW] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s", 
		req.UserID, req.Amount, req.IdempotencyKey)

	// 1. Validate idempotency key FIRST
	if req.IdempotencyKey == "" {
		log.Printf("[WITHDRAW] Failed - Missing idempotency key")
		return nil, ErrInvalidIdempotencyKey
	}

	// 2. Check if this request was already processed
	existingTxnID, exists, err := s.repo.CheckIdempotency(ctx, req.IdempotencyKey)
	if err != nil {
		log.Printf("[WITHDRAW] Error checking idempotency: %v", err)
		return nil, s.translateError(err)
	}
	
	if exists {
		log.Printf("[WITHDRAW] Idempotent request detected - Returning existing transaction: %d", existingTxnID)
		_, balance, err := s.repo.GetAccountByUserID(ctx, req.UserID)
		if err != nil {
			return nil, s.translateError(err)
		}
		
		return &models.TransactionResponse{
			TransactionID: existingTxnID,
			Status:        "posted",
			Balance:       balance,
			Reference:     req.Reference,
			Message:       "Transaction already processed (idempotent)",
		}, nil
	}

	// 3. Validate amount
	if err := s.validateWithdrawAmount(req.Amount); err != nil {
		log.Printf("[WITHDRAW] Failed validation - %v", err)
		return nil, err
	}

	// 4. Process withdrawal through repository
	txnID, newBalance, err := s.repo.Withdraw(ctx, req.UserID, req.Amount, req.IdempotencyKey, req.Reference)
	if err != nil {
		log.Printf("[WITHDRAW] Repository error - UserID: %d, Error: %v", req.UserID, err)
		return nil, s.translateError(err)
	}

	// 5. Sanity check: balance should never be negative after withdrawal
	if newBalance < 0 {
		log.Printf("[WITHDRAW] CRITICAL - Negative balance detected! UserID: %d, Balance: %d", req.UserID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[WITHDRAW] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	// 6. Build response
	message := fmt.Sprintf("Successfully withdrew ₦%.2f", float64(req.Amount)/100)
	if req.Reference != "" {
		message += fmt.Sprintf(" (Ref: %s)", req.Reference)
	}

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       message,
	}, nil
}

// GetBalance retrieves a user's current balance
func (s *WalletService) GetBalance(ctx context.Context, userID int) (int64, error) {
	log.Printf("[GET_BALANCE] UserID: %d", userID)
	
	_, balance, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		log.Printf("[GET_BALANCE] Error - UserID: %d, Error: %v", userID, err)
		return 0, s.translateError(err)
	}
	
	log.Printf("[GET_BALANCE] Success - UserID: %d, Balance: %d kobo", userID, balance)
	return balance, nil
}

// validateDepositAmount checks if deposit amount meets business rules
func (s *WalletService) validateDepositAmount(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	
	if amount < MinDepositAmount {
		return fmt.Errorf("%w: minimum deposit is ₦%.2f", ErrAmountTooSmall, float64(MinDepositAmount)/100)
	}
	
	if amount > MaxTransactionAmount {
		return fmt.Errorf("%w: maximum per transaction is ₦%.2f", ErrAmountTooLarge, float64(MaxTransactionAmount)/100)
	}
	
	return nil
}

// validateWithdrawAmount checks if withdrawal amount meets business rules
func (s *WalletService) validateWithdrawAmount(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	
	if amount < MinWithdrawAmount {
		return fmt.Errorf("%w: minimum withdrawal is ₦%.2f", ErrAmountTooSmall, float64(MinWithdrawAmount)/100)
	}
	
	if amount > MaxTransactionAmount {
		return fmt.Errorf("%w: maximum per transaction is ₦%.2f", ErrAmountTooLarge, float64(MaxTransactionAmount)/100)
	}
	
	return nil
}

// GetTransactionHistory retrieves all transaction history for a user
/* func (s *WalletService) GetTransactionHistory(ctx context.Context, userID int) ([]models.TransactionHistoryItem, error) {
	log.Printf("[GET_HISTORY] UserID: %d", userID)
	
	transactions, err := s.repo.GetTransactionHistory(ctx, userID)
	if err != nil {
		log.Printf("[GET_HISTORY] Error - UserID: %d, Error: %v", userID, err)
		return nil, s.translateError(err)
	}
	
	log.Printf("[GET_HISTORY] Success - UserID: %d, Found: %d transactions", userID, len(transactions))
	
	return transactions, nil
} */

// translateError converts repository errors into service-level errors with context preserved
func (s *WalletService) translateError(err error) error {
	switch {
	case errors.Is(err, repository.ErrAccountNotFound):
		return fmt.Errorf("account not found: %w", err)
	case errors.Is(err, repository.ErrInsufficientBalance):
		return fmt.Errorf("insufficient balance for withdrawal: %w", err)
	case errors.Is(err, repository.ErrDuplicateIdempotencyKey):
		return fmt.Errorf("duplicate transaction detected: %w", err)
	default:
		// Preserve original error for debugging while providing user-friendly message
		log.Printf("[ERROR] Unexpected repository error: %v", err)
		return fmt.Errorf("transaction failed, please try again: %w", err)
	}
}