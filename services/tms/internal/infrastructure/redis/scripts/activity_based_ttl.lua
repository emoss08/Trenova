-- Activity-based TTL calculation and cache update
-- KEYS[1]: activity counter key
-- KEYS[2]: data key to set
-- ARGV[1]: data value (JSON)
-- ARGV[2]: high activity threshold (e.g., 10)
-- ARGV[3]: medium activity threshold (e.g., 3)
-- ARGV[4]: high TTL in seconds (e.g., 1800 for 30 min)
-- ARGV[5]: medium TTL in seconds (e.g., 900 for 15 min)
-- ARGV[6]: low TTL in seconds (e.g., 300 for 5 min)
-- Returns: TTL that was set

local activity_key = KEYS[1]
local data_key = KEYS[2]
local data_value = ARGV[1]
local high_threshold = tonumber(ARGV[2])
local medium_threshold = tonumber(ARGV[3])
local high_ttl = tonumber(ARGV[4])
local medium_ttl = tonumber(ARGV[5])
local low_ttl = tonumber(ARGV[6])

-- Increment activity counter
local activity_count = redis.call('INCR', activity_key)

-- Set activity key expiry to 1 hour if it's new
if activity_count == 1 then
    redis.call('EXPIRE', activity_key, 3600)
end

-- Determine TTL based on activity
local ttl = low_ttl
if activity_count > high_threshold then
    ttl = high_ttl
elseif activity_count > medium_threshold then
    ttl = medium_ttl
end

-- Add jitter (0-60 seconds) to prevent synchronized expiration
local jitter = math.random(0, 60)
ttl = ttl + jitter

-- Set the data with calculated TTL
if data_value then
    redis.call('JSON.SET', data_key, '.', data_value)
    redis.call('EXPIRE', data_key, ttl)
end

return ttl