BEGIN;

-- Function to automatically update account balance when posting is inserted
CREATE OR REPLACE FUNCTION update_account_balance_on_posting()
RETURNS TRIGGER AS $$
BEGIN
    -- Add the posting amount to the account's balance
    UPDATE accounts 
    SET balance = balance + NEW.amount 
    WHERE id = NEW.account_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger fires BEFORE the double-entry check
CREATE TRIGGER update_balance_trigger
AFTER INSERT ON postings
FOR EACH ROW 
EXECUTE FUNCTION update_account_balance_on_posting();

COMMIT;