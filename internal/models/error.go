package models

import (
	"errors"
	"fmt"
)

// ==============================================
// CUSTOM ERROR TYPES
// ==============================================

// AppError represents a structured application error
type AppError struct {
	Code    string // Error code for client
	Message string // Human-readable message
	Err     error  // Underlying error (for logging)
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// ==============================================
// PREDEFINED ERRORS
// ==============================================

// User/Auth Errors
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrAccountLocked        = errors.New("account is locked")
	ErrAccountInactive      = errors.New("account is inactive")
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrPhoneAlreadyExists   = errors.New("phone number already registered")
	ErrEmailAlreadyExists   = errors.New("email already registered")
	ErrUsernameAlreadyExists = errors.New("username already taken")
	ErrInvalidPhone         = errors.New("invalid phone number")
	ErrInvalidEmail         = errors.New("invalid email address")
	ErrWeakPassword         = errors.New("password too weak")
	ErrInvalidPin           = errors.New("invalid PIN")
	ErrPinNotSet            = errors.New("transaction PIN not set")
	ErrIncorrectPin         = errors.New("incorrect PIN")
)

// OTP Errors
var (
	ErrOTPExpired      = errors.New("OTP has expired")
	ErrOTPInvalid      = errors.New("invalid OTP")
	ErrOTPAlreadyUsed  = errors.New("OTP already used")
	ErrOTPMaxAttempts  = errors.New("maximum OTP attempts exceeded")
	ErrOTPNotFound     = errors.New("OTP not found")
	ErrOTPResendCooldown = errors.New("please wait before requesting another OTP")
)

// Wallet/Account Errors
var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrAccountFrozen      = errors.New("account is frozen")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrSameAccount        = errors.New("cannot transfer to same account")
	ErrSystemAccountTransfer = errors.New("cannot transfer directly to system account")
)

// Transaction Errors
var (
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrTransactionAlreadyExists = errors.New("transaction already exists (duplicate idempotency key)")
	ErrTransactionFailed       = errors.New("transaction failed")
	ErrTransactionPending      = errors.New("transaction is still pending")
	ErrInvalidTransactionKind  = errors.New("invalid transaction kind")
	ErrInvalidTransactionStatus = errors.New("invalid transaction status")
	ErrPostingMismatch         = errors.New("postings do not balance (double-entry violation)")
)

// Session Errors
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrSessionRevoked   = errors.New("session revoked")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
)

// ==============================================
// ERROR CODES (for API responses)
// ==============================================
const (
	// Auth error codes
	ErrCodeInvalidCredentials   = "INVALID_CREDENTIALS"
	ErrCodeAccountLocked        = "ACCOUNT_LOCKED"
	ErrCodeAccountInactive      = "ACCOUNT_INACTIVE"
	ErrCodeEmailNotVerified     = "EMAIL_NOT_VERIFIED"
	ErrCodeUserExists           = "USER_EXISTS"
	ErrCodeWeakPassword         = "WEAK_PASSWORD"
	ErrCodeInvalidPin           = "INVALID_PIN"
	
	// OTP error codes
	ErrCodeOTPExpired          = "OTP_EXPIRED"
	ErrCodeOTPInvalid          = "OTP_INVALID"
	ErrCodeOTPMaxAttempts      = "OTP_MAX_ATTEMPTS"
	
	// Wallet error codes
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodeAccountFrozen       = "ACCOUNT_FROZEN"
	ErrCodeInvalidAmount       = "INVALID_AMOUNT"
	
	// Transaction error codes
	ErrCodeTransactionFailed   = "TRANSACTION_FAILED"
	ErrCodeDuplicateTransaction = "DUPLICATE_TRANSACTION"
	
	// Generic error codes
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
)

// ==============================================
// HELPER FUNCTIONS
// ==============================================

// IsNotFoundError checks if error is a "not found" error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrAccountNotFound) ||
		errors.Is(err, ErrTransactionNotFound) ||
		errors.Is(err, ErrSessionNotFound)
}

// IsAuthError checks if error is authentication-related
func IsAuthError(err error) bool {
	return errors.Is(err, ErrInvalidCredentials) ||
		errors.Is(err, ErrAccountLocked) ||
		errors.Is(err, ErrAccountInactive) ||
		errors.Is(err, ErrSessionExpired) ||
		errors.Is(err, ErrInvalidToken)
}

// IsValidationError checks if error is validation-related
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidPhone) ||
		errors.Is(err, ErrInvalidEmail) ||
		errors.Is(err, ErrWeakPassword) ||
		errors.Is(err, ErrInvalidAmount)
}