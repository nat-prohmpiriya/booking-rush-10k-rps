--[[
    Reserve Seats Lua Script
    ========================
    Atomically reserves seats for a booking.
    
    Key Structure:
    - KEYS[1]: zone:availability:{zone_id}      - Available seats count (string/integer)
    - KEYS[2]: user:reservations:{user_id}:{event_id} - User's total reserved for this event
    - KEYS[3]: reservation:{booking_id}         - Reservation record (hash)
    
    Arguments:
    - ARGV[1]: quantity           - Number of seats to reserve
    - ARGV[2]: max_per_user       - Maximum seats allowed per user per event
    - ARGV[3]: user_id            - User ID
    - ARGV[4]: booking_id         - Booking ID (for reservation record)
    - ARGV[5]: zone_id            - Zone ID
    - ARGV[6]: event_id           - Event ID
    - ARGV[7]: show_id            - Show ID
    - ARGV[8]: unit_price         - Price per seat
    - ARGV[9]: ttl_seconds        - Reservation TTL (default 600 = 10 min)
    
    Returns:
    - Success: {1, remaining_seats, total_user_reserved}
    - Error: {0, error_code, error_message}
    
    Error Codes:
    - INSUFFICIENT_STOCK: Not enough seats available
    - USER_LIMIT_EXCEEDED: User has reached max reservation limit
    - INVALID_QUANTITY: Quantity must be positive
    - ZONE_NOT_FOUND: Zone availability key not found
--]]

local zone_availability_key = KEYS[1]
local user_reservations_key = KEYS[2]
local reservation_key = KEYS[3]

local quantity = tonumber(ARGV[1])
local max_per_user = tonumber(ARGV[2])
local user_id = ARGV[3]
local booking_id = ARGV[4]
local zone_id = ARGV[5]
local event_id = ARGV[6]
local show_id = ARGV[7]
local unit_price = ARGV[8]
local ttl_seconds = tonumber(ARGV[9]) or 600

-- Validate quantity
if not quantity or quantity <= 0 then
    return {0, "INVALID_QUANTITY", "Quantity must be a positive number"}
end

-- Get current available seats
local available = redis.call("GET", zone_availability_key)
if not available then
    return {0, "ZONE_NOT_FOUND", "Zone availability not initialized"}
end
available = tonumber(available)

-- Check seat availability
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats available. Available: " .. available .. ", Requested: " .. quantity}
end

-- Get user's current reservations for this event
local user_reserved = redis.call("GET", user_reservations_key)
user_reserved = tonumber(user_reserved) or 0

-- Check user limit
if max_per_user and max_per_user > 0 then
    if (user_reserved + quantity) > max_per_user then
        return {0, "USER_LIMIT_EXCEEDED", "User limit exceeded. Current: " .. user_reserved .. ", Requested: " .. quantity .. ", Max: " .. max_per_user}
    end
end

-- === ATOMIC RESERVATION ===

-- 1. Deduct seats from availability
local remaining = redis.call("DECRBY", zone_availability_key, quantity)

-- 2. Increment user's reserved count for this event
local new_user_reserved = redis.call("INCRBY", user_reservations_key, quantity)

-- 3. Set expiry on user reservation key (same as booking TTL + buffer)
redis.call("EXPIRE", user_reservations_key, ttl_seconds + 60)

-- 4. Create reservation record
local timestamp = redis.call("TIME")
local created_at = timestamp[1] .. "." .. timestamp[2]

redis.call("HSET", reservation_key,
    "booking_id", booking_id,
    "user_id", user_id,
    "zone_id", zone_id,
    "event_id", event_id,
    "show_id", show_id,
    "quantity", quantity,
    "unit_price", unit_price,
    "status", "reserved",
    "created_at", created_at,
    "expires_at", timestamp[1] + ttl_seconds
)

-- 5. Set TTL on reservation
redis.call("EXPIRE", reservation_key, ttl_seconds)

-- Return success with remaining seats and user's total reserved
return {1, remaining, new_user_reserved}
