-- Seed Data for Load Testing - Ticket DB
-- Creates events, shows, and zones for 10k RPS load testing

-- Tenant ID: 00000000-0000-0000-0000-000000000001
-- Organizer ID: ff782c9d-f2a3-42b3-9ba8-c16a3f1354e1

-- Clean up existing load test data
DELETE FROM seat_zones WHERE id::text LIKE 'b0000000-%';
DELETE FROM shows WHERE id::text LIKE 'c0000000-%';
DELETE FROM events WHERE id::text LIKE 'd0000000-%';

-- Insert test events (3 events)
INSERT INTO events (id, tenant_id, organizer_id, name, slug, description, venue_name, city, status, max_tickets_per_user, booking_start_at, booking_end_at, is_public)
VALUES
    ('d0000000-0000-0000-0000-000000000001'::uuid, '00000000-0000-0000-0000-000000000001'::uuid, 'ff782c9d-f2a3-42b3-9ba8-c16a3f1354e1'::uuid, 'Load Test Concert 1', 'load-test-concert-1', 'High volume concert for load testing', 'Test Stadium', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true),
    ('d0000000-0000-0000-0000-000000000002'::uuid, '00000000-0000-0000-0000-000000000001'::uuid, 'ff782c9d-f2a3-42b3-9ba8-c16a3f1354e1'::uuid, 'Load Test Concert 2', 'load-test-concert-2', 'Another high volume event', 'Test Arena', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true),
    ('d0000000-0000-0000-0000-000000000003'::uuid, '00000000-0000-0000-0000-000000000001'::uuid, 'ff782c9d-f2a3-42b3-9ba8-c16a3f1354e1'::uuid, 'Load Test Concert 3', 'load-test-concert-3', 'Third load test event', 'Test Hall', 'Bangkok', 'published', 10, NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true)
ON CONFLICT DO NOTHING;

-- Insert test shows (3 shows per event = 9 shows total)
INSERT INTO shows (id, event_id, name, show_date, start_time, end_time, doors_open_at, status, total_capacity)
VALUES
    -- Event 1 shows
    ('c0000000-0000-0000-0001-000000000001'::uuid, 'd0000000-0000-0000-0000-000000000001'::uuid, 'Show 1 - Evening', (CURRENT_DATE + INTERVAL '7 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0001-000000000002'::uuid, 'd0000000-0000-0000-0000-000000000001'::uuid, 'Show 2 - Evening', (CURRENT_DATE + INTERVAL '14 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0001-000000000003'::uuid, 'd0000000-0000-0000-0000-000000000001'::uuid, 'Show 3 - Evening', (CURRENT_DATE + INTERVAL '21 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    -- Event 2 shows
    ('c0000000-0000-0000-0002-000000000001'::uuid, 'd0000000-0000-0000-0000-000000000002'::uuid, 'Show 1', (CURRENT_DATE + INTERVAL '8 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0002-000000000002'::uuid, 'd0000000-0000-0000-0000-000000000002'::uuid, 'Show 2', (CURRENT_DATE + INTERVAL '15 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0002-000000000003'::uuid, 'd0000000-0000-0000-0000-000000000002'::uuid, 'Show 3', (CURRENT_DATE + INTERVAL '22 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    -- Event 3 shows
    ('c0000000-0000-0000-0003-000000000001'::uuid, 'd0000000-0000-0000-0000-000000000003'::uuid, 'Show 1', (CURRENT_DATE + INTERVAL '9 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0003-000000000002'::uuid, 'd0000000-0000-0000-0000-000000000003'::uuid, 'Show 2', (CURRENT_DATE + INTERVAL '16 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000),
    ('c0000000-0000-0000-0003-000000000003'::uuid, 'd0000000-0000-0000-0000-000000000003'::uuid, 'Show 3', (CURRENT_DATE + INTERVAL '23 days')::date, '19:00:00+07', '22:00:00+07', '17:00:00+07', 'on_sale', 100000)
ON CONFLICT DO NOTHING;

-- Insert seat zones (5 zones per show = 45 zones total)
-- Each zone has 20,000 seats to handle 10k RPS
-- Total: 45 zones x 20,000 seats = 900,000 seats

-- Generate zones for all 9 shows
DO $$
DECLARE
    show_ids UUID[] := ARRAY[
        'c0000000-0000-0000-0001-000000000001'::uuid,
        'c0000000-0000-0000-0001-000000000002'::uuid,
        'c0000000-0000-0000-0001-000000000003'::uuid,
        'c0000000-0000-0000-0002-000000000001'::uuid,
        'c0000000-0000-0000-0002-000000000002'::uuid,
        'c0000000-0000-0000-0002-000000000003'::uuid,
        'c0000000-0000-0000-0003-000000000001'::uuid,
        'c0000000-0000-0000-0003-000000000002'::uuid,
        'c0000000-0000-0000-0003-000000000003'::uuid
    ];
    zone_names TEXT[] := ARRAY['VIP', 'Gold', 'Silver', 'Bronze', 'Standing'];
    zone_prices NUMERIC[] := ARRAY[5000.00, 3000.00, 2000.00, 1000.00, 500.00];
    zone_colors TEXT[] := ARRAY['#FFD700', '#FFA500', '#C0C0C0', '#CD7F32', '#90EE90'];
    show_id UUID;
    show_idx INT;
    zone_idx INT;
    zone_id UUID;
BEGIN
    FOR show_idx IN 1..9 LOOP
        show_id := show_ids[show_idx];
        FOR zone_idx IN 1..5 LOOP
            -- Generate deterministic UUID: b0000000-0000-{show_idx:04d}-{zone_idx:04d}-000000000000
            zone_id := ('b0000000-0000-' || LPAD(show_idx::text, 4, '0') || '-' || LPAD(zone_idx::text, 4, '0') || '-000000000000')::uuid;

            INSERT INTO seat_zones (id, show_id, name, description, color, price, currency, total_seats, available_seats, min_per_order, max_per_order, is_active, sort_order)
            VALUES (
                zone_id,
                show_id,
                zone_names[zone_idx],
                'Load test zone - ' || zone_names[zone_idx],
                zone_colors[zone_idx],
                zone_prices[zone_idx],
                'THB',
                250000,  -- 250,000 seats per zone
                250000,  -- All available
                1,
                10,
                true,
                zone_idx
            )
            ON CONFLICT DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- Summary
SELECT
    'Ticket DB Seed Summary' as info,
    (SELECT COUNT(*) FROM events WHERE id::text LIKE 'd0000000-%') as test_events,
    (SELECT COUNT(*) FROM shows WHERE id::text LIKE 'c0000000-%') as test_shows,
    (SELECT COUNT(*) FROM seat_zones WHERE id::text LIKE 'b0000000-%') as test_zones,
    (SELECT SUM(available_seats) FROM seat_zones WHERE id::text LIKE 'b0000000-%') as total_available_seats;
