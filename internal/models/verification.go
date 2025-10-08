package models

import (
	"database/sql"
	"time"
)

// ==============================================
// VERIFICATION CODE MODEL
// ==============================================

// VerificationCode represents an OTP sent via email
type VerificationCode struct {
	ID        int            `db:"id"`
	UserID    sql.NullInt32  `db:"user_id"` // NULL for pre-registration verification
	Email     string         `db:"email"`
	Code      string         `db:"code"` // 6-digit OTP
	Purpose   string         `db:"purpose"`
	ExpiresAt time.Time      `db:"expires_at"`
	UsedAt    sql.NullTime   `db:"used_at"`
	Attempts  int            `db:"attempts"`
	IPAddress sql.NullString `db:"ip_address"`
	CreatedAt time.Time      `db:"created_at"`
}

// IsExpired checks if the OTP has expired
func (v *VerificationCode) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

// IsUsed checks if the OTP has been used
func (v *VerificationCode) IsUsed() bool {
	return v.UsedAt.Valid
}

// IsValid checks if OTP can still be verified
func (v *VerificationCode) IsValid() bool {
	return !v.IsExpired() && !v.IsUsed() && v.Attempts < 5
}

// ==============================================
// OTP PURPOSE CONSTANTS
// ==============================================
const (
	OTPPurposeEmailVerify     = "email_verify"
	OTPPurposePasswordReset   = "password_reset"
	OTPPurposeTransactionAuth = "transaction_auth"
	OTPPurposeSettingsChange  = "settings_change"
	OTPPurposeLoginMFA        = "login_mfa"
)

// ==============================================
// OTP CONFIGURATION
// ==============================================
const (
	OTPLength          = 6                // 6-digit OTP
	OTPExpiryMinutes   = 10               // OTP expires in 10 minutes
	OTPMaxAttempts     = 5                // Max verification attempts
	OTPResendCooldown  = 60 * time.Second // 60 seconds between resends
)