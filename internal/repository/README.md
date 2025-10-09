### 📁 `internal/repository/README.md`
```markdown
# Repository Layer

This layer handles **all database operations** - the only place where SQL queries live.

## Purpose
- Execute database queries
- Manage transactions (Begin, Commit, Rollback)
- Convert database rows to models
- Handle database-specific errors

## Key Repositories

### `user_repository.go`
User account operations:
- CreateUser, GetUserByPhone, GetUserByEmail, GetUserByUsername
- SetUsername, SetPin, VerifyEmail
- Login management (UpdateLastLogin, LockAccount, IncrementFailedLogins)

### `wallet_repository.go`
Wallet/account operations:
- GetAccountByUserID, GetAccountByAccountNumber
- Transaction and posting creation
- Account locking (FOR UPDATE queries for concurrency safety)

### `verification_repository.go`
OTP/verification operations:
- CreateOTP, GetLatestOTP, VerifyOTP
- Rate limiting (CanResendOTP, CountRecentOTPs)
- Cleanup (DeleteExpiredOTPs)

## Key Patterns

### Transaction Management
```go
tx, err := repo.BeginTx(ctx)
defer tx.Rollback(ctx) // Always rollback (no-op if committed)

// ... do work ...

tx.Commit(ctx)