--[[
    Release Seats Lua Script
    ========================
    Atomically releases reserved seats back to inventory.

    Key Structure:
    - KEYS[1]: zone:availability:{zone_id}           - Available seats count (string/integer)
    - KEYS[2]: user:reservations:{user_id}:{event_id} - User's total reserved for this event
    - KEYS[3]: reservation:{booking_id}              - Reservation record (hash)

    Arguments:
    - ARGV[1]: booking_id        - Booking ID (for validation)
    - ARGV[2]: user_id           - User ID (for validation)

    Returns:
    - Success: {1, new_available_seats, new_user_reserved}
    - Error: {0, error_code, error_message}

    Error Codes:
    - RESERVATION_NOT_FOUND: Reservation record does not exist
    - INVALID_BOOKING_ID: Booking ID does not match
    - INVALID_USER_ID: User ID does not match
    - ALREADY_RELEASED: Reservation already released or confirmed
--]]

local zone_availability_key = KEYS[1]
local user_reservations_key = KEYS[2]
local reservation_key = KEYS[3]

local booking_id = ARGV[1]
local user_id = ARGV[2]

-- Get reservation record
local reservation = redis.call("HGETALL", reservation_key)
if #reservation == 0 then
    return {0, "RESERVATION_NOT_FOUND", "Reservation does not exist or has expired"}
end

-- Convert HGETALL result to table
local reservation_data = {}
for i = 1, #reservation, 2 do
    reservation_data[reservation[i]] = reservation[i + 1]
end

-- Validate booking_id
if reservation_data["booking_id"] ~= booking_id then
    return {0, "INVALID_BOOKING_ID", "Booking ID does not match"}
end

-- Validate user_id
if reservation_data["user_id"] ~= user_id then
    return {0, "INVALID_USER_ID", "User ID does not match"}
end

-- Check if already released or confirmed
local status = reservation_data["status"]
if status ~= "reserved" then
    return {0, "ALREADY_RELEASED", "Reservation status is '" .. (status or "unknown") .. "', cannot release"}
end

-- Get quantity from reservation
local quantity = tonumber(reservation_data["quantity"])
if not quantity or quantity <= 0 then
    return {0, "INVALID_QUANTITY", "Invalid quantity in reservation"}
end

-- === ATOMIC RELEASE ===

-- 1. Increment seats back to availability (INCRBY)
local new_available = redis.call("INCRBY", zone_availability_key, quantity)

-- 2. Decrement user's reserved count
local current_user_reserved = redis.call("GET", user_reservations_key)
current_user_reserved = tonumber(current_user_reserved) or 0

local new_user_reserved = current_user_reserved - quantity
if new_user_reserved < 0 then
    new_user_reserved = 0
end

if new_user_reserved > 0 then
    redis.call("SET", user_reservations_key, new_user_reserved)
    -- Keep the same TTL as before
    redis.call("EXPIRE", user_reservations_key, 660) -- 10 min + 1 min buffer
else
    -- If user has no more reservations, delete the key
    redis.call("DEL", user_reservations_key)
end

-- 3. Delete reservation record
redis.call("DEL", reservation_key)

-- Return success with new available seats and user's new reserved count
return {1, new_available, new_user_reserved}
