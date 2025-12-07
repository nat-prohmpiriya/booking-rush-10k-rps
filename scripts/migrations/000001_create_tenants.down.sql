-- 000001_create_tenants.down.sql

DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants;
-- Note: We don't drop the update_updated_at_column function as it may be used by other tables
