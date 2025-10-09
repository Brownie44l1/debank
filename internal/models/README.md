# Models Layer

This layer contains **database models** that map directly to PostgreSQL tables.

## Purpose
- Map database tables to Go structs
- Define database-level constraints and relationships
- Provide helper methods for model operations

## Key Models

### Core Models
- **`user.go`** - User accounts (phone-first auth)
- **`wallet.go`** - Accounts (user wallets + system accounts)
- **`transaction.go`** - Transactions and postings (double-entry ledger)
- **`verification.go`** - OTP/email verification codes

### Supporting Models
- **`error.go`** - Custom error types and codes

## Important Notes

### Using pgx Types
We use `pgx/v5/pgtype` for nullable fields instead of `database/sql`:
```go
Username pgtype.Text           // Nullable string
UserID   pgtype.Int4           // Nullable int32
LockedUntil pgtype.Timestamptz // Nullable timestamp