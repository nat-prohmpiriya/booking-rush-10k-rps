--[[
    Join Queue Lua Script
    =====================
    Atomically adds a user to the virtual queue using Sorted Set.

    Key Structure:
    - KEYS[1]: queue:{event_id}              - Sorted Set (score = timestamp, member = user_id)
    - KEYS[2]: queue:user:{event_id}:{user_id} - Hash with user queue info

    Arguments:
    - ARGV[1]: user_id           - User ID
    - ARGV[2]: event_id          - Event ID
    - ARGV[3]: token             - Unique queue token
    - ARGV[4]: ttl_seconds       - TTL for queue entry (default 1800 = 30 min)
    - ARGV[5]: max_queue_size    - Maximum queue size (0 = unlimited)

    Returns:
    - Success: {1, position, total_in_queue, joined_at_timestamp}
    - Error: {0, error_code, error_message}

    Error Codes:
    - ALREADY_IN_QUEUE: User is already in the queue
    - QUEUE_FULL: Queue has reached maximum capacity
--]]

local queue_key = KEYS[1]
local user_queue_key = KEYS[2]

local user_id = ARGV[1]
local event_id = ARGV[2]
local token = ARGV[3]
local ttl_seconds = tonumber(ARGV[4]) or 1800
local max_queue_size = tonumber(ARGV[5]) or 0

-- Check if user is already in queue
local existing_score = redis.call("ZSCORE", queue_key, user_id)
if existing_score then
    -- User is already in queue, return their position
    local position = redis.call("ZRANK", queue_key, user_id)
    local total = redis.call("ZCARD", queue_key)
    return {0, "ALREADY_IN_QUEUE", "User is already in queue at position " .. (position + 1)}
end

-- Check queue size limit
if max_queue_size > 0 then
    local current_size = redis.call("ZCARD", queue_key)
    if current_size >= max_queue_size then
        return {0, "QUEUE_FULL", "Queue has reached maximum capacity of " .. max_queue_size}
    end
end

-- Get current timestamp
local timestamp = redis.call("TIME")
local joined_at = tonumber(timestamp[1]) + (tonumber(timestamp[2]) / 1000000)

-- Add user to queue with timestamp as score
redis.call("ZADD", queue_key, joined_at, user_id)

-- Get user's position (0-indexed, so add 1 for human-readable)
local position = redis.call("ZRANK", queue_key, user_id)
local total = redis.call("ZCARD", queue_key)

-- Store user queue info
local expires_at = timestamp[1] + ttl_seconds
redis.call("HSET", user_queue_key,
    "user_id", user_id,
    "event_id", event_id,
    "token", token,
    "joined_at", joined_at,
    "expires_at", expires_at,
    "position", position + 1
)
redis.call("EXPIRE", user_queue_key, ttl_seconds)

-- Return success with position (1-indexed) and total
return {1, position + 1, total, joined_at}
