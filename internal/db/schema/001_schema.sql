-- ============================================
-- BANK LEDGER SCHEMA
-- ============================================
-- Double-entry accounting system for mobile wallet
-- All amounts in kobo (1 NGN = 100 kobo)
-- Balance maintained automatically via triggers
-- ============================================

BEGIN;

-- ============================================
-- USERS
-- ============================================
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================
-- ACCOUNTS
-- ============================================
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT UNIQUE,           -- e.g. 'user_123', 'sys_reserve'
    name TEXT NOT NULL,
    type TEXT NOT NULL,                -- 'user', 'system', 'reserve', 'fee'
    balance BIGINT NOT NULL DEFAULT 0, -- in kobo, auto-updated
    currency CHAR(3) NOT NULL,         -- only 'NGN' for now
    user_id INT UNIQUE REFERENCES users(id), -- NULL for system accounts
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================
-- TRANSACTIONS
-- ============================================
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key TEXT UNIQUE,       -- prevents duplicate processing
    kind TEXT NOT NULL,                -- 'p2p', 'deposit', 'withdrawal', 'fee'
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'posted', 'failed'
    reference TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================
-- POSTINGS (Double-Entry Lines)
-- ============================================
-- All postings for a transaction MUST sum to zero
-- Positive = credit, Negative = debit
CREATE TABLE postings (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,            -- can be positive or negative
    currency CHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================
-- INDEXES
-- ============================================
CREATE INDEX idx_postings_account_id_created_at ON postings(account_id, created_at);
CREATE INDEX idx_postings_transaction_id ON postings(transaction_id);
CREATE INDEX idx_accounts_external_id ON accounts(external_id);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);

COMMIT;

-- ============================================
-- VERIFICATION
-- ============================================
\echo '=== Schema (tables and indexes) created successfully ==='
\df enforce_double_entry
\df update_account_balance_on_posting
\echo '=== Run seeds/002_functions.sql next ==='