package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/repository"
)

// Business logic constants
const (
	MinDepositAmount  = 10000  // ₦100.00 minimum deposit
	MinWithdrawAmount = 10000  // ₦100.00 minimum withdrawal
	MaxTransactionAmount = 100000000 // ₦1,000,000.00 maximum per transaction
)

var (
	ErrInvalidAmount       = errors.New("invalid transaction amount")
	ErrAmountTooSmall      = errors.New("amount is below minimum")
	ErrAmountTooLarge      = errors.New("amount exceeds maximum")
	ErrInvalidIdempotencyKey = errors.New("idempotency key is required")
)

type WalletService struct {
	repo *repository.WalletRepository
}

func NewWalletService(repo *repository.WalletRepository) *WalletService {
	return &WalletService{repo: repo}
}

// Deposit processes a deposit request
func (s *WalletService) Deposit(ctx context.Context, req models.DepositRequest) (*models.TransactionResponse, error) {
	// 1. Validate amount
	if err := s.validateAmount(req.Amount); err != nil {
		return nil, err
	}

	// 2. Validate idempotency key
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotencyKey
	}

	// 3. Process deposit through repository
	txnID, newBalance, err := s.repo.Deposit(ctx, req.UserID, req.Amount, req.IdempotencyKey, req.Reference)
	if err != nil {
		return nil, s.translateError(err)
	}

	// 4. Build response
	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Message:       fmt.Sprintf("Successfully deposited ₦%.2f", float64(req.Amount)/100),
	}, nil
}

// Withdraw processes a withdrawal request
func (s *WalletService) Withdraw(ctx context.Context, req models.WithdrawRequest) (*models.TransactionResponse, error) {
	// 1. Validate amount
	if err := s.validateAmount(req.Amount); err != nil {
		return nil, err
	}

	// 2. Validate idempotency key
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotencyKey
	}

	// 3. Process withdrawal through repository
	txnID, newBalance, err := s.repo.Withdraw(ctx, req.UserID, req.Amount, req.IdempotencyKey, req.Reference)
	if err != nil {
		return nil, s.translateError(err)
	}

	// 4. Build response
	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Message:       fmt.Sprintf("Successfully withdrew ₦%.2f", float64(req.Amount)/100),
	}, nil
}

// GetBalance retrieves a user's current balance
func (s *WalletService) GetBalance(ctx context.Context, userID int) (int64, error) {
	_, balance, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		return 0, s.translateError(err)
	}
	return balance, nil
}

// validateAmount checks if the amount meets business rules
func (s *WalletService) validateAmount(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	
	if amount < MinDepositAmount {
		return fmt.Errorf("%w: minimum is ₦%.2f", ErrAmountTooSmall, float64(MinDepositAmount)/100)
	}
	
	if amount > MaxTransactionAmount {
		return fmt.Errorf("%w: maximum is ₦%.2f", ErrAmountTooLarge, float64(MaxTransactionAmount)/100)
	}
	
	return nil
}

// translateError converts repository errors into service-level errors
func (s *WalletService) translateError(err error) error {
	switch {
	case errors.Is(err, repository.ErrAccountNotFound):
		return errors.New("account not found")
	case errors.Is(err, repository.ErrInsufficientBalance):
		return errors.New("insufficient balance for withdrawal")
	case errors.Is(err, repository.ErrDuplicateIdempotencyKey):
		return errors.New("duplicate transaction detected")
	default:
		// Log the original error for debugging
		// logger.Error("repository error", "error", err)
		return errors.New("transaction failed, please try again")
	}
}