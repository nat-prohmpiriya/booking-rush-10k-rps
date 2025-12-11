-- Rollback migration: Revert email uniqueness fix

-- Drop the CHECK constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_organizer_has_tenant;

-- Drop the partial unique indexes
DROP INDEX IF EXISTS idx_users_email_null_tenant;
DROP INDEX IF EXISTS idx_users_email_with_tenant;

-- Restore the original composite unique constraint
-- Note: This constraint has the bug where NULL tenant_id allows duplicate emails
ALTER TABLE users
ADD CONSTRAINT unique_email_per_tenant UNIQUE (tenant_id, email);
