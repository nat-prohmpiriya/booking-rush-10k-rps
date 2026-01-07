-- 000005_add_available_seats_check.up.sql
-- Add CHECK constraint to prevent negative available_seats (Layer 3 of Zero Overselling)
-- This is the last line of defense against overselling bugs

-- Add CHECK constraint to ensure available_seats is never negative
ALTER TABLE seat_zones
ADD CONSTRAINT chk_available_seats_non_negative
CHECK (available_seats >= 0);

-- Add CHECK constraint to ensure reserved_seats is never negative
ALTER TABLE seat_zones
ADD CONSTRAINT chk_reserved_seats_non_negative
CHECK (reserved_seats >= 0);

-- Add CHECK constraint to ensure sold_seats is never negative
ALTER TABLE seat_zones
ADD CONSTRAINT chk_sold_seats_non_negative
CHECK (sold_seats >= 0);

-- Add CHECK constraint to ensure total_seats is positive
ALTER TABLE seat_zones
ADD CONSTRAINT chk_total_seats_positive
CHECK (total_seats > 0);

-- Add CHECK constraint for logical consistency:
-- available + reserved + sold should not exceed total
ALTER TABLE seat_zones
ADD CONSTRAINT chk_seats_sum_valid
CHECK (available_seats + reserved_seats + sold_seats <= total_seats);

COMMENT ON CONSTRAINT chk_available_seats_non_negative ON seat_zones IS
'Layer 3 defense: Prevents overselling by rejecting transactions that would make available_seats negative';
