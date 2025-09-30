-- 001_core_ledger.sql
BEGIN;

-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT UNIQUE,           -- optional external reference
    name TEXT NOT NULL,
    type TEXT NOT NULL,                -- e.g., 'user', 'system', 'reserve', 'fee'
    currency CHAR(3) NOT NULL,         -- ISO 4217 code
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transactions (logical operations)
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key TEXT UNIQUE,       -- client-supplied to ensure idempotency
    kind TEXT NOT NULL,                -- e.g., 'p2p', 'deposit', 'withdrawal'
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending','posted','failed'
    reference TEXT,                    -- optional human reference
    metadata JSONB,                    -- arbitrary metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Postings (double-entry lines)
CREATE TABLE IF NOT EXISTS postings (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,            -- integer in smallest currency unit (e.g., cents). Positive or negative.
    currency CHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ensure postings use same currency as account (optional, if enforcing)
-- Note: cross-currency ledgers require a different design; keep single-currency per account for simplicity.

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_postings_account_id_created_at ON postings(account_id, created_at);
CREATE INDEX IF NOT EXISTS idx_postings_transaction_id ON postings(transaction_id);

COMMIT;
