### 📁 `internal/api/dto/README.md`
```markdown
# DTO (Data Transfer Object) Layer

This layer defines the **API contract** - what data flows in and out of your API endpoints.

## Purpose
- Define request structures (what clients send)
- Define response structures (what API returns)
- Validation rules using struct tags
- Type conversion and data transformation

## Key DTOs

### Auth DTOs (`auth_dto.go`)
- **SignupRequest** - User registration (phone, email, password)
- **LoginRequest** - User login (phone/username + password)
- **VerifyEmailRequest** - Email OTP verification
- **CompleteOnboardingRequest** - Set username + PIN

### Wallet DTOs (`wallet_dto.go`)
- **DepositRequest** - Deposit money
- **WithdrawRequest** - Withdraw money
- **TransferRequest** - P2P transfer (username/phone/account_number)
- **BalanceResponse** - Account balance

### User DTOs (`user_dto.go`)
- **PublicUserDTO** - Safe user info (no passwords/pins)
- **UpdateProfileRequest** - Update user details

## Validation Tags

We use Gin's binding validation:
```go
type SignupRequest struct {
    Phone    string `json:"phone" binding:"required,e164"`       // E.164 format
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Pin      string `json:"pin" binding:"required,len=4,numeric"`
}