-- Seed Data for Load Testing - Auth DB
-- Creates 10,000 test users for k6 load testing

-- Use existing tenant
-- Tenant ID: 00000000-0000-0000-0000-000000000001

-- Clean up existing load test users
DELETE FROM users WHERE email LIKE 'loadtest%@test.com';

-- Insert test users (10,000 users for high concurrency testing)
-- Using deterministic UUIDs for easy reference in k6
INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, is_active)
SELECT
    ('a0000000-0000-0000-0000-' || LPAD(i::text, 12, '0'))::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'loadtest' || i || '@test.com',
    '$2a$12$WC0ROfKFxUM0T1JkoCozau5.QGXlM6Sfo.g2/YPvJWDufPQD2be0e', -- password: Test123!
    'LoadTest',
    'User' || i,
    'customer',
    true
FROM generate_series(1, 10000) AS i
ON CONFLICT DO NOTHING;

-- Summary
SELECT
    'Auth DB Seed Summary' as info,
    (SELECT COUNT(*) FROM users WHERE email LIKE 'loadtest%@test.com') as load_test_users;
