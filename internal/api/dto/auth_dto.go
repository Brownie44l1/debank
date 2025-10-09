package dto

// ==============================================
// AUTH REQUEST DTOs
// ==============================================

// SignupRequest - Phone-first registration
type SignupRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Phone    string `json:"phone" binding:"required"`       // Will validate with custom validator
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// VerifyEmailRequest - Email OTP verification
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

// LoginRequest - Phone or Username + Password (NO EMAIL)
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // phone or username
	Password   string `json:"password" binding:"required"`
}

// CompleteOnboardingRequest - Set username and PIN after email verification
type CompleteOnboardingRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20,alphanum"`
	Pin      string `json:"pin" binding:"required,len=4,numeric"`
}

// ResendOTPRequest
type ResendOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=email_verify password_reset transaction_auth"`
}

// ForgotPasswordRequest - Initiate password reset via email OTP
type ForgotPasswordRequest struct {
	Phone string `json:"phone" binding:"required"` // User provides phone, we send OTP to email
}

// ResetPasswordRequest - Complete password reset with OTP
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6,numeric"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// ChangePasswordRequest - User is logged in
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
}

// SetPinRequest - Set or update transaction PIN
type SetPinRequest struct {
	Pin        string `json:"pin" binding:"required,len=4,numeric"`
	ConfirmPin string `json:"confirm_pin" binding:"required,len=4,numeric,eqfield=Pin"`
}

// ValidatePinRequest - For transaction authorization
type ValidatePinRequest struct {
	Pin string `json:"pin" binding:"required,len=4,numeric"`
}

// LogoutRequest
type LogoutRequest struct {
	Token string `json:"token,omitempty"` // Optional: logout specific session
}

// ==============================================
// AUTH RESPONSE DTOs
// ==============================================

// SignupResponse - Returns user info + instructions
type SignupResponse struct {
	User     *UserDTO `json:"user"`
	Message  string   `json:"message"`
	NextStep string   `json:"next_step"` // "verify_email"
}

// VerifyEmailResponse
type VerifyEmailResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	NextStep string `json:"next_step,omitempty"` // "complete_onboarding"
}

// LoginResponse
type LoginResponse struct {
	User         *UserDTO `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	ExpiresIn    int      `json:"expires_in"` // seconds
	TokenType    string   `json:"token_type"` // "Bearer"
}

// CompleteOnboardingResponse
type CompleteOnboardingResponse struct {
	User    *UserDTO    `json:"user"`
	Account *AccountDTO `json:"account"` // Include account number
	Message string      `json:"message"`
}

// ResendOTPResponse
type ResendOTPResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"` // seconds until OTP expires
}

// ForgotPasswordResponse
type ForgotPasswordResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"` // Masked: "j***@example.com"
}

// ResetPasswordResponse
type ResetPasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ChangePasswordResponse
type ChangePasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SetPinResponse
type SetPinResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// LogoutResponse
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ==============================================
// SUPPORTING DTOs
// ==============================================

// UserDTO - Safe user representation
type UserDTO struct {
	ID                  int     `json:"id"`
	Name                string  `json:"name"`
	Phone               string  `json:"phone"`
	Email               string  `json:"email"`
	Username            *string `json:"username,omitempty"`
	IsEmailVerified     bool    `json:"is_email_verified"`
	OnboardingCompleted bool    `json:"onboarding_completed"`
	CreatedAt           string  `json:"created_at"` // ISO 8601
}

// AccountDTO - Wallet account info
type AccountDTO struct {
	ID            int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Balance       int64  `json:"balance"` // In kobo
	BalanceNGN    float64 `json:"balance_ngn"` // In Naira
	Currency      string `json:"currency"`
}