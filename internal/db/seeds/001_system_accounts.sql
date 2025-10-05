-- ============================================
-- SYSTEM ACCOUNTS SEED DATA
-- ============================================
-- Creates essential system accounts needed for operations
-- These accounts should exist in ALL environments (dev, staging, production)
--
-- System accounts:
-- - Reserve Account: Holds system funds, used for initial funding
-- - Fee Account: Collects transaction fees
-- ============================================

BEGIN;

INSERT INTO accounts (external_id, name, type, currency) VALUES
('sys_reserve', 'Reserve Account', 'system', 'NGN'),
('sys_fee', 'Fee Account', 'system', 'NGN')
ON CONFLICT (external_id) DO NOTHING; -- Skip if already exists

COMMIT;

-- ============================================
-- VERIFICATION
-- ============================================
\echo '=== System accounts created ==='
SELECT id, external_id, name, type, balance, currency FROM accounts WHERE type = 'system';
\echo '=== Run seeds/002_test_data.sql for test data (dev only) ==='