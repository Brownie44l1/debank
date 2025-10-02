BEGIN;

-- Backfill balances from existing postings (important if seed ran before)
UPDATE accounts a
SET balance = COALESCE((
    SELECT SUM(p.amount) FROM postings p WHERE p.account_id = a.id
), 0);

COMMIT;
