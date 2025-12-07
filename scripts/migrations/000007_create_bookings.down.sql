-- 000007_create_bookings.down.sql

DROP TRIGGER IF EXISTS generate_booking_confirmation_code ON bookings;
DROP FUNCTION IF EXISTS generate_confirmation_code();
DROP TRIGGER IF EXISTS update_bookings_updated_at ON bookings;
DROP TABLE IF EXISTS bookings;
DROP TYPE IF EXISTS booking_status;
