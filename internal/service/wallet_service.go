package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Brownie44l1/debank/internal/api/dto"
	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5"
)

// ==============================================
// REPOSITORY INTERFACE (for testing)
// ==============================================

type WalletRepositoryInterface interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	GetAccountByUserID(ctx context.Context, userID int) (*models.Account, error)
	GetAccountByUserIDForUpdate(ctx context.Context, tx pgx.Tx, userID int) (*models.Account, error)
	GetSystemAccount(ctx context.Context, externalID string) (*models.Account, error)
	GetSystemAccountForUpdate(ctx context.Context, tx pgx.Tx, externalID string) (*models.Account, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error)
	CreateTransaction(ctx context.Context, tx pgx.Tx, txn *models.Transaction) error
	CreatePosting(ctx context.Context, tx pgx.Tx, posting *models.Posting) error
	GetTransactionHistory(ctx context.Context, userID int, limit, offset int) ([]models.TransactionHistoryItem, error)
	CountTransactionHistory(ctx context.Context, userID int) (int, error)
}

// ==============================================
// BUSINESS RULES (Constants)
// ==============================================

const (
	MinDepositAmount     = 10000     // ₦100.00 minimum deposit
	MinWithdrawAmount    = 10000     // ₦100.00 minimum withdrawal
	MinTransferAmount    = 10000     // ₦100.00 minimum transfer
	MaxTransactionAmount = 100000000 // ₦1,000,000.00 maximum per transaction
	DefaultTransferFee   = 0         // ₦0.00 (free transfers for now)
)

// ==============================================
// SERVICE ERRORS
// ==============================================

var (
	ErrInvalidAmount         = errors.New("invalid transaction amount")
	ErrAmountTooSmall        = errors.New("amount is below minimum")
	ErrAmountTooLarge        = errors.New("amount exceeds maximum")
	ErrInvalidIdempotencyKey = errors.New("idempotency key is required")
	ErrNegativeBalance       = errors.New("balance integrity error: negative balance detected")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrAccountNotFound       = errors.New("account not found")
	ErrSameAccount           = errors.New("cannot transfer to same account")
)

// ==============================================
// SERVICE
// ==============================================

type WalletService struct {
	repo WalletRepositoryInterface
}

func NewWalletService(repo WalletRepositoryInterface) *WalletService {
	return &WalletService{repo: repo}
}

// ==============================================
// DEPOSIT
// ==============================================

func (s *WalletService) Deposit(ctx context.Context, userID int, req dto.DepositRequest) (*dto.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[DEPOSIT] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s",
		userID, req.Amount, req.IdempotencyKey)

	// 1. Validate inputs
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotencyKey
	}
	if err := s.validateDepositAmount(req.Amount); err != nil {
		log.Printf("[DEPOSIT] Validation failed: %v", err)
		return nil, err
	}

	// 2. Check idempotency (before starting transaction)
	existingTxn, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil && !isNoRowsError(err) {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if existingTxn != nil {
		log.Printf("[DEPOSIT] Idempotent request - Returning existing transaction: %d", existingTxn.ID)
		return s.buildIdempotentResponse(ctx, existingTxn.ID, userID, req.Reference)
	}

	// 3. Execute deposit transaction with locking
	txnID, newBalance, err := s.executeDeposit(ctx, userID, req)
	if err != nil {
		log.Printf("[DEPOSIT] Failed - UserID: %d, Error: %v", userID, err)
		return nil, err
	}

	// 4. Validate result
	if newBalance < 0 {
		log.Printf("[DEPOSIT] CRITICAL - Negative balance! UserID: %d, Balance: %d", userID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[DEPOSIT] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	return &dto.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       fmt.Sprintf("Successfully deposited ₦%.2f", float64(req.Amount)/100),
	}, nil
}

func (s *WalletService) executeDeposit(ctx context.Context, userID int, req dto.DepositRequest) (int64, int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Lock user account
	userAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, err
	}

	// Lock reserve account
	reserveAccount, err := s.repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")
	if err != nil {
		return 0, 0, fmt.Errorf("reserve account not found: %w", err)
	}

	// Create transaction
	txn := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Reference:      req.Reference,
		Kind:           models.TransactionKindDeposit,
		Status:         models.TransactionStatusPosted,
		Amount:         req.Amount,
		Currency:       "NGN",
	}
	
	// Set account IDs
	txn.FromAccountID.Int64 = reserveAccount.ID
	txn.FromAccountID.Valid = true
	txn.ToAccountID.Int64 = userAccount.ID
	txn.ToAccountID.Valid = true

	if err := s.repo.CreateTransaction(ctx, tx, txn); err != nil {
		return 0, 0, err
	}

	// Debit reserve
	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     reserveAccount.ID,
		Amount:        -req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, err
	}

	// Credit user
	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     userAccount.ID,
		Amount:        req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to commit: %w", err)
	}

	newBalance := userAccount.Balance + req.Amount
	return txn.ID, newBalance, nil
}

// ==============================================
// WITHDRAW
// ==============================================

func (s *WalletService) Withdraw(ctx context.Context, userID int, req dto.WithdrawRequest) (*dto.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[WITHDRAW] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s",
		userID, req.Amount, req.IdempotencyKey)

	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotencyKey
	}
	if err := s.validateWithdrawAmount(req.Amount); err != nil {
		log.Printf("[WITHDRAW] Validation failed: %v", err)
		return nil, err
	}

	existingTxn, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil && !isNoRowsError(err) {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if existingTxn != nil {
		log.Printf("[WITHDRAW] Idempotent request - Returning existing transaction: %d", existingTxn.ID)
		return s.buildIdempotentResponse(ctx, existingTxn.ID, userID, req.Reference)
	}

	txnID, newBalance, err := s.executeWithdraw(ctx, userID, req)
	if err != nil {
		log.Printf("[WITHDRAW] Failed - UserID: %d, Error: %v", userID, err)
		return nil, err
	}

	if newBalance < 0 {
		log.Printf("[WITHDRAW] CRITICAL - Negative balance! UserID: %d, Balance: %d", userID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[WITHDRAW] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	return &dto.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       fmt.Sprintf("Successfully withdrew ₦%.2f", float64(req.Amount)/100),
	}, nil
}

func (s *WalletService) executeWithdraw(ctx context.Context, userID int, req dto.WithdrawRequest) (int64, int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, err
	}

	if userAccount.Balance < req.Amount {
		return 0, 0, ErrInsufficientBalance
	}

	reserveAccount, err := s.repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")
	if err != nil {
		return 0, 0, fmt.Errorf("reserve account not found: %w", err)
	}

	txn := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Reference:      req.Reference,
		Kind:           models.TransactionKindWithdraw,
		Status:         models.TransactionStatusPosted,
		Amount:         req.Amount,
		Currency:       "NGN",
	}
	
	txn.FromAccountID.Int64 = userAccount.ID
	txn.FromAccountID.Valid = true
	txn.ToAccountID.Int64 = reserveAccount.ID
	txn.ToAccountID.Valid = true

	if err := s.repo.CreateTransaction(ctx, tx, txn); err != nil {
		return 0, 0, err
	}

	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     userAccount.ID,
		Amount:        -req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, err
	}

	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     reserveAccount.ID,
		Amount:        req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to commit: %w", err)
	}

	newBalance := userAccount.Balance - req.Amount
	return txn.ID, newBalance, nil
}

// ==============================================
// GET BALANCE
// ==============================================

func (s *WalletService) GetBalance(ctx context.Context, userID int) (*dto.BalanceResponse, error) {
	log.Printf("[GET_BALANCE] UserID: %d", userID)

	account, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	accountNumber := ""
	if account.AccountNumber.Valid {
		accountNumber = account.AccountNumber.String
	}

	return &dto.BalanceResponse{
		UserID:        userID,
		AccountNumber: accountNumber,
		Balance:       account.Balance,
		BalanceNGN:    float64(account.Balance) / 100,
		Currency:      account.Currency,
	}, nil
}

// ==============================================
// GET TRANSACTION HISTORY
// ==============================================

func (s *WalletService) GetTransactionHistory(ctx context.Context, userID, page, perPage int) (*dto.TransactionHistoryResponse, error) {
	log.Printf("[GET_HISTORY] UserID: %d, Page: %d, PerPage: %d", userID, page, perPage)

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	transactions, err := s.repo.GetTransactionHistory(ctx, userID, perPage, offset)
	if err != nil {
		if isAccountNotFoundError(err) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	total, err := s.repo.CountTransactionHistory(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtoTransactions := make([]dto.TransactionHistoryItem, len(transactions))
	for i, txn := range transactions {
		dtoTransactions[i] = dto.TransactionHistoryItem{
			ID:           txn.ID,
			Reference:    txn.Reference,
			Type:         txn.Type,
			Status:       txn.Status,
			Amount:       txn.Amount,
			AmountNGN:    float64(txn.Amount) / 100,
			Description:  txn.Description,
			Direction:    txn.Direction,
			Counterparty: txn.Counterparty,
			CreatedAt:    txn.CreatedAt.Format(time.RFC3339),
		}
	}

	log.Printf("[GET_HISTORY] Success - UserID: %d, Found: %d/%d transactions", userID, len(transactions), total)

	return &dto.TransactionHistoryResponse{
		UserID:       userID,
		Transactions: dtoTransactions,
		Total:        total,
		Page:         page,
		PerPage:      perPage,
	}, nil
}

// ==============================================
// VALIDATION & HELPERS
// ==============================================

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

func (s *WalletService) validateTransferAmount(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount < MinTransferAmount {
		return fmt.Errorf("%w: minimum transfer is ₦%.2f", ErrAmountTooSmall, float64(MinTransferAmount)/100)
	}
	if amount > MaxTransactionAmount {
		return fmt.Errorf("%w: maximum per transaction is ₦%.2f", ErrAmountTooLarge, float64(MaxTransactionAmount)/100)
	}
	return nil
}