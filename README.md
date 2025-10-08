# Debank - Digital Wallet System

A production-ready digital wallet system built with Go, implementing double-entry accounting principles for financial transactions.

## 🏗️ Architecture

```
debank/
├── cmd/                    # Application entrypoints
├── internal/              # Private application code
│   ├── api/              # API layer (NEW - HTTP concerns)
│   ├── auth/             # Authentication & authorization (NEW)
│   ├── config/           # Configuration management
│   ├── db/               # Database setup & migrations
│   ├── domain/           # Business domain models (NEW)
│   ├── repository/       # Data access layer
│   ├── service/          # Business logic layer
│   └── worker/           # Background jobs
├── pkg/                  # Public libraries (NEW)
└── tests/                # Integration tests
```

## 🎯 Design Principles

1. **Clean Architecture**: Dependency inversion - domain doesn't depend on infrastructure
2. **Double-Entry Accounting**: All transactions maintain zero-sum integrity
3. **Concurrency Safe**: Row-level locking prevents race conditions
4. **Idempotency**: Duplicate requests are safely handled
5. **ACID Compliance**: Database transactions ensure data consistency

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Docker & Docker Compose (optional)

### Setup

```bash
# Clone and setup
git clone <repo>
cd debank

# Copy environment file
cp .env.example .env

# Start database
docker-compose up -d postgres

# Run migrations
psql -U postgres -d debank -f internal/db/schema/001_init.sql
psql -U postgres -d debank -f internal/db/seeds/001_system_accounts.sql

# Run tests
./run_tests.sh

# Start server
go run cmd/server/main.go
```

### API Endpoints

```bash
# Health check
GET /api/v1/health

# Get balance
GET /api/v1/balance/:user_id

# Transfer money
POST /api/v1/transfer
{
  "from_user_id": 1,
  "to_user_id": 2,
  "amount": 100000,
  "idempotency_key": "unique-key-123"
}

# Transaction history
GET /api/v1/transactions/:user_id?page=1&per_page=20
```

## 📁 Project Structure Details

### `/cmd` - Application Entrypoints
Contains `main.go` files for different binaries (server, worker, CLI tools)

### `/internal` - Private Code
- **api/**: HTTP handlers, middleware, routing
- **auth/**: JWT, sessions, user authentication
- **domain/**: Core business models and interfaces
- **repository/**: Database queries and data access
- **service/**: Business logic and orchestration
- **worker/**: Background jobs (notifications, reconciliation)

### `/pkg` - Reusable Libraries
Utilities that could be extracted to separate packages

### `/tests` - Integration Tests
End-to-end and integration test suites

## 🔒 Security Features

- JWT-based authentication (coming in Phase 1)
- Transaction PIN verification
- Idempotency keys for duplicate prevention
- Row-level locking for concurrency
- Input validation at all layers

## 🧪 Testing

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./tests/...

# With coverage
go test -cover ./...

# Concurrency test
go run test_concurrency.go
```

## 📊 Monitoring

- Health check endpoint: `/api/v1/health`
- Structured logging with timestamps
- Transaction timing metrics
- Balance integrity checks

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL 14+
- **Driver**: pgx/v5 (PostgreSQL driver)
- **Testing**: Go testing package + testify

## 📈 Roadmap

- [x] Phase 0: Core wallet operations
- [ ] Phase 1: Authentication & user management
- [ ] Phase 2: Paystack integration (deposits/withdrawals)
- [ ] Phase 3: Inter-bank transfers
- [ ] Phase 4: Advanced features (QR codes, recurring transfers)

## 📝 License

MIT

## 👥 Contributing

See CONTRIBUTING.md for development guidelines