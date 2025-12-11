-- Migration: Fix user email uniqueness constraint
-- Problem: UNIQUE (tenant_id, email) allows duplicate customer emails because NULL != NULL in SQL
-- Solution: Add partial unique index for customer emails (where tenant_id IS NULL)

-- Drop the old composite unique constraint that doesn't work for NULL tenant_id
ALTER TABLE users DROP CONSTRAINT IF EXISTS unique_email_per_tenant;

-- Create partial unique index for customer emails (tenant_id IS NULL)
-- This ensures globally unique emails for customers
CREATE UNIQUE INDEX idx_users_email_null_tenant
ON users(email)
WHERE tenant_id IS NULL;

-- Create composite unique index for organizer emails (tenant_id IS NOT NULL)
-- This ensures email uniqueness per tenant for organizers
CREATE UNIQUE INDEX idx_users_email_with_tenant
ON users(tenant_id, email)
WHERE tenant_id IS NOT NULL;

-- Add CHECK constraint to ensure organizers must have tenant_id
ALTER TABLE users
ADD CONSTRAINT check_organizer_has_tenant
CHECK (
    (role != 'organizer' AND role != 'admin')
    OR tenant_id IS NOT NULL
);

-- Add comment explaining the constraint
COMMENT ON CONSTRAINT check_organizer_has_tenant ON users IS
'Ensures that users with organizer or admin role must have a tenant_id';
