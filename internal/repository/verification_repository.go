package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ==============================================
// ERRORS
// ==============================================

var (
	ErrOTPNotFound = errors.New("OTP not found")
	ErrOTPExpired  = errors.New("OTP has expired")
	ErrOTPUsed     = errors.New("OTP already used")
)

// ==============================================
// VERIFICATION REPOSITORY
// ==============================================

type VerificationRepository struct {
	db *pgxpool.Pool
}

func NewVerificationRepository(db *pgxpool.Pool) *VerificationRepository {
	return &VerificationRepository{db: db}
}

// ==============================================
// CREATE OTP
// ==============================================

func (r *VerificationRepository) CreateOTP(ctx context.Context, otp *models.VerificationCode) error {
	query := `
		INSERT INTO verification_codes (
			user_id, email, code, purpose, expires_at, ip_address
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	row := r.db.QueryRow(ctx, query,
		otp.UserID,
		otp.Email,
		otp.Code,
		otp.Purpose,
		otp.ExpiresAt,
		otp.IPAddress,
	)

	if err := row.Scan(&otp.ID, &otp.CreatedAt); err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	return nil
}

// ==============================================
// GET OTP
// ==============================================

func (r *VerificationRepository) GetLatestOTP(ctx context.Context, email, purpose string) (*models.VerificationCode, error) {
	query := `
		SELECT id, user_id, email, code, purpose, expires_at, used_at, attempts, ip_address, created_at
		FROM verification_codes
		WHERE email = $1 AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var otp models.VerificationCode
	err := r.db.QueryRow(ctx, query, email, purpose).Scan(
		&otp.ID,
		&otp.UserID,
		&otp.Email,
		&otp.Code,
		&otp.Purpose,
		&otp.ExpiresAt,
		&otp.UsedAt,
		&otp.Attempts,
		&otp.IPAddress,
		&otp.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, fmt.Errorf("failed to get OTP: %w", err)
	}

	return &otp, nil
}

func (r *VerificationRepository) GetOTPByCode(ctx context.Context, email, code, purpose string) (*models.VerificationCode, error) {
	query := `
		SELECT id, user_id, email, code, purpose, expires_at, used_at, attempts, ip_address, created_at
		FROM verification_codes
		WHERE email = $1 AND code = $2 AND purpose = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var otp models.VerificationCode
	err := r.db.QueryRow(ctx, query, email, code, purpose).Scan(
		&otp.ID,
		&otp.UserID,
		&otp.Email,
		&otp.Code,
		&otp.Purpose,
		&otp.ExpiresAt,
		&otp.UsedAt,
		&otp.Attempts,
		&otp.IPAddress,
		&otp.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, fmt.Errorf("failed to get OTP by code: %w", err)
	}

	return &otp, nil
}

// ==============================================
// VERIFY OTP
// ==============================================

func (r *VerificationRepository) VerifyOTP(ctx context.Context, email, code, purpose string) (bool, error) {
	query := `SELECT verify_email_otp($1, $2, $3)`

	var valid bool
	err := r.db.QueryRow(ctx, query, email, code, purpose).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("failed to verify OTP: %w", err)
	}

	return valid, nil
}

func (r *VerificationRepository) MarkOTPAsUsed(ctx context.Context, otpID int64) error {
	query := `
		UPDATE verification_codes
		SET used_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, otpID)
	if err != nil {
		return fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	return nil
}

func (r *VerificationRepository) IncrementOTPAttempts(ctx context.Context, otpID int64) error {
	query := `
		UPDATE verification_codes
		SET attempts = attempts + 1
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, otpID)
	if err != nil {
		return fmt.Errorf("failed to increment OTP attempts: %w", err)
	}

	return nil
}

// ==============================================
// OTP CHECKS
// ==============================================

func (r *VerificationRepository) CanResendOTP(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error) {
	query := `
		SELECT created_at
		FROM verification_codes
		WHERE email = $1 AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var lastCreated time.Time
	err := r.db.QueryRow(ctx, query, email, purpose).Scan(&lastCreated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("failed to check resend eligibility: %w", err)
	}

	return time.Since(lastCreated) >= cooldown, nil
}

func (r *VerificationRepository) CountRecentOTPs(ctx context.Context, email string, since time.Duration) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM verification_codes
		WHERE email = $1 AND created_at > $2
	`

	sinceTime := time.Now().Add(-since)
	var count int
	err := r.db.QueryRow(ctx, query, email, sinceTime).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count recent OTPs: %w", err)
	}

	return count, nil
}

// ==============================================
// CLEANUP
// ==============================================

func (r *VerificationRepository) DeleteExpiredOTPs(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM verification_codes
		WHERE expires_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	tag, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired OTPs: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *VerificationRepository) DeleteUsedOTPs(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM verification_codes
		WHERE used_at IS NOT NULL AND used_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	tag, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete used OTPs: %w", err)
	}

	return tag.RowsAffected(), nil
}