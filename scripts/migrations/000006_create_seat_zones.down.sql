-- 000006_create_seat_zones.down.sql

DROP TRIGGER IF EXISTS update_show_capacity_trigger ON seat_zones;
DROP FUNCTION IF EXISTS update_show_capacity();
DROP TRIGGER IF EXISTS update_seat_zones_updated_at ON seat_zones;
DROP TABLE IF EXISTS seat_zones;
