-- Fixed window rate limiting script with improved consistency
-- KEYS[1]: rate limit counter key
-- KEYS[2]: window key (timestamp when the current window started)
-- ARGV[1]: max requests allowed
-- ARGV[2]: window size in seconds
-- ARGV[3]: current timestamp

local counter_key = KEYS[1]
local window_key = KEYS[2]
local max_requests = tonumber(ARGV[1])
local window_size_sec = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Use a transaction-like approach with a single combined key
-- Store both window timestamp and counter in a hash
local data_key = counter_key .. ":data"

-- Get current window data
local window_data = redis.call('HMGET', data_key, 'window_start', 'counter')
local window_start = window_data[1] and tonumber(window_data[1])
local counter = window_data[2] and tonumber(window_data[2]) or 0

-- Check if we need to create or reset the window
local needs_reset = false
if not window_start then
    needs_reset = true
else
    -- Calculate window expiration with 1-second grace period for clock skew
    local window_expires = window_start + window_size_sec
    if now >= window_expires - 1 then
        needs_reset = true
    end
end

if needs_reset then
    -- Reset window with atomic operation
    redis.call('HMSET', data_key, 'window_start', now, 'counter', 1)
    redis.call('EXPIRE', data_key, window_size_sec + 2) -- Add 2 seconds buffer
    
    -- Clean up old keys if they exist (migration support)
    redis.call('DEL', counter_key, window_key)
    
    return { 1, max_requests - 1, now + window_size_sec }
end

-- Window is still active
local window_expires = window_start + window_size_sec

-- Check rate limit
if counter < max_requests then
    -- Increment counter atomically
    redis.call('HINCRBY', data_key, 'counter', 1)
    
    -- Update expiration to match window end (with buffer)
    local remaining_ttl = window_expires - now + 2
    if remaining_ttl > 0 then
        redis.call('EXPIRE', data_key, remaining_ttl)
    end
    
    return { 1, max_requests - counter - 1, window_expires }
else
    -- Rate limit exceeded
    return { 0, 0, window_expires }
end
