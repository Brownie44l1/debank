-- ============================================
-- TRANSACTION HISTORY QUERIES
-- ============================================
-- Various ways to get user transaction history
-- from your current schema
-- ============================================

-- ============================================
-- 1. BASIC ACCOUNT STATEMENT
-- ============================================
-- Shows all transactions for a specific account
-- with amount, date, and reference

SELECT 
    p.created_at as date,
    t.kind as type,
    t.reference,
    p.amount as amount_kobo,
    p.amount / 100.0 as amount_ngn,
    CASE 
        WHEN p.amount > 0 THEN 'Credit'
        ELSE 'Debit'
    END as direction
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1  -- Alice's account
ORDER BY p.created_at DESC
LIMIT 50;

-- ============================================
-- 2. DETAILED TRANSACTION VIEW
-- ============================================
-- Shows full transaction details including
-- the other party (who sent/received)

WITH user_postings AS (
    SELECT 
        p.created_at,
        t.id as txn_id,
        t.kind,
        t.reference,
        t.status,
        p.amount,
        p.account_id
    FROM postings p
    JOIN transactions t ON t.id = p.transaction_id
    WHERE p.account_id = 3  -- Your account ID
)
SELECT 
    up.created_at,
    up.kind,
    up.reference,
    up.status,
    up.amount / 100.0 as amount_ngn,
    CASE 
        WHEN up.amount > 0 THEN 'Credit'
        ELSE 'Debit'
    END as direction,
    -- Find the other account in this transaction
    other_acc.name as counterparty,
    other_acc.type as counterparty_type
FROM user_postings up
LEFT JOIN postings other_p ON other_p.transaction_id = up.txn_id 
    AND other_p.account_id != up.account_id
    AND SIGN(other_p.amount) != SIGN(up.amount)  -- opposite direction
LEFT JOIN accounts other_acc ON other_acc.id = other_p.account_id
ORDER BY up.created_at DESC;

-- ============================================
-- 3. RUNNING BALANCE (Bank Statement Style)
-- ============================================
-- Shows balance after each transaction
-- Note: This is computed on-the-fly. For production,
-- consider storing running_balance in postings table

SELECT 
    p.created_at,
    t.kind,
    t.reference,
    p.amount / 100.0 as amount_ngn,
    SUM(p.amount) OVER (
        ORDER BY p.created_at, p.id
    ) / 100.0 as running_balance_ngn
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
ORDER BY p.created_at DESC, p.id DESC;

-- ============================================
-- 4. FILTERED BY DATE RANGE
-- ============================================
-- Get transactions for a specific period

SELECT 
    p.created_at,
    t.kind,
    t.reference,
    p.amount / 100.0 as amount_ngn,
    CASE 
        WHEN p.amount > 0 THEN 'Credit'
        ELSE 'Debit'
    END as direction
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
  AND p.created_at >= '2024-01-01'
  AND p.created_at < '2024-02-01'
ORDER BY p.created_at DESC;

-- ============================================
-- 5. SUMMARY STATISTICS
-- ============================================
-- Total debits, credits, transaction count

SELECT 
    COUNT(*) as total_transactions,
    SUM(CASE WHEN p.amount > 0 THEN p.amount ELSE 0 END) / 100.0 as total_credits_ngn,
    SUM(CASE WHEN p.amount < 0 THEN ABS(p.amount) ELSE 0 END) / 100.0 as total_debits_ngn,
    (SUM(p.amount)) / 100.0 as net_change_ngn
FROM postings p
WHERE p.account_id = 1
  AND p.created_at >= '2024-01-01'
  AND p.created_at < '2025-01-01';

-- ============================================
-- 6. BY TRANSACTION TYPE
-- ============================================
-- Group transactions by type (p2p, deposit, etc)

SELECT 
    t.kind,
    COUNT(*) as count,
    SUM(CASE WHEN p.amount > 0 THEN p.amount ELSE 0 END) / 100.0 as total_received,
    SUM(CASE WHEN p.amount < 0 THEN ABS(p.amount) ELSE 0 END) / 100.0 as total_sent
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
GROUP BY t.kind
ORDER BY count DESC;

-- ============================================
-- 7. PAGINATION (For Mobile App)
-- ============================================
-- Efficiently load transactions in pages

SELECT 
    p.id,
    p.created_at,
    t.kind,
    t.reference,
    p.amount / 100.0 as amount_ngn,
    CASE 
        WHEN p.amount > 0 THEN 'Credit'
        ELSE 'Debit'
    END as direction
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
ORDER BY p.created_at DESC, p.id DESC
LIMIT 20 OFFSET 0;  -- Page 1: OFFSET 0, Page 2: OFFSET 20, etc.

-- ============================================
-- 8. SEARCH BY REFERENCE/DESCRIPTION
-- ============================================
-- Find specific transactions

SELECT 
    p.created_at,
    t.kind,
    t.reference,
    t.metadata,
    p.amount / 100.0 as amount_ngn
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
  AND (
    t.reference ILIKE '%search_term%'
    OR t.metadata::text ILIKE '%search_term%'
  )
ORDER BY p.created_at DESC;

-- ============================================
-- 9. RECENT TRANSACTIONS (Dashboard)
-- ============================================
-- Last 5 transactions for quick overview

SELECT 
    p.created_at,
    t.kind,
    t.reference,
    p.amount / 100.0 as amount_ngn,
    CASE 
        WHEN p.amount > 0 THEN '↓ Received'
        ELSE '↑ Sent'
    END as direction
FROM postings p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = 1
  AND t.status = 'posted'  -- Only successful transactions
ORDER BY p.created_at DESC
LIMIT 5;

-- ============================================
-- PERFORMANCE NOTE
-- ============================================
-- All these queries use the index:
-- idx_postings_account_id_created_at
-- 
-- This makes them FAST even with millions of rows
-- because the database can:
-- 1. Jump to account_id = X
-- 2. Read rows already sorted by created_at
-- 3. Stop after LIMIT is reached
-- ============================================