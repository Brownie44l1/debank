package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Brownie44l1/debank/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ==============================================
// ERRORS
// ==============================================

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// ==============================================
// USER REPOSITORY
// ==============================================

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// ==============================================
// CREATE USER
// ==============================================

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (name, phone, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.Name,
		user.Phone,
		user.Email,
		user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// ==============================================
// GET USER (Read Operations)
// ==============================================

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password_hash, username, pin_hash,
		       is_email_verified, is_active, onboarding_completed,
		       failed_login_attempts, locked_until,
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.PinHash,
		&user.IsEmailVerified,
		&user.IsActive,
		&user.OnboardingCompleted,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByPhone retrieves a user by phone number
func (r *UserRepository) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password_hash, username, pin_hash,
		       is_email_verified, is_active, onboarding_completed,
		       failed_login_attempts, locked_until,
		       created_at, updated_at, last_login_at
		FROM users
		WHERE phone = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.PinHash,
		&user.IsEmailVerified,
		&user.IsActive,
		&user.OnboardingCompleted,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password_hash, username, pin_hash,
		       is_email_verified, is_active, onboarding_completed,
		       failed_login_attempts, locked_until,
		       created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.PinHash,
		&user.IsEmailVerified,
		&user.IsActive,
		&user.OnboardingCompleted,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password_hash, username, pin_hash,
		       is_email_verified, is_active, onboarding_completed,
		       failed_login_attempts, locked_until,
		       created_at, updated_at, last_login_at
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.PinHash,
		&user.IsEmailVerified,
		&user.IsActive,
		&user.OnboardingCompleted,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// ==============================================
// UPDATE USER
// ==============================================

// SetUsername sets the username for a user (onboarding step)
func (r *UserRepository) SetUsername(ctx context.Context, userID int, username string) error {
	query := `
		UPDATE users
		SET username = $1, updated_at = now()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, username, userID)
	if err != nil {
		return fmt.Errorf("failed to set username: %w", err)
	}

	return nil
}

// SetPin sets the transaction PIN for a user
func (r *UserRepository) SetPin(ctx context.Context, userID int, pinHash string) error {
	query := `
		UPDATE users
		SET pin_hash = $1, updated_at = now()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, pinHash, userID)
	if err != nil {
		return fmt.Errorf("failed to set PIN: %w", err)
	}

	return nil
}

// CompleteOnboarding marks user onboarding as complete
func (r *UserRepository) CompleteOnboarding(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET onboarding_completed = true, updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to complete onboarding: %w", err)
	}

	return nil
}

// VerifyEmail marks user's email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET is_email_verified = true, updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// UpdatePassword updates user's password hash
func (r *UserRepository) UpdatePassword(ctx context.Context, userID int, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET last_login_at = now(),
		    failed_login_attempts = 0,
		    locked_until = NULL,
		    updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// IncrementFailedLogins increments failed login attempts
func (r *UserRepository) IncrementFailedLogins(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment failed logins: %w", err)
	}

	return nil
}

// LockAccount locks a user account until specified time
func (r *UserRepository) LockAccount(ctx context.Context, userID int, until time.Time) error {
	query := `
		UPDATE users
		SET locked_until = $1, updated_at = now()
		WHERE id = $2
	`

	lockedUntil := pgtype.Timestamptz{Time: until, Valid: true}
	_, err := r.db.Exec(ctx, query, lockedUntil, userID)
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}

	return nil
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET locked_until = NULL,
		    failed_login_attempts = 0,
		    updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}

	return nil
}

// ==============================================
// USERNAME AVAILABILITY
// ==============================================

// IsUsernameAvailable checks if a username is available
func (r *UserRepository) IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	query := `SELECT is_username_available($1)`

	var available bool
	err := r.db.QueryRow(ctx, query, username).Scan(&available)
	if err != nil {
		return false, fmt.Errorf("failed to check username availability: %w", err)
	}

	return available, nil
}

// SuggestUsernames generates username suggestions based on a base username
func (r *UserRepository) SuggestUsernames(ctx context.Context, baseUsername string, limit int) ([]string, error) {
	query := `SELECT suggestion FROM suggest_usernames($1, $2)`

	rows, err := r.db.Query(ctx, query, baseUsername, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest usernames: %w", err)
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var suggestion string
		if err := rows.Scan(&suggestion); err != nil {
			return nil, fmt.Errorf("failed to scan username suggestion: %w", err)
		}
		suggestions = append(suggestions, suggestion)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating username suggestions: %w", err)
	}

	return suggestions, nil
}