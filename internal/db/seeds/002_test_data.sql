-- ============================================
-- TEST DATA SEED
-- ============================================
-- Creates sample users and wallets for development/testing
-- ⚠️ DO NOT RUN IN PRODUCTION ⚠️
--
-- This creates:
-- - 3 test users (Alice, Bob, Charlie)
-- - 3 user wallets linked to those users
-- - Initial funding from Reserve account (₦1,000 each = 100,000 kobo)
-- ============================================

BEGIN;

-- ============================================
-- 1. CREATE TEST USERS
-- ============================================
INSERT INTO users (name, email) VALUES
('Alice', 'alice@example.com'),
('Bob', 'bob@example.com'),
('Charlie', 'charlie@example.com')
ON CONFLICT (email) DO NOTHING;

-- ============================================
-- 2. CREATE USER WALLETS
-- ============================================
INSERT INTO accounts (external_id, name, type, currency, user_id)
SELECT 
    'user_' || u.id,
    u.name || ' Wallet',
    'user',
    'NGN',
    u.id
FROM users u
WHERE u.email IN ('alice@example.com', 'bob@example.com', 'charlie@example.com')
ON CONFLICT (user_id) DO NOTHING;

-- ============================================
-- 3. FUND TEST ACCOUNTS
-- ============================================
-- Transfer ₦3,000 total from Reserve to test users
-- Alice: ₦1,000 (100,000 kobo)
-- Bob:   ₦1,000 (100,000 kobo)  
-- Charlie: ₦1,000 (100,000 kobo)
-- Reserve: -₦3,000 (-300,000 kobo)
-- Sum: 0 ✓

-- Create transaction
INSERT INTO transactions (idempotency_key, kind, status, reference)
VALUES ('seed_initial_funding', 'deposit', 'posted', 'Initial test funding')
ON CONFLICT (idempotency_key) DO NOTHING;

-- Create postings (only if transaction was just created)
INSERT INTO postings (transaction_id, account_id, amount, currency)
SELECT 
    t.id,
    a.id,
    CASE 
        WHEN a.external_id = 'sys_reserve' THEN -300000  -- Debit reserve
        ELSE 100000                                       -- Credit each user
    END,
    'NGN'
FROM transactions t
CROSS JOIN accounts a
WHERE t.idempotency_key = 'seed_initial_funding'
  AND a.external_id IN ('sys_reserve', 'user_1', 'user_2', 'user_3')
  AND NOT EXISTS (
    SELECT 1 FROM postings p WHERE p.transaction_id = t.id
  );

COMMIT;

-- ============================================
-- VERIFICATION
-- ============================================
\echo '=== Test data created ==='
\echo '\n--- Users ---'
SELECT * FROM users;

\echo '\n--- User Accounts (should each have 100,000 kobo = ₦1,000) ---'
SELECT id, external_id, name, type, balance, currency, user_id 
FROM accounts 
WHERE type = 'user'
ORDER BY id;

\echo '\n--- System Accounts (Reserve should have -300,000) ---'
SELECT id, external_id, name, type, balance, currency 
FROM accounts 
WHERE type = 'system'
ORDER BY id;

\echo '\n--- Transactions ---'
SELECT * FROM transactions;

\echo '\n--- Postings (should sum to 0) ---'
SELECT 
    p.id,
    p.transaction_id,
    a.external_id as account,
    p.amount,
    p.currency
FROM postings p
JOIN accounts a ON a.id = p.account_id
ORDER BY p.id;

\echo '\n--- Balance Check (should be 0) ---'
SELECT 
    t.id as txn_id,
    t.idempotency_key,
    SUM(p.amount) as total_sum,
    CASE WHEN SUM(p.amount) = 0 THEN '✓ Valid' ELSE '✗ INVALID' END as status
FROM transactions t
JOIN postings p ON p.transaction_id = t.id
GROUP BY t.id, t.idempotency_key;

\echo '\n=== Setup complete! ==='