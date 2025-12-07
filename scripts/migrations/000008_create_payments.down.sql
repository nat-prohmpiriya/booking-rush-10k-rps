-- 000008_create_payments.down.sql

DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS payment_status;
