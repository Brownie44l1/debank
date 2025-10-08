-- ============================================
-- FUNCTIONS AND TRIGGERS
-- ============================================

BEGIN;

-- ============================================
-- 1. UPDATE TIMESTAMP TRIGGER
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_accounts_updated_at
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 2. ACCOUNT NUMBER GENERATION
-- ============================================

-- Calculate Luhn checksum digit
CREATE OR REPLACE FUNCTION calculate_luhn_checksum(num TEXT)
RETURNS INT AS $$
DECLARE
    total INT := 0;
    digit INT;
    i INT;
BEGIN
    FOR i IN 1..length(num) LOOP
        digit := substring(num from i for 1)::INT;
        IF i % 2 = 1 THEN
            digit := digit * 2;
            IF digit > 9 THEN
                digit := digit - 9;
            END IF;
        END IF;
        total := total + digit;
    END LOOP;
    
    RETURN (10 - (total % 10)) % 10;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Generate 10-digit account number from phone
CREATE OR REPLACE FUNCTION generate_account_number(phone TEXT)
RETURNS TEXT AS $$
DECLARE
    base_number TEXT;
    checksum INT;
BEGIN
    -- Remove country code (+234) and leading 0
    base_number := regexp_replace(phone, '^\+?234|^0', '');
    
    -- Take first 9 digits
    base_number := substring(base_number from 1 for 9);
    
    -- Add Luhn checksum as 10th digit
    checksum := calculate_luhn_checksum(base_number);
    
    RETURN base_number || checksum::TEXT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================
-- 3. AUTO-CREATE ACCOUNT ON USER SIGNUP
-- ============================================
CREATE OR REPLACE FUNCTION create_user_account()
RETURNS TRIGGER AS $$
DECLARE
    acct_number TEXT;
BEGIN
    acct_number := generate_account_number(NEW.phone);
    
    INSERT INTO accounts (
        account_number,
        name,
        type,
        user_id,
        balance
    ) VALUES (
        acct_number,
        NEW.name || '''s Account',
        'user',
        NEW.id,
        0
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_account_on_user_signup
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_user_account();

-- ============================================
-- 4. USERNAME UTILITIES
-- ============================================
CREATE OR REPLACE FUNCTION is_username_available(check_username TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN NOT EXISTS (
        SELECT 1 FROM users WHERE username = check_username
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION suggest_usernames(base_username TEXT, limit_count INT DEFAULT 5)
RETURNS TABLE(suggestion TEXT) AS $$
BEGIN
    RETURN QUERY
    WITH suggestions AS (
        SELECT base_username || generate_series(1, limit_count * 2) AS potential_username
    )
    SELECT s.potential_username
    FROM suggestions s
    WHERE NOT EXISTS (
        SELECT 1 FROM users WHERE username = s.potential_username
    )
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 5. OTP UTILITIES
-- ============================================
CREATE OR REPLACE FUNCTION generate_otp()
RETURNS TEXT AS $$
BEGIN
    RETURN lpad(floor(random() * 1000000)::TEXT, 6, '0');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION verify_email_otp(
    input_email TEXT,
    input_code TEXT,
    input_purpose TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    valid_code RECORD;
BEGIN
    -- Find valid OTP
    SELECT * INTO valid_code
    FROM verification_codes
    WHERE email = input_email
      AND code = input_code
      AND purpose = input_purpose
      AND used_at IS NULL
      AND expires_at > now()
      AND attempts < 5
    ORDER BY created_at DESC
    LIMIT 1;
    
    -- If not found, increment attempts
    IF valid_code IS NULL THEN
        UPDATE verification_codes
        SET attempts = attempts + 1
        WHERE email = input_email 
          AND code = input_code
          AND purpose = input_purpose;
        
        RETURN FALSE;
    END IF;
    
    -- Mark OTP as used
    UPDATE verification_codes
    SET used_at = now()
    WHERE id = valid_code.id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Clean up expired OTPs (run this periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_otps()
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM verification_codes
    WHERE expires_at < now() - INTERVAL '7 days'
    RETURNING COUNT(*) INTO deleted_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 6. DOUBLE-ENTRY ACCOUNTING
-- ============================================
CREATE OR REPLACE FUNCTION enforce_double_entry() 
RETURNS TRIGGER AS $$
DECLARE
    txn_balance BIGINT;
BEGIN
    SELECT COALESCE(SUM(amount), 0) INTO txn_balance
    FROM postings 
    WHERE transaction_id = NEW.transaction_id;
    
    IF txn_balance <> 0 THEN
        RAISE EXCEPTION 'Transaction postings must balance to zero (current sum: %)', txn_balance;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER postings_balance_trigger
AFTER INSERT OR UPDATE ON postings
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW 
EXECUTE FUNCTION enforce_double_entry();

-- ============================================
-- 7. AUTO-UPDATE ACCOUNT BALANCE
-- ============================================
CREATE OR REPLACE FUNCTION update_account_balance_on_posting()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE accounts 
    SET balance = balance + NEW.amount 
    WHERE id = NEW.account_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_balance_trigger
AFTER INSERT ON postings
FOR EACH ROW 
EXECUTE FUNCTION update_account_balance_on_posting();

-- ============================================
-- 8. ACCOUNT BALANCE QUERY HELPER
-- ============================================
CREATE OR REPLACE FUNCTION get_account_balance(input_account_id BIGINT)
RETURNS BIGINT AS $$
DECLARE
    current_balance BIGINT;
BEGIN
    SELECT balance INTO current_balance
    FROM accounts
    WHERE id = input_account_id;
    
    RETURN COALESCE(current_balance, 0);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 9. TRANSACTION HISTORY HELPER
-- ============================================
CREATE OR REPLACE FUNCTION get_user_transactions(
    input_user_id INT,
    limit_count INT DEFAULT 20,
    offset_count INT DEFAULT 0
)
RETURNS TABLE(
    id BIGINT,
    reference TEXT,
    kind TEXT,
    status TEXT,
    amount BIGINT,
    description TEXT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.reference,
        t.kind,
        t.status,
        t.amount,
        t.description,
        t.created_at
    FROM transactions t
    INNER JOIN accounts a ON (t.from_account_id = a.id OR t.to_account_id = a.id)
    WHERE a.user_id = input_user_id
    ORDER BY t.created_at DESC
    LIMIT limit_count
    OFFSET offset_count;
END;
$$ LANGUAGE plpgsql;

COMMIT;

\echo '=== Functions and triggers created successfully ==='