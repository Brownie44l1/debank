BEGIN;

-- Insert system accounts (no user_id, since these are not tied to individuals)
INSERT INTO accounts (external_id, name, type, currency)
VALUES
('sys_reserve', 'Reserve Account', 'system', 'NGN'),
('sys_fee', 'Fee Account', 'system', 'NGN');

COMMIT;
