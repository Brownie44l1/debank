-- ==============================================
-- TEST DATA SETUP FOR WALLET API (10 USERS)
-- ==============================================
-- Extended version to support concurrency testing with 10 workers

-- Clear previous test data
DELETE FROM postings WHERE transaction_id IN (SELECT id FROM transactions WHERE idempotency_key LIKE 'test_hist_%');
DELETE FROM transactions WHERE idempotency_key LIKE 'test_hist_%';

-- ==============================================
-- CREATE TEST USERS (1-10)
-- ==============================================

INSERT INTO users (id, name, email)
VALUES
    (1, 'User 1', 'user1@test.com'),
    (2, 'User 2', 'user2@test.com'),
    (3, 'User 3', 'user3@test.com'),
    (4, 'User 4', 'user4@test.com'),
    (5, 'User 5', 'user5@test.com'),
    (6, 'User 6', 'user6@test.com'),
    (7, 'User 7', 'user7@test.com'),
    (8, 'User 8', 'user8@test.com'),
    (9, 'User 9', 'user9@test.com'),
    (10, 'User 10', 'user10@test.com')
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
-- CREATE TEST USER ACCOUNTS (1-10)
-- ==============================================

-- Users 1-5: Original test users
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 1 Wallet', 'user', 500000, 'NGN', 1)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 500000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 2 Wallet', 'user', 200000, 'NGN', 2)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 200000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 3 Wallet', 'user', 1000000, 'NGN', 3)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 4 Wallet', 'user', 10000, 'NGN', 4)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 10000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 5 Wallet', 'user', 0, 'NGN', 5)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 0;

-- Users 6-10: New users for concurrency testing
-- Each starts with ₦10,000 (1,000,000 kobo) for adequate test balance
INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 6 Wallet', 'user', 1000000, 'NGN', 6)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 7 Wallet', 'user', 1000000, 'NGN', 7)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 8 Wallet', 'user', 1000000, 'NGN', 8)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 9 Wallet', 'user', 1000000, 'NGN', 9)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

INSERT INTO accounts (name, type, balance, currency, user_id)
VALUES ('User 10 Wallet', 'user', 1000000, 'NGN', 10)
ON CONFLICT (user_id) DO UPDATE 
SET balance = 1000000;

-- ==============================================
-- CREATE HISTORICAL TRANSACTIONS (Optional)
-- ==============================================

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

-- Count total users
SELECT COUNT(*) as total_users FROM users;

-- Count total user accounts
SELECT COUNT(*) as total_accounts FROM accounts WHERE user_id IS NOT NULL;

-- ==============================================
-- SUMMARY
-- ==============================================
-- After running this script, you should have:
-- - User 1: ₦5,000 balance with historical transactions
-- - User 2: ₦2,000 balance with historical transactions
-- - User 3: ₦10,000 balance
-- - User 4: ₦100 balance (for minimum amount tests)
-- - User 5: ₦0 balance (for insufficient balance tests)
-- - Users 6-10: ₦10,000 each (for concurrency testing)
-- 
-- Total: 10 users ready for 10-worker load testing!
-- ==============================================