BEGIN;

DROP TRIGGER IF EXISTS update_balance_trigger ON postings;
DROP FUNCTION IF EXISTS update_account_balance_on_posting();

COMMIT;