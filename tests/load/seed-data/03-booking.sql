-- Seed Data for Load Testing
-- This script creates test data for k6 load testing

-- Clean up existing test data
DELETE FROM bookings WHERE id LIKE 'load-test-%' OR user_id LIKE 'load-test-user-%';
DELETE FROM seat_zones WHERE id LIKE 'load-test-%';
DELETE FROM shows WHERE id LIKE 'load-test-%';
DELETE FROM events WHERE id LIKE 'load-test-%';
DELETE FROM users WHERE id LIKE 'load-test-user-%';
DELETE FROM tenants WHERE id = 'load-test-tenant';

-- Insert test tenant
INSERT INTO tenants (id, name, slug, is_active)
VALUES ('load-test-tenant', 'Load Test Tenant', 'load-test', true)
ON CONFLICT DO NOTHING;

-- Insert test users (10,000 users for high concurrency testing)
INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, is_active)
SELECT
    'load-test-user-' || i,
    'load-test-tenant',
    'loadtest' || i || '@test.com',
    '$2a$10$dummyhashforloadtesting',
    'User',
    CAST(i AS VARCHAR),
    'customer',
    true
FROM generate_series(1, 10000) AS i
ON CONFLICT DO NOTHING;

-- Insert test organizer
INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, is_active)
VALUES (
    'load-test-organizer',
    'load-test-tenant',
    'organizer@test.com',
    '$2a$10$dummyhashforloadtesting',
    'Test',
    'Organizer',
    'organizer',
    true
)
ON CONFLICT DO NOTHING;

-- Insert test events (3 events)
INSERT INTO events (id, tenant_id, organizer_id, name, slug, description, venue_name, city, status, max_tickets_per_user, booking_start_at, booking_end_at, is_public)
VALUES
    ('load-test-event-1', 'load-test-tenant', 'load-test-organizer', 'Load Test Concert 1', 'load-test-concert-1', 'High volume concert for load testing', 'Test Stadium', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true),
    ('load-test-event-2', 'load-test-tenant', 'load-test-organizer', 'Load Test Concert 2', 'load-test-concert-2', 'Another high volume event', 'Test Arena', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true),
    ('load-test-event-3', 'load-test-tenant', 'load-test-organizer', 'Load Test Concert 3', 'load-test-concert-3', 'Third load test event', 'Test Hall', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true)
ON CONFLICT DO NOTHING;

-- Insert test shows (3 shows per event = 9 shows total)
INSERT INTO shows (id, event_id, name, show_date, doors_open_at, start_time, end_time, status, total_capacity)
VALUES
    -- Event 1 shows
    ('load-test-show-1-1', 'load-test-event-1', 'Show 1 - Evening', NOW() + INTERVAL '7 days', NOW() + INTERVAL '7 days' - INTERVAL '2 hours', NOW() + INTERVAL '7 days', NOW() + INTERVAL '7 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-1-2', 'load-test-event-1', 'Show 2 - Evening', NOW() + INTERVAL '14 days', NOW() + INTERVAL '14 days' - INTERVAL '2 hours', NOW() + INTERVAL '14 days', NOW() + INTERVAL '14 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-1-3', 'load-test-event-1', 'Show 3 - Evening', NOW() + INTERVAL '21 days', NOW() + INTERVAL '21 days' - INTERVAL '2 hours', NOW() + INTERVAL '21 days', NOW() + INTERVAL '21 days' + INTERVAL '3 hours', 'on_sale', 100000),
    -- Event 2 shows
    ('load-test-show-2-1', 'load-test-event-2', 'Show 1', NOW() + INTERVAL '8 days', NOW() + INTERVAL '8 days' - INTERVAL '2 hours', NOW() + INTERVAL '8 days', NOW() + INTERVAL '8 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-2-2', 'load-test-event-2', 'Show 2', NOW() + INTERVAL '15 days', NOW() + INTERVAL '15 days' - INTERVAL '2 hours', NOW() + INTERVAL '15 days', NOW() + INTERVAL '15 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-2-3', 'load-test-event-2', 'Show 3', NOW() + INTERVAL '22 days', NOW() + INTERVAL '22 days' - INTERVAL '2 hours', NOW() + INTERVAL '22 days', NOW() + INTERVAL '22 days' + INTERVAL '3 hours', 'on_sale', 100000),
    -- Event 3 shows
    ('load-test-show-3-1', 'load-test-event-3', 'Show 1', NOW() + INTERVAL '9 days', NOW() + INTERVAL '9 days' - INTERVAL '2 hours', NOW() + INTERVAL '9 days', NOW() + INTERVAL '9 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-3-2', 'load-test-event-3', 'Show 2', NOW() + INTERVAL '16 days', NOW() + INTERVAL '16 days' - INTERVAL '2 hours', NOW() + INTERVAL '16 days', NOW() + INTERVAL '16 days' + INTERVAL '3 hours', 'on_sale', 100000),
    ('load-test-show-3-3', 'load-test-event-3', 'Show 3', NOW() + INTERVAL '23 days', NOW() + INTERVAL '23 days' - INTERVAL '2 hours', NOW() + INTERVAL '23 days', NOW() + INTERVAL '23 days' + INTERVAL '3 hours', 'on_sale', 100000)
ON CONFLICT DO NOTHING;

-- Insert seat zones (5 zones per show = 45 zones total)
-- Each zone has 20,000 seats to handle 10k RPS
INSERT INTO seat_zones (id, show_id, name, description, price, currency, total_seats, available_seats, min_per_order, max_per_order, is_active, sort_order)
SELECT
    'load-test-zone-' || s.show_num || '-' || z.zone_num,
    'load-test-show-' || s.event_num || '-' || s.show_num,
    z.name,
    'Load test zone for ' || z.name,
    z.price,
    'THB',
    20000,  -- 20,000 seats per zone
    20000,  -- All available
    1,
    10,
    true,
    z.zone_num
FROM
    (SELECT 1 as event_num, 1 as show_num UNION ALL
     SELECT 1, 2 UNION ALL SELECT 1, 3 UNION ALL
     SELECT 2, 1 UNION ALL SELECT 2, 2 UNION ALL SELECT 2, 3 UNION ALL
     SELECT 3, 1 UNION ALL SELECT 3, 2 UNION ALL SELECT 3, 3) s,
    (SELECT 1 as zone_num, 'VIP' as name, 5000.00 as price UNION ALL
     SELECT 2, 'Gold', 3000.00 UNION ALL
     SELECT 3, 'Silver', 2000.00 UNION ALL
     SELECT 4, 'Bronze', 1000.00 UNION ALL
     SELECT 5, 'Standing', 500.00) z
ON CONFLICT DO NOTHING;

-- Summary query
SELECT
    'Test Data Summary' as info,
    (SELECT COUNT(*) FROM users WHERE id LIKE 'load-test-%') as test_users,
    (SELECT COUNT(*) FROM events WHERE id LIKE 'load-test-%') as test_events,
    (SELECT COUNT(*) FROM shows WHERE id LIKE 'load-test-%') as test_shows,
    (SELECT COUNT(*) FROM seat_zones WHERE id LIKE 'load-test-%') as test_zones,
    (SELECT SUM(available_seats) FROM seat_zones WHERE id LIKE 'load-test-%') as total_available_seats;
