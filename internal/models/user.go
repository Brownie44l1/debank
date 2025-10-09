package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// USER MODEL (Database mapping)
// ==============================================

type User struct {
	ID                  int32           `db:"id"`
	Name                string          `db:"name"`
	Phone               string          `db:"phone"`
	Email               string          `db:"email"`
	PasswordHash        string          `db:"password_hash"`
	Username            pgtype.Text     `db:"username"`
	PinHash             pgtype.Text     `db:"pin_hash"`
	IsEmailVerified     bool            `db:"is_email_verified"`
	IsActive            bool            `db:"is_active"`
	OnboardingCompleted bool            `db:"onboarding_completed"`
	FailedLoginAttempts int32           `db:"failed_login_attempts"`
	LockedUntil         pgtype.Timestamp `db:"locked_until"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
	LastLoginAt         pgtype.Timestamp `db:"last_login_at"`
}

// ==============================================
// PUBLIC USER MODEL
// ==============================================

type PublicUser struct {
	ID                  int32       `json:"id"`
	Name                string      `json:"name"`
	Phone               string      `json:"phone"`
	Email               string      `json:"email"`
	Username            *string     `json:"username,omitempty"`
	IsEmailVerified     bool        `json:"is_email_verified"`
	OnboardingCompleted bool        `json:"onboarding_completed"`
	CreatedAt           time.Time   `json:"created_at"`
	LastLoginAt         *time.Time  `json:"last_login_at,omitempty"`
}

func (u *User) ToPublic() *PublicUser {
	pu := &PublicUser{
		ID:                  u.ID,
		Name:                u.Name,
		Phone:               u.Phone,
		Email:               u.Email,
		IsEmailVerified:     u.IsEmailVerified,
		OnboardingCompleted: u.OnboardingCompleted,
		CreatedAt:           u.CreatedAt,
	}

	if u.Username.Valid {
		pu.Username = &u.Username.String
	}
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		pu.LastLoginAt = &t
	}

	return pu
}

func (u *User) HasUsername() bool {
	return u.Username.Valid && u.Username.String != ""
}

func (u *User) HasPin() bool {
	return u.PinHash.Valid && u.PinHash.String != ""
}

func (u *User) IsLocked() bool {
	return u.LockedUntil.Valid && u.LockedUntil.Time.After(time.Now())
}

// ==============================================
// LOGIN SESSION MODEL
// ==============================================

type LoginSession struct {
	ID         int32            `db:"id"`
	UserID     int32            `db:"user_id"`
	Token      string           `db:"token"`
	DeviceInfo pgtype.Text      `db:"device_info"`
	IPAddress  pgtype.Text      `db:"ip_address"`
	ExpiresAt  time.Time        `db:"expires_at"`
	RevokedAt  pgtype.Timestamp `db:"revoked_at"`
	CreatedAt  time.Time        `db:"created_at"`
}

func (s *LoginSession) IsValid() bool {
	return !s.RevokedAt.Valid && time.Now().Before(s.ExpiresAt)
}

// ==============================================
// AUDIT LOG MODEL
// ==============================================

type AuditLog struct {
	ID         int64            `db:"id"`
	UserID     pgtype.Int4      `db:"user_id"`
	Action     string           `db:"action"`
	EntityType pgtype.Text      `db:"entity_type"`
	EntityID   pgtype.Int8      `db:"entity_id"`
	Metadata   pgtype.Text      `db:"metadata"`
	IPAddress  pgtype.Text      `db:"ip_address"`
	UserAgent  pgtype.Text      `db:"user_agent"`
	CreatedAt  time.Time        `db:"created_at"`
}

// ==============================================
// AUDIT ACTION CONSTANTS
// ==============================================
const (
	AuditActionLogin           = "login"
	AuditActionLoginFailed     = "login_failed"
	AuditActionLogout          = "logout"
	AuditActionPasswordChange  = "password_change"
	AuditActionPinChange       = "pin_change"
	AuditActionTransfer        = "transfer"
	AuditActionOTPSent         = "otp_sent"
	AuditActionOTPVerified     = "otp_verified"
	AuditActionAccountLocked   = "account_locked"
	AuditActionAccountUnlocked = "account_unlocked"
	AuditActionSettingsChanged = "settings_changed"
)
