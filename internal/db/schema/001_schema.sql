-- ============================================
-- SCHEMA: USERS + WALLET LEDGER (Phone-first Auth)
-- ============================================
-- Email: For OTP verification only (free)
-- Phone: Primary login identifier (no SMS cost)
-- ============================================

BEGIN;

-- ============================================
-- USERS TABLE
-- ============================================
DROP TABLE IF EXISTS users CASCADE;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT UNIQUE NOT NULL,           -- PRIMARY login identifier
    email TEXT UNIQUE NOT NULL,           -- For OTP verification only
    password_hash TEXT NOT NULL,          -- Used with phone/username for login
    username TEXT UNIQUE,                 -- Optional, set after onboarding
    pin_hash TEXT,                        -- 4-6 digit transaction PIN
    
    -- Verification status
    is_email_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    onboarding_completed BOOLEAN DEFAULT FALSE,
    
    -- Security
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMPTZ,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    last_login_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT phone_format CHECK (phone ~ '^\+?[0-9]{10,15}$'),
    CONSTRAINT username_format CHECK (username IS NULL OR username ~ '^[a-zA-Z0-9_]{3,20}$')
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_username ON users(username) WHERE username IS NOT NULL;
CREATE INDEX idx_users_email ON users(email);

-- ============================================
-- VERIFICATION CODES TABLE
-- ============================================
DROP TABLE IF EXISTS verification_codes CASCADE;

CREATE TABLE verification_codes (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,                  -- Always email-based (free OTP)
    code TEXT NOT NULL,                   -- 6-digit OTP
    purpose TEXT NOT NULL,                -- What this OTP is for
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    attempts INT DEFAULT 0,
    ip_address TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT valid_purpose CHECK (purpose IN (
        'email_verify',
        'password_reset',
        'transaction_auth',
        'settings_change',
        'login_mfa'
    ))
);

CREATE INDEX idx_verification_codes_user_id ON verification_codes(user_id);
CREATE INDEX idx_verification_codes_email ON verification_codes(email);
CREATE INDEX idx_verification_codes_expires ON verification_codes(expires_at) WHERE used_at IS NULL;

-- ============================================
-- ACCOUNTS TABLE
-- ============================================
DROP TABLE IF EXISTS accounts CASCADE;

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    account_number TEXT UNIQUE NOT NULL,  -- Generated from phone
    external_id TEXT UNIQUE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,    -- In kobo (1 NGN = 100 kobo)
    currency CHAR(3) NOT NULL DEFAULT 'NGN',
    
    -- User relationship
    user_id INT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Bank integration (for interbank transfers)
    bank_code TEXT,
    bank_name TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    frozen_at TIMESTAMPTZ,
    frozen_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT valid_account_type CHECK (type IN ('user', 'system', 'reserve', 'fee')),
    CONSTRAINT valid_balance CHECK (balance >= 0 OR type IN ('system', 'reserve'))
);

CREATE INDEX idx_accounts_account_number ON accounts(account_number);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_type ON accounts(type);

-- ============================================
-- TRANSACTIONS TABLE
-- ============================================
DROP TABLE IF EXISTS transactions CASCADE;

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key TEXT UNIQUE NOT NULL,
    reference TEXT UNIQUE NOT NULL,
    
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    amount BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'NGN',
    
    from_account_id BIGINT REFERENCES accounts(id),
    to_account_id BIGINT REFERENCES accounts(id),
    from_identifier TEXT,
    to_identifier TEXT,
    
    description TEXT,
    metadata JSONB,
    
    created_at TIMESTAMPTZ DEFAULT now(),
    posted_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,
    
    CONSTRAINT valid_kind CHECK (kind IN ('p2p', 'deposit', 'withdrawal', 'fee', 'interbank', 'refund')),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'posted', 'failed', 'reversed'))
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id);
CREATE INDEX idx_transactions_reference ON transactions(reference);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status) WHERE status = 'pending';

-- ============================================
-- POSTINGS TABLE
-- ============================================
DROP TABLE IF EXISTS postings CASCADE;

CREATE TABLE postings (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'NGN',
    created_at TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT non_zero_amount CHECK (amount != 0)
);

CREATE INDEX idx_postings_account_id_created_at ON postings(account_id, created_at DESC);
CREATE INDEX idx_postings_transaction_id ON postings(transaction_id);

-- ============================================
-- LOGIN SESSIONS TABLE
-- ============================================
DROP TABLE IF EXISTS login_sessions CASCADE;

CREATE TABLE login_sessions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    device_info JSONB,
    ip_address TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_login_sessions_user_id ON login_sessions(user_id);
CREATE INDEX idx_login_sessions_token ON login_sessions(token) WHERE revoked_at IS NULL;

-- ============================================
-- AUDIT LOGS TABLE
-- ============================================
DROP TABLE IF EXISTS audit_logs CASCADE;

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id BIGINT,
    metadata JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

COMMIT;

\echo '=== Functions and triggers created successfully ==='