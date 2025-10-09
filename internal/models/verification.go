package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// VERIFICATION CODE MODEL
// ==============================================

type VerificationCode struct {
	ID        int32            `db:"id"`
	UserID    pgtype.Int4      `db:"user_id"`     // NULL for pre-registration verification
	Email     string           `db:"email"`
	Code      string           `db:"code"`        // 6-digit OTP
	Purpose   string           `db:"purpose"`
	ExpiresAt time.Time        `db:"expires_at"`
	UsedAt    pgtype.Timestamp `db:"used_at"`
	Attempts  int32            `db:"attempts"`
	IPAddress pgtype.Text      `db:"ip_address"`
	CreatedAt time.Time        `db:"created_at"`
}

func (v *VerificationCode) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

func (v *VerificationCode) IsUsed() bool {
	return v.UsedAt.Valid
}

func (v *VerificationCode) IsValid() bool {
	return !v.IsExpired() && !v.IsUsed() && v.Attempts < OTPMaxAttempts
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
	OTPLength         = 6                // 6-digit OTP
	OTPExpiryMinutes  = 10               // OTP expires in 10 minutes
	OTPMaxAttempts    = 5                // Max verification attempts
	OTPResendCooldown = 60 * time.Second // 60 seconds between resends
)