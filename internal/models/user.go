package models

import (
	"database/sql"
	"time"
)

// ==============================================
// USER MODEL (Database mapping)
// ==============================================

// User represents a wallet user
type User struct {
	ID                  int            `db:"id"`
	Name                string         `db:"name"`
	Phone               string         `db:"phone"`     // PRIMARY login identifier
	Email               string         `db:"email"`     // For OTP verification only
	PasswordHash        string         `db:"password_hash"`
	Username            sql.NullString `db:"username"` // Optional, set after onboarding
	PinHash             sql.NullString `db:"pin_hash"` // Transaction PIN
	IsEmailVerified     bool           `db:"is_email_verified"`
	IsActive            bool           `db:"is_active"`
	OnboardingCompleted bool           `db:"onboarding_completed"`
	FailedLoginAttempts int            `db:"failed_login_attempts"`
	LockedUntil         sql.NullTime   `db:"locked_until"`
	CreatedAt           time.Time      `db:"created_at"`
	UpdatedAt           time.Time      `db:"updated_at"`
	LastLoginAt         sql.NullTime   `db:"last_login_at"`
}

// PublicUser is the safe version to return to clients (no sensitive fields)
type PublicUser struct {
	ID                  int        `json:"id"`
	Name                string     `json:"name"`
	Phone               string     `json:"phone"`
	Email               string     `json:"email"`
	Username            *string    `json:"username,omitempty"`
	IsEmailVerified     bool       `json:"is_email_verified"`
	OnboardingCompleted bool       `json:"onboarding_completed"`
	CreatedAt           time.Time  `json:"created_at"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
}

// ToPublic converts User to PublicUser (removes sensitive fields)
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

	// Handle nullable fields
	if u.Username.Valid {
		pu.Username = &u.Username.String
	}
	if u.LastLoginAt.Valid {
		pu.LastLoginAt = &u.LastLoginAt.Time
	}

	return pu
}

// HasUsername checks if user has set a username
func (u *User) HasUsername() bool {
	return u.Username.Valid && u.Username.String != ""
}

// HasPin checks if user has set a transaction PIN
func (u *User) HasPin() bool {
	return u.PinHash.Valid && u.PinHash.String != ""
}

// IsLocked checks if account is currently locked
func (u *User) IsLocked() bool {
	return u.LockedUntil.Valid && u.LockedUntil.Time.After(time.Now())
}

// ==============================================
// LOGIN SESSION MODEL
// ==============================================

// LoginSession represents an active user session
type LoginSession struct {
	ID         int            `db:"id"`
	UserID     int            `db:"user_id"`
	Token      string         `db:"token"`
	DeviceInfo sql.NullString `db:"device_info"` // JSON string
	IPAddress  sql.NullString `db:"ip_address"`
	ExpiresAt  time.Time      `db:"expires_at"`
	RevokedAt  sql.NullTime   `db:"revoked_at"`
	CreatedAt  time.Time      `db:"created_at"`
}

// IsValid checks if session is still valid (not expired or revoked)
func (s *LoginSession) IsValid() bool {
	return !s.RevokedAt.Valid && time.Now().Before(s.ExpiresAt)
}

// ==============================================
// AUDIT LOG MODEL
// ==============================================

// AuditLog represents a security/activity log entry
type AuditLog struct {
	ID         int64          `db:"id"`
	UserID     sql.NullInt32  `db:"user_id"`
	Action     string         `db:"action"`
	EntityType sql.NullString `db:"entity_type"`
	EntityID   sql.NullInt64  `db:"entity_id"`
	Metadata   sql.NullString `db:"metadata"` // JSON string
	IPAddress  sql.NullString `db:"ip_address"`
	UserAgent  sql.NullString `db:"user_agent"`
	CreatedAt  time.Time      `db:"created_at"`
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