package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Brownie44l1/debank/internal/api/dto"
	"github.com/Brownie44l1/debank/internal/auth"
	"github.com/Brownie44l1/debank/internal/models"
	"github.com/Brownie44l1/debank/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ==============================================
// AUTH SERVICE
// ==============================================

type AuthService struct {
	userRepo         *repository.UserRepository
	verificationRepo *repository.VerificationRepository
	walletRepo       *repository.WalletRepository
	emailService     *EmailService
	jwtSecret        string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	verificationRepo *repository.VerificationRepository,
	walletRepo *repository.WalletRepository,
	emailService *EmailService,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		verificationRepo: verificationRepo,
		walletRepo:       walletRepo,
		emailService:     emailService,
		jwtSecret:        jwtSecret,
	}
}

// ==============================================
// SIGNUP
// ==============================================

func (s *AuthService) Signup(ctx context.Context, req dto.SignupRequest) (*dto.SignupResponse, error) {
	// 1. Check if phone already exists
	existingUser, err := s.userRepo.GetUserByPhone(ctx, req.Phone)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check phone: %w", err)
	}
	if existingUser != nil {
		return nil, models.ErrPhoneAlreadyExists
	}

	// 2. Check if email already exists
	existingUser, err = s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, models.ErrEmailAlreadyExists
	}

	// 3. Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Create user
	user := &models.User{
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		PasswordHash: passwordHash,
		IsActive:     true,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 5. Send email verification OTP (async)
	go s.sendEmailVerificationOTP(context.Background(), user.Email, int(user.ID))

	// 6. Build response
	userDTO := s.userToDTO(user)

	return &dto.SignupResponse{
		User:     userDTO,
		Message:  "Account created successfully. Please check your email for verification code.",
		NextStep: "verify_email",
	}, nil
}

// ==============================================
// EMAIL VERIFICATION
// ==============================================

func (s *AuthService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (*dto.VerifyEmailResponse, error) {
	// 1. Verify OTP
	valid, err := s.verificationRepo.VerifyOTP(ctx, req.Email, req.Code, models.OTPPurposeEmailVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to verify OTP: %w", err)
	}

	if !valid {
		return nil, models.ErrOTPInvalid
	}

	// 2. Get user by email
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 3. Mark email as verified
	if err := s.userRepo.VerifyEmail(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to mark email as verified: %w", err)
	}

	return &dto.VerifyEmailResponse{
		Success:  true,
		Message:  "Email verified successfully. Please complete your profile setup.",
		NextStep: "complete_onboarding",
	}, nil
}

// ==============================================
// RESEND OTP
// ==============================================

func (s *AuthService) ResendOTP(ctx context.Context, req dto.ResendOTPRequest) (*dto.ResendOTPResponse, error) {
	// 1. Check cooldown
	canResend, err := s.verificationRepo.CanResendOTP(ctx, req.Email, req.Purpose, models.OTPResendCooldown)
	if err != nil {
		return nil, fmt.Errorf("failed to check resend eligibility: %w", err)
	}

	if !canResend {
		return nil, models.ErrOTPResendCooldown
	}

	// 2. Rate limit check (max 5 OTPs per hour)
	recentCount, err := s.verificationRepo.CountRecentOTPs(ctx, req.Email, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if recentCount >= 5 {
		return nil, errors.New("too many OTP requests, please try again later")
	}

	// 3. Get user (if purpose requires it)
	var userID *int
	if req.Purpose == models.OTPPurposePasswordReset {
		user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
		if err != nil {
			return nil, models.ErrUserNotFound
		}
		userID = &user.ID
	}

	// 4. Generate and send new OTP
	code := auth.GenerateOTP()
	expiresAt := time.Now().Add(time.Duration(models.OTPExpiryMinutes) * time.Minute)

	otp := &models.VerificationCode{
		Email:     req.Email,
		Code:      code,
		Purpose:   req.Purpose,
		ExpiresAt: expiresAt,
	}

	if userID != nil {
		otp.UserID = pgtype.Int4{Int32: int32(*userID), Valid: true}
	}

	if err := s.verificationRepo.CreateOTP(ctx, otp); err != nil {
		return nil, fmt.Errorf("failed to create OTP: %w", err)
	}

	// 5. Send email
	if err := s.emailService.SendOTP(req.Email, code, req.Purpose); err != nil {
		return nil, fmt.Errorf("failed to send OTP email: %w", err)
	}

	return &dto.ResendOTPResponse{
		Success:   true,
		Message:   "Verification code sent to your email",
		ExpiresIn: models.OTPExpiryMinutes * 60, // in seconds
	}, nil
}

// ==============================================
// COMPLETE ONBOARDING
// ==============================================

func (s *AuthService) CompleteOnboarding(ctx context.Context, userID int, req dto.CompleteOnboardingRequest) (*dto.CompleteOnboardingResponse, error) {
	// 1. Get user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 2. Check if email is verified
	if !user.IsEmailVerified {
		return nil, models.ErrEmailNotVerified
	}

	// 3. Check if username is available
	available, err := s.userRepo.IsUsernameAvailable(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	if !available {
		return nil, models.ErrUsernameAlreadyExists
	}

	// 4. Hash PIN
	pinHash, err := auth.HashPin(req.Pin)
	if err != nil {
		return nil, fmt.Errorf("failed to hash PIN: %w", err)
	}

	// 5. Set username
	if err := s.userRepo.SetUsername(ctx, userID, req.Username); err != nil {
		return nil, fmt.Errorf("failed to set username: %w", err)
	}

	// 6. Set PIN
	if err := s.userRepo.SetPin(ctx, userID, pinHash); err != nil {
		return nil, fmt.Errorf("failed to set PIN: %w", err)
	}

	// 7. Mark onboarding as complete
	if err := s.userRepo.CompleteOnboarding(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to complete onboarding: %w", err)
	}

	// 8. Get updated user and account
	user, err = s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	account, err := s.walletRepo.GetAccountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// 9. Build response
	userDTO := s.userToDTO(user)
	accountDTO := s.accountToDTO(account)

	return &dto.CompleteOnboardingResponse{
		User:    userDTO,
		Account: accountDTO,
		Message: "Onboarding completed successfully! You can now start using your wallet.",
	}, nil
}

// ==============================================
// LOGIN
// ==============================================

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	// 1. Determine identifier type and get user
	var user *models.User
	var err error

	identifier := strings.TrimSpace(req.Identifier)

	if strings.HasPrefix(identifier, "@") {
		// Username (remove @ prefix)
		username := strings.TrimPrefix(identifier, "@")
		user, err = s.userRepo.GetUserByUsername(ctx, username)
	} else if strings.HasPrefix(identifier, "+") || len(identifier) >= 10 {
		// Phone number
		user, err = s.userRepo.GetUserByPhone(ctx, identifier)
	} else {
		// Assume it's username without @
		user, err = s.userRepo.GetUserByUsername(ctx, identifier)
	}

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 2. Check if account is locked
	if user.IsLocked() {
		return nil, models.ErrAccountLocked
	}

	// 3. Check if account is active
	if !user.IsActive {
		return nil, models.ErrAccountInactive
	}

	// 4. Verify password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		// Increment failed login attempts
		_ = s.userRepo.IncrementFailedLogins(ctx, user.ID)

		// Lock account after 5 failed attempts
		if user.FailedLoginAttempts >= 4 { // Will be 5 after increment
			lockUntil := time.Now().Add(30 * time.Minute)
			_ = s.userRepo.LockAccount(ctx, user.ID, lockUntil)
			return nil, errors.New("account locked due to too many failed login attempts")
		}

		return nil, models.ErrInvalidCredentials
	}

	// 5. Update last login and reset failed attempts
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// 6. Generate JWT token
	token, expiresIn, err := auth.GenerateJWT(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 7. Build response
	userDTO := s.userToDTO(user)

	return &dto.LoginResponse{
		User:        userDTO,
		AccessToken: token,
		ExpiresIn:   expiresIn,
		TokenType:   "Bearer",
	}, nil
}

// ==============================================
// PASSWORD RESET
// ==============================================

func (s *AuthService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	// 1. Get user by phone
	user, err := s.userRepo.GetUserByPhone(ctx, req.Phone)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Don't reveal if user exists or not
			return &dto.ForgotPasswordResponse{
				Message: "If this phone number is registered, you'll receive a password reset code via email",
				Email:   "",
			}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 2. Generate OTP
	code := auth.GenerateOTP()
	expiresAt := time.Now().Add(time.Duration(models.OTPExpiryMinutes) * time.Minute)

	otp := &models.VerificationCode{
		UserID:    pgtype.Int4{Int32: int32(user.ID), Valid: true},
		Email:     user.Email,
		Code:      code,
		Purpose:   models.OTPPurposePasswordReset,
		ExpiresAt: expiresAt,
	}

	if err := s.verificationRepo.CreateOTP(ctx, otp); err != nil {
		return nil, fmt.Errorf("failed to create OTP: %w", err)
	}

	// 3. Send email
	if err := s.emailService.SendOTP(user.Email, code, models.OTPPurposePasswordReset); err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	// 4. Mask email
	maskedEmail := maskEmail(user.Email)

	return &dto.ForgotPasswordResponse{
		Message: "Password reset code sent to your email",
		Email:   maskedEmail,
	}, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error) {
	// 1. Verify OTP
	valid, err := s.verificationRepo.VerifyOTP(ctx, req.Email, req.Code, models.OTPPurposePasswordReset)
	if err != nil {
		return nil, fmt.Errorf("failed to verify OTP: %w", err)
	}

	if !valid {
		return nil, models.ErrOTPInvalid
	}

	// 2. Get user
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 3. Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Update password
	if err := s.userRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// 5. Unlock account if locked
	if user.IsLocked() {
		_ = s.userRepo.UnlockAccount(ctx, user.ID)
	}

	return &dto.ResetPasswordResponse{
		Success: true,
		Message: "Password reset successfully. You can now login with your new password.",
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error) {
	// 1. Get user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 2. Verify current password
	if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return nil, errors.New("current password is incorrect")
	}

	// 3. Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return &dto.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

// ==============================================
// PIN MANAGEMENT
// ==============================================

func (s *AuthService) SetPin(ctx context.Context, userID int, req dto.SetPinRequest) (*dto.SetPinResponse, error) {
	// Hash PIN
	pinHash, err := auth.HashPin(req.Pin)
	if err != nil {
		return nil, fmt.Errorf("failed to hash PIN: %w", err)
	}

	// Set PIN
	if err := s.userRepo.SetPin(ctx, userID, pinHash); err != nil {
		return nil, fmt.Errorf("failed to set PIN: %w", err)
	}

	return &dto.SetPinResponse{
		Success: true,
		Message: "Transaction PIN set successfully",
	}, nil
}

func (s *AuthService) ValidatePin(ctx context.Context, userID int, pin string) error {
	// Get user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if PIN is set
	if !user.HasPin() {
		return models.ErrPinNotSet
	}

	// Verify PIN
	if !auth.CheckPin(pin, user.PinHash.String) {
		return models.ErrIncorrectPin
	}

	return nil
}

// ==============================================
// HELPER FUNCTIONS
// ==============================================

func (s *AuthService) sendEmailVerificationOTP(ctx context.Context, email string, userID int) {
	code := auth.GenerateOTP()
	expiresAt := time.Now().Add(time.Duration(models.OTPExpiryMinutes) * time.Minute)

	otp := &models.VerificationCode{
		UserID:    pgtype.Int4{Int32: int32(userID), Valid: true},
		Email:     email,
		Code:      code,
		Purpose:   models.OTPPurposeEmailVerify,
		ExpiresAt: expiresAt,
	}

	if err := s.verificationRepo.CreateOTP(ctx, otp); err != nil {
		// Log error but don't fail signup
		fmt.Printf("Failed to create OTP: %v\n", err)
		return
	}

	if err := s.emailService.SendOTP(email, code, models.OTPPurposeEmailVerify); err != nil {
		fmt.Printf("Failed to send OTP email: %v\n", err)
	}
}

func (s *AuthService) userToDTO(user *models.User) *dto.UserDTO {
	userDTO := &dto.UserDTO{
		ID:                  user.ID,
		Name:                user.Name,
		Phone:               user.Phone,
		Email:               user.Email,
		IsEmailVerified:     user.IsEmailVerified,
		OnboardingCompleted: user.OnboardingCompleted,
		CreatedAt:           user.CreatedAt.Format(time.RFC3339),
	}

	if user.Username.Valid {
		username := user.Username.String
		userDTO.Username = &username
	}

	return userDTO
}

func (s *AuthService) accountToDTO(account *models.Account) *dto.AccountDTO {
	accountNumber := ""
	if account.AccountNumber.Valid {
		accountNumber = account.AccountNumber.String
	}

	return &dto.AccountDTO{
		ID:            account.ID,
		AccountNumber: accountNumber,
		Name:          account.Name,
		Balance:       account.Balance,
		BalanceNGN:    account.GetBalanceNGN(),
		Currency:      account.Currency,
	}
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return username[0:1] + "***@" + domain
	}

	return username[0:1] + "***" + username[len(username)-1:] + "@" + domain
}