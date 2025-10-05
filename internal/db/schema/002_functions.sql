-- ============================================
-- FUNCTIONS AND TRIGGERS
-- ============================================
-- This file contains database functions and triggers that enforce
-- business rules and maintain data integrity.
--
-- Key features:
-- 1. Double-entry enforcement: All postings in a transaction must sum to zero
-- 2. Auto-update balances: Account balances update automatically when postings are inserted
-- ============================================

BEGIN;

-- ============================================
-- 1. DOUBLE-ENTRY ENFORCEMENT FUNCTION
-- ============================================
-- Ensures all postings for a transaction sum to zero
-- This is the core rule of double-entry bookkeeping
--
-- Example valid transaction:
--   Alice: -1000 (debit)
--   Bob:   +1000 (credit)
--   Sum:   0 ✓
--
-- Example invalid transaction:
--   Alice: -1000
--   Bob:   +950
--   Sum:   -50 ✗ (REJECTED)

CREATE OR REPLACE FUNCTION enforce_double_entry() 
RETURNS TRIGGER AS $$
DECLARE
  txn_balance BIGINT;
BEGIN
  -- Calculate sum of all postings for this transaction
  SELECT COALESCE(SUM(amount), 0) INTO txn_balance
  FROM postings 
  WHERE transaction_id = NEW.transaction_id;
  
  -- Reject if sum is not zero
  IF txn_balance <> 0 THEN
    RAISE EXCEPTION 'Transaction postings must balance to zero (current sum: %)', txn_balance;
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION enforce_double_entry() IS 
'Validates that all postings in a transaction sum to zero (double-entry rule)';

-- ============================================
-- 2. DOUBLE-ENTRY ENFORCEMENT TRIGGER
-- ============================================
-- Runs AFTER each posting is inserted/updated
-- DEFERRABLE INITIALLY DEFERRED means the check happens at end of transaction
-- This allows multiple postings to be inserted before validation

CREATE CONSTRAINT TRIGGER postings_balance_trigger
AFTER INSERT OR UPDATE ON postings
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW 
EXECUTE FUNCTION enforce_double_entry();

COMMENT ON TRIGGER postings_balance_trigger ON postings IS
'Deferred trigger that enforces double-entry rule at transaction commit time';

-- ============================================
-- 3. BALANCE AUTO-UPDATE FUNCTION
-- ============================================
-- Automatically updates account balance when a posting is inserted
-- This eliminates need for manual balance calculations
--
-- Example:
--   Account balance: 5000
--   New posting: +1000
--   New balance: 6000 (auto-calculated)

CREATE OR REPLACE FUNCTION update_account_balance_on_posting()
RETURNS TRIGGER AS $$
BEGIN
    -- Add posting amount to account's current balance
    UPDATE accounts 
    SET balance = balance + NEW.amount 
    WHERE id = NEW.account_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_account_balance_on_posting() IS
'Automatically updates account balance when posting is inserted';

-- ============================================
-- 4. BALANCE AUTO-UPDATE TRIGGER
-- ============================================
-- Runs AFTER each posting is inserted
-- Fires BEFORE the double-entry check (ordering matters!)

CREATE TRIGGER update_balance_trigger
AFTER INSERT ON postings
FOR EACH ROW 
EXECUTE FUNCTION update_account_balance_on_posting();

COMMENT ON TRIGGER update_balance_trigger ON postings IS
'Automatically updates account balance when posting is created';

COMMIT;

-- ============================================
-- VERIFICATION
-- ============================================
\echo '=== Functions and triggers created successfully ==='
\df enforce_double_entry
\df update_account_balance_on_posting
\echo '=== Run schema/001_system_accounts.sql next ==='