package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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
	DefaultTransferFee   = 5000      // ₦50.00 default transfer fee
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

func (s *WalletService) Deposit(ctx context.Context, req models.DepositRequest) (*models.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[DEPOSIT] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s",
		req.UserID, req.Amount, req.IdempotencyKey)

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
		return s.buildIdempotentResponse(ctx, existingTxn.ID, req.UserID, req.Reference)
	}

	// 3. Execute deposit transaction with locking
	txnID, newBalance, err := s.executeDeposit(ctx, req)
	if err != nil {
		log.Printf("[DEPOSIT] Failed - UserID: %d, Error: %v", req.UserID, err)
		return nil, err
	}

	// 4. Validate result
	if newBalance < 0 {
		log.Printf("[DEPOSIT] CRITICAL - Negative balance! UserID: %d, Balance: %d", req.UserID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[DEPOSIT] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       fmt.Sprintf("Successfully deposited ₦%.2f", float64(req.Amount)/100),
	}, nil
}

func (s *WalletService) executeDeposit(ctx context.Context, req models.DepositRequest) (int64, int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Lock user account (prevents concurrent deposits to same account)
	userAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, req.UserID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, err
	}

	// Lock reserve account (prevents concurrent access to reserve)
	reserveAccount, err := s.repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")
	if err != nil {
		return 0, 0, fmt.Errorf("reserve account not found: %w", err)
	}

	txn := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Kind:           "deposit",
		Status:         "posted",
		Reference:      &req.Reference,
	}
	if err := s.repo.CreateTransaction(ctx, tx, txn); err != nil {
		return 0, 0, err
	}

	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     reserveAccount.ID,
		Amount:        -req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, err
	}

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

	// Calculate new balance after commit
	newBalance := userAccount.Balance + req.Amount

	return txn.ID, newBalance, nil
}

// ==============================================
// WITHDRAW
// ==============================================

func (s *WalletService) Withdraw(ctx context.Context, req models.WithdrawRequest) (*models.TransactionResponse, error) {
	startTime := time.Now()
	log.Printf("[WITHDRAW] Started - UserID: %d, Amount: %d kobo, IdempotencyKey: %s",
		req.UserID, req.Amount, req.IdempotencyKey)

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
		return s.buildIdempotentResponse(ctx, existingTxn.ID, req.UserID, req.Reference)
	}

	txnID, newBalance, err := s.executeWithdraw(ctx, req)
	if err != nil {
		log.Printf("[WITHDRAW] Failed - UserID: %d, Error: %v", req.UserID, err)
		return nil, err
	}

	if newBalance < 0 {
		log.Printf("[WITHDRAW] CRITICAL - Negative balance! UserID: %d, Balance: %d", req.UserID, newBalance)
		return nil, ErrNegativeBalance
	}

	duration := time.Since(startTime)
	log.Printf("[WITHDRAW] Success - TxnID: %d, NewBalance: %d kobo, Duration: %v", txnID, newBalance, duration)

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       newBalance,
		Reference:     req.Reference,
		Message:       fmt.Sprintf("Successfully withdrew ₦%.2f", float64(req.Amount)/100),
	}, nil
}

func (s *WalletService) executeWithdraw(ctx context.Context, req models.WithdrawRequest) (int64, int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Lock user account and check balance atomically
	userAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, req.UserID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, ErrAccountNotFound
		}
		return 0, 0, err
	}

	// Check sufficient balance while holding lock
	if userAccount.Balance < req.Amount {
		return 0, 0, ErrInsufficientBalance
	}

	// Lock reserve account
	reserveAccount, err := s.repo.GetSystemAccountForUpdate(ctx, tx, "sys_reserve")
	if err != nil {
		return 0, 0, fmt.Errorf("reserve account not found: %w", err)
	}

	txn := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Kind:           "withdrawal",
		Status:         "posted",
		Reference:      &req.Reference,
	}
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
// TRANSFER (CRITICAL - Deadlock Prevention)
// ==============================================

func (s *WalletService) Transfer(ctx context.Context, req models.TransferRequest) (*models.TransferResponse, error) {
	startTime := time.Now()
	log.Printf("[TRANSFER] Started - From: %d, To: %d, Amount: %d kobo, Fee: %d kobo",
		req.FromUserID, req.ToUserID, req.Amount, req.Fee)

	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotencyKey
	}
	if req.FromUserID == req.ToUserID {
		return nil, ErrSameAccount
	}
	if err := s.validateTransferAmount(req.Amount); err != nil {
		log.Printf("[TRANSFER] Validation failed: %v", err)
		return nil, err
	}
	if req.Fee < 0 {
		return nil, ErrInvalidAmount
	}

	existingTxn, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil && !isNoRowsError(err) {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if existingTxn != nil {
		log.Printf("[TRANSFER] Idempotent request - Returning existing transaction: %d", existingTxn.ID)
		return s.buildIdempotentTransferResponse(ctx, existingTxn.ID, req.FromUserID, req.ToUserID)
	}

	txnID, senderBalance, recipientBalance, err := s.executeTransfer(ctx, req)
	if err != nil {
		log.Printf("[TRANSFER] Failed - Error: %v", err)
		return nil, err
	}

	duration := time.Since(startTime)
	log.Printf("[TRANSFER] Success - TxnID: %d, Duration: %v", txnID, duration)

	return &models.TransferResponse{
		TransactionID:    txnID,
		Status:           "posted",
		SenderBalance:    senderBalance,
		RecipientBalance: recipientBalance,
		Message:          fmt.Sprintf("Successfully transferred ₦%.2f", float64(req.Amount)/100),
	}, nil
}

func (s *WalletService) executeTransfer(ctx context.Context, req models.TransferRequest) (int64, int64, int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// CRITICAL: Lock accounts in consistent order to prevent deadlocks
	// Always lock the account with lower user_id first
	firstUserID := req.FromUserID
	secondUserID := req.ToUserID
	if req.ToUserID < req.FromUserID {
		firstUserID = req.ToUserID
		secondUserID = req.FromUserID
	}

	log.Printf("[TRANSFER] Locking accounts in order: %d, %d", firstUserID, secondUserID)

	// Lock first account
	firstAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, firstUserID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, 0, ErrAccountNotFound
		}
		return 0, 0, 0, err
	}

	// Lock second account
	secondAccount, err := s.repo.GetAccountByUserIDForUpdate(ctx, tx, secondUserID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return 0, 0, 0, ErrAccountNotFound
		}
		return 0, 0, 0, err
	}

	// Map back to sender/recipient
	var senderAccount, recipientAccount *models.Account
	if firstUserID == req.FromUserID {
		senderAccount = firstAccount
		recipientAccount = secondAccount
	} else {
		senderAccount = secondAccount
		recipientAccount = firstAccount
	}

	// Check sufficient balance while holding locks
	totalDebit := req.Amount + req.Fee
	if senderAccount.Balance < totalDebit {
		return 0, 0, 0, ErrInsufficientBalance
	}

	txn := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Kind:           "p2p",
		Status:         "posted",
		Reference:      &req.Reference,
	}
	if err := s.repo.CreateTransaction(ctx, tx, txn); err != nil {
		return 0, 0, 0, err
	}

	// Debit sender
	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     senderAccount.ID,
		Amount:        -totalDebit,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, 0, err
	}

	// Credit recipient
	if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
		TransactionID: txn.ID,
		AccountID:     recipientAccount.ID,
		Amount:        req.Amount,
		Currency:      "NGN",
	}); err != nil {
		return 0, 0, 0, err
	}

	// Handle fee if present
	if req.Fee > 0 {
		feeAccount, err := s.repo.GetSystemAccountForUpdate(ctx, tx, "sys_fee")
		if err != nil {
			return 0, 0, 0, fmt.Errorf("fee account not found: %w", err)
		}

		if err := s.repo.CreatePosting(ctx, tx, &models.Posting{
			TransactionID: txn.ID,
			AccountID:     feeAccount.ID,
			Amount:        req.Fee,
			Currency:      "NGN",
		}); err != nil {
			return 0, 0, 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to commit: %w", err)
	}

	// Calculate final balances
	senderBalance := senderAccount.Balance - totalDebit
	recipientBalance := recipientAccount.Balance + req.Amount

	return txn.ID, senderBalance, recipientBalance, nil
}

// ==============================================
// GET BALANCE & HISTORY
// ==============================================

func (s *WalletService) GetBalance(ctx context.Context, userID int) (*models.BalanceResponse, error) {
	log.Printf("[GET_BALANCE] UserID: %d", userID)

	account, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		if isAccountNotFoundError(err) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	return &models.BalanceResponse{
		UserID:     userID,
		Balance:    account.Balance,
		BalanceNGN: float64(account.Balance) / 100,
		Currency:   account.Currency,
	}, nil
}

func (s *WalletService) GetTransactionHistory(ctx context.Context, userID, page, perPage int) (*models.TransactionHistoryResponse, error) {
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

	log.Printf("[GET_HISTORY] Success - UserID: %d, Found: %d/%d transactions", userID, len(transactions), total)

	return &models.TransactionHistoryResponse{
		UserID:       userID,
		Transactions: transactions,
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

func (s *WalletService) buildIdempotentResponse(ctx context.Context, txnID int64, userID int, reference string) (*models.TransactionResponse, error) {
	account, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        "posted",
		Balance:       account.Balance,
		Reference:     reference,
		Message:       "Transaction already processed (idempotent)",
	}, nil
}

func (s *WalletService) buildIdempotentTransferResponse(ctx context.Context, txnID int64, fromUserID, toUserID int) (*models.TransferResponse, error) {
	senderAccount, err := s.repo.GetAccountByUserID(ctx, fromUserID)
	if err != nil {
		return nil, err
	}

	recipientAccount, err := s.repo.GetAccountByUserID(ctx, toUserID)
	if err != nil {
		return nil, err
	}

	return &models.TransferResponse{
		TransactionID:    txnID,
		Status:           "posted",
		SenderBalance:    senderAccount.Balance,
		RecipientBalance: recipientAccount.Balance,
		Message:          "Transaction already processed (idempotent)",
	}, nil
}

// Error helper functions
func isNoRowsError(err error) bool {
	return err != nil && (err.Error() == "no rows found" || errors.Is(err, errors.New("no rows found")))
}

func isAccountNotFoundError(err error) bool {
	return err != nil && (err.Error() == "account not found" || errors.Is(err, errors.New("account not found")))
}