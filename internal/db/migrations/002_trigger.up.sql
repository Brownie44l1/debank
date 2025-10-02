-- Function to enforce double-entry
CREATE OR REPLACE FUNCTION enforce_double_entry() RETURNS trigger AS $$
BEGIN
  -- Check if postings for this transaction balance
  IF (SELECT COALESCE(SUM(amount), 0) 
      FROM postings 
      WHERE transaction_id = NEW.transaction_id) <> 0 THEN
    RAISE EXCEPTION 'Transaction postings must balance to zero';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger
CREATE TRIGGER postings_balance_trigger
AFTER INSERT OR UPDATE ON postings
FOR EACH ROW EXECUTE FUNCTION enforce_double_entry();
