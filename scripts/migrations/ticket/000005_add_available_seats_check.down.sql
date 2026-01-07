-- 000005_add_available_seats_check.down.sql
-- Remove CHECK constraints

ALTER TABLE seat_zones DROP CONSTRAINT IF EXISTS chk_seats_sum_valid;
ALTER TABLE seat_zones DROP CONSTRAINT IF EXISTS chk_total_seats_positive;
ALTER TABLE seat_zones DROP CONSTRAINT IF EXISTS chk_sold_seats_non_negative;
ALTER TABLE seat_zones DROP CONSTRAINT IF EXISTS chk_reserved_seats_non_negative;
ALTER TABLE seat_zones DROP CONSTRAINT IF EXISTS chk_available_seats_non_negative;
