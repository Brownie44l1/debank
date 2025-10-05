# Bank Ledger Database

A double-entry accounting ledger system for managing user wallets and transactions.

## 📁 Folder Structure

```
db/
├── README.md                     # This file
├── schema/
│   ├── 001_schema.sql            # Table definitions (users, accounts, transactions, postings)
│   └── 002_functions.sql         # Functions and triggers (double-entry, balance updates)
├── seeds/
│   ├── 001_system_accounts.sql   # System accounts (Reserve, Fee) - PRODUCTION
│   └── 002_test_data.sql         # Test users and data - DEV ONLY
└── scripts/
    ├── setup.sh                  # Complete setup script
    └── reset.sh                  # Drop and recreate (coming soon)
```

## 🚀 Quick Start

### Setup Database

```bash
# Set your database password
export DB_PASSWORD=your_password

# Run complete setup (dev environment with test data)
cd internal/db/scripts
./setup.sh

# For production (no test data)
./setup.sh --prod
```

### Manual Setup

If you prefer to run SQL files manually:

```bash
psql -U postgres -h localhost -d bank_ledger -f internal/db/schema/001_schema.sql
psql -U postgres -h localhost -d bank_ledger -f internal/db/schema/002_functions.sql
psql -U postgres -h localhost -d bank_ledger -f internal/db/seeds/001_system_accounts.sql
psql -U postgres -h localhost -d bank_ledger -f internal/db/seeds/002_test_data.sql  # dev only
```

## 📊 Database Design

### Core Concepts

This is a **double-entry ledger** system where:
- Every transaction has 2+ postings (debits and credits)
- All postings for a transaction must sum to **zero**
- Account balances are **automatically maintained** by triggers

### Tables

#### 1. **users**
End users who own wallets
- `id`: Primary key
- `email`: Unique identifier
- `name`: Display name

#### 2. **accounts**
All accounts in the system (user wallets + system accounts)
- `id`: Primary key
- `external_id`: External reference (e.g., 'user_123', 'sys_reserve')
- `type`: 'user', 'system', 'reserve', 'fee'
- `balance`: Current balance in kobo (auto-updated by trigger)
- `user_id`: Links to users table (NULL for system accounts)

#### 3. **transactions**
Logical grouping of postings
- `id`: Primary key
- `idempotency_key`: Prevents duplicate processing
- `kind`: 'p2p', 'deposit', 'withdrawal', 'fee'
- `status`: 'pending', 'posted', 'failed'

#### 4. **postings**
Individual debit/credit entries (implements double-entry)
- `id`: Primary key
- `transaction_id`: Links to transaction
- `account_id`: Links to account
- `amount`: Positive (credit) or negative (debit) in kobo

### Example: P2P Transfer

Alice sends ₦1,000 to Bob with ₦50 fee:

```sql
-- Transaction
INSERT INTO transactions (idempotency_key, kind, status) 
VALUES ('txn_abc123', 'p2p', 'posted');

-- Postings (must sum to zero)
INSERT INTO postings (transaction_id, account_id, amount) VALUES
(1, alice_account_id, -100000),  -- Alice debited ₦1,000
(1, bob_account_id, 95000),      -- Bob credited ₦950
(1, fee_account_id, 5000);       -- Fee account credited ₦50
-- Sum: -100000 + 95000 + 5000 = 0 ✓
```

### Currency

- All amounts stored in **kobo** (smallest unit)
- 1 NGN = 100 kobo
- Example: ₦1,000 = 100,000 kobo

### System Accounts

- **Reserve Account** (`sys_reserve`): Holds system funds, used for funding
- **Fee Account** (`sys_fee`): Collects transaction fees

## 🔍 Common Queries

### Check account balance
```sql
SELECT name, balance, currency FROM accounts WHERE id = 1;
```

### Get account statement
```sql
SELECT 
    t.created_at,
    t.kind,
    t.reference,
    p.amount,
    p.currency
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
ORDER BY t.created_at DESC;
```

### Verify transaction balances (should always be 0)
```sql
SELECT 
    t.id,
    t.idempotency_key,
    SUM(p.amount) as balance
FROM transactions t
JOIN postings p ON p.transaction_id = t.id
GROUP BY t.id
HAVING SUM(p.amount) != 0;  -- Should return no rows
```

## ✅ Data Integrity Rules

1. **Double-entry enforcement**: All postings in a transaction must sum to zero (enforced by trigger)
2. **Balance auto-update**: Account balances update automatically when postings are inserted
3. **Idempotency**: Use `idempotency_key` to prevent duplicate transactions
4. **Referential integrity**: Cannot delete accounts with postings (ON DELETE RESTRICT)

## 🧪 Test Data

The `002_test_data.sql` seed includes:
- 3 test users: Alice, Bob, Charlie
- Each user has a wallet with ₦1,000 (100,000 kobo)
- Reserve account debited -₦3,000 to fund test users

⚠️ **Never run test data in production!**

## 🔧 Maintenance

### Reset database (coming soon)
```bash
./internal/db/scripts/reset.sh
```

### Add new test scenarios
Edit `seeds/002_test_data.sql` to add more test users or transactions

### Backup database (not yet deployed)
```bash
pg_dump -U postgres bank_ledger > backup_$(date +%Y%m%d).sql
```

## 📝 Notes

- All timestamps are in UTC (`TIMESTAMPTZ`)
- Indexes optimized for common queries (account statements, transaction details)
- Comments added to tables and columns for documentation
- Files numbered to show execution order (001, 002, etc.)