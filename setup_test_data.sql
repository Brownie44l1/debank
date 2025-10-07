-- ==============================================
-- TEST DATA SETUP FOR WALLET API
-- ==============================================
-- Run this to populate your database with test data
-- for testing with Postman

-- Clear existing test data (optional - be careful in production!)
-- DELETE FROM postings WHERE transaction_id IN (SELECT id FROM transactions WHERE reference LIKE 'test_%');
-- DELETE FROM transactions WHERE reference LIKE 'test_%';
-- Clear previous test data
DELETE FROM postings WHERE transaction_id IN (SELECT id FROM transactions WHERE idempotency_key LIKE 'test_hist_%');
DELETE FROM transactions WHERE idempotency_key LIKE 'test_hist_%';
INSERT INTO users (id, name, email)
VALUES
    (1, 'User 1', 'user1@test.com'),
    (2, 'User 2', 'user2@test.com'),
    (3, 'User 3', 'user3@test.com'),
    (4, 'User 4', 'user4@test.com'),
    (5, 'User 5', 'user5@test.com')
ON CONFLICT (id) DO NOTHING;

-- ==============================================
-- CREATE SYSTEM ACCOUNTS (if not exists)
-- ==============================================

INSERT INTO accounts (external_id, name, type, balance, currency, user_id)
VALUES 
    ('sys_reserve', 'System Reserve Account', 'system', 100000000000, 'NGN', NULL),
    ('sys_fee', 'System Fee Account', 'system', 0, 'NGN', NULL)
ON CONFLICT (external_id) DO NOTHING;

-- ==============================================
-- CREATE TEST USERS ACCOUNTS
-- ==============================================

-- User 1: Starting with ₦5,000 (500,000 kobo)
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 1 Wallet', 'user', 500000, 'NGN', 1)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 500000;

-- User 2: Starting with ₦2,000 (200,000 kobo)
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 2 Wallet', 'user', 200000, 'NGN', 2)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 200000;

-- User 3: Starting with ₦10,000 (1,000,000 kobo)
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 3 Wallet', 'user', 1000000, 'NGN', 3)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

-- User 4: Starting with ₦100 (10,000 kobo) - for testing minimum amounts
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 4 Wallet', 'user', 10000, 'NGN', 4)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 10000;

-- User 5: Starting with ₦0 (0 kobo) - for testing insufficient balance
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 5 Wallet', 'user', 0, 'NGN', 5)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 0;

-- ==============================================
-- CREATE SOME HISTORICAL TRANSACTIONS (Optional)
-- ==============================================

-- Get account IDs
DO $$
DECLARE
    user1_account_id BIGINT;
    user2_account_id BIGINT;
    reserve_account_id BIGINT;
    fee_account_id BIGINT;
    txn_id BIGINT;
BEGIN
    -- Get account IDs
    SELECT id INTO user1_account_id FROM accounts WHERE user_id = 1;
    SELECT id INTO user2_account_id FROM accounts WHERE user_id = 2;
    SELECT id INTO reserve_account_id FROM accounts WHERE external_id = 'sys_reserve';
    SELECT id INTO fee_account_id FROM accounts WHERE external_id = 'sys_fee';

    -- Historical Deposit for User 1
    INSERT INTO transactions (idempotency_key, kind, status, reference, created_at)
    VALUES ('test_hist_deposit_1', 'deposit', 'posted', 'Initial deposit', NOW() - INTERVAL '7 days')
    RETURNING id INTO txn_id;

    INSERT INTO postings (transaction_id, account_id, amount, currency, created_at)
    VALUES 
        (txn_id, reserve_account_id, -100000, 'NGN', NOW() - INTERVAL '7 days'),
        (txn_id, user1_account_id, 100000, 'NGN', NOW() - INTERVAL '7 days');

    -- Historical Transfer from User 1 to User 2
    INSERT INTO transactions (idempotency_key, kind, status, reference, created_at)
    VALUES ('test_hist_transfer_1', 'p2p', 'posted', 'Payment for services', NOW() - INTERVAL '5 days')
    RETURNING id INTO txn_id;

    INSERT INTO postings (transaction_id, account_id, amount, currency, created_at)
    VALUES 
        (txn_id, user1_account_id, -55000, 'NGN', NOW() - INTERVAL '5 days'),
        (txn_id, user2_account_id, 50000, 'NGN', NOW() - INTERVAL '5 days'),
        (txn_id, fee_account_id, 5000, 'NGN', NOW() - INTERVAL '5 days');

    -- Historical Withdrawal for User 1
    INSERT INTO transactions (idempotency_key, kind, status, reference, created_at)
    VALUES ('test_hist_withdraw_1', 'withdrawal', 'posted', 'ATM withdrawal', NOW() - INTERVAL '3 days')
    RETURNING id INTO txn_id;

    INSERT INTO postings (transaction_id, account_id, amount, currency, created_at)
    VALUES 
        (txn_id, user1_account_id, -20000, 'NGN', NOW() - INTERVAL '3 days'),
        (txn_id, reserve_account_id, 20000, 'NGN', NOW() - INTERVAL '3 days');

    -- Historical Deposit for User 2
    INSERT INTO transactions (idempotency_key, kind, status, reference, created_at)
    VALUES ('test_hist_deposit_2', 'deposit', 'posted', 'Salary payment', NOW() - INTERVAL '2 days')
    RETURNING id INTO txn_id;

    INSERT INTO postings (transaction_id, account_id, amount, currency, created_at)
    VALUES 
        (txn_id, reserve_account_id, -150000, 'NGN', NOW() - INTERVAL '2 days'),
        (txn_id, user2_account_id, 150000, 'NGN', NOW() - INTERVAL '2 days');

END $$;

-- ==============================================
-- VERIFY SETUP
-- ==============================================

-- Check all user accounts
SELECT 
    user_id,
    name,
    balance,
    ROUND(balance::numeric / 100, 2) as balance_ngn,
    currency
FROM accounts
WHERE user_id IS NOT NULL
ORDER BY user_id;

-- Check system accounts
SELECT 
    external_id,
    name,
    balance,
    ROUND(balance::numeric / 100, 2) as balance_ngn,
    currency
FROM accounts
WHERE type = 'system';

-- Check transaction history for User 1
SELECT 
    t.id,
    t.kind,
    t.reference,
    p.amount,
    ROUND(p.amount::numeric / 100, 2) as amount_ngn,
    CASE 
        WHEN p.amount > 0 THEN 'credit'
        ELSE 'debit'
    END as direction,
    t.created_at
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
JOIN accounts a ON a.id = p.account_id
WHERE a.user_id = 1
ORDER BY t.created_at DESC;

-- ==============================================
-- SUMMARY
-- ==============================================
-- After running this script, you should have:
-- - User 1: ₦5,000 balance with 3 historical transactions
-- - User 2: ₦2,000 balance with 2 historical transactions
-- - User 3: ₦10,000 balance (no history)
-- - User 4: ₦100 balance (for minimum amount tests)
-- - User 5: ₦0 balance (for insufficient balance tests)
-- ==============================================
