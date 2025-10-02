BEGIN;

-- Insert users
INSERT INTO users (name, email) VALUES
('Alice', 'alice@example.com'),
('Bob', 'bob@example.com'),
('Charlie', 'charlie@example.com');

-- Link wallets to users
INSERT INTO accounts (external_id, name, type, currency, user_id)
VALUES
('user_1', 'Alice Wallet', 'user', 'NGN', (SELECT id FROM users WHERE email = 'alice@example.com')),
('user_2', 'Bob Wallet', 'user', 'NGN', (SELECT id FROM users WHERE email = 'bob@example.com')),
('user_3', 'Charlie Wallet', 'user', 'NGN', (SELECT id FROM users WHERE email = 'charlie@example.com'));

-- Seed balances by transferring from Reserve
INSERT INTO transactions (idempotency_key, kind, status, reference)
VALUES ('seed_txn_1', 'deposit', 'posted', 'Initial funding');

WITH last_txn AS (SELECT currval(pg_get_serial_sequence('transactions','id')) AS txn_id)
INSERT INTO postings (transaction_id, account_id, amount, currency)
SELECT txn_id, id, amt, 'NGN'
FROM last_txn, (VALUES
     ((SELECT id FROM accounts WHERE external_id = 'sys_reserve'), -300000),
     ((SELECT id FROM accounts WHERE external_id = 'user_1'), 100000),
     ((SELECT id FROM accounts WHERE external_id = 'user_2'), 100000),
     ((SELECT id FROM accounts WHERE external_id = 'user_3'), 100000)
) AS t(id, amt);

COMMIT;
