--[[
    Confirm Booking Lua Script
    ==========================
    Atomically confirms a reservation, making it permanent.

    Key Structure:
    - KEYS[1]: reservation:{booking_id}              - Reservation record (hash)

    Arguments:
    - ARGV[1]: booking_id        - Booking ID (for validation)
    - ARGV[2]: user_id           - User ID (for validation)
    - ARGV[3]: payment_id        - Payment ID (optional, for tracking)

    Returns:
    - Success: {1, "CONFIRMED", confirmed_at}
    - Error: {0, error_code, error_message}

    Error Codes:
    - RESERVATION_NOT_FOUND: Reservation record does not exist
    - INVALID_BOOKING_ID: Booking ID does not match
    - INVALID_USER_ID: User ID does not match
    - ALREADY_CONFIRMED: Reservation already confirmed
    - INVALID_STATUS: Reservation status is not 'reserved'
--]]

local reservation_key = KEYS[1]

local booking_id = ARGV[1]
local user_id = ARGV[2]
local payment_id = ARGV[3] or ""

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

-- Check current status
local status = reservation_data["status"]
if status == "confirmed" then
    return {0, "ALREADY_CONFIRMED", "Reservation is already confirmed"}
end

if status ~= "reserved" then
    return {0, "INVALID_STATUS", "Reservation status is '" .. (status or "unknown") .. "', expected 'reserved'"}
end

-- === ATOMIC CONFIRM ===

-- Get current timestamp
local timestamp = redis.call("TIME")
local confirmed_at = timestamp[1] .. "." .. timestamp[2]

-- 1. Update reservation status to confirmed
redis.call("HSET", reservation_key,
    "status", "confirmed",
    "confirmed_at", confirmed_at,
    "payment_id", payment_id
)

-- 2. Remove TTL - make reservation permanent
redis.call("PERSIST", reservation_key)

-- Return success with confirmation timestamp
return {1, "CONFIRMED", confirmed_at}
