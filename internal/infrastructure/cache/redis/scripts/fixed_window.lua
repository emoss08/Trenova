-- Fixed window rate limiting script with robust window expiration
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

-- Get the window timestamp and counter atomically
local window_timestamp = redis.call('GET', window_key)
local counter = redis.call('GET', counter_key)

-- Initialize window if needed or check if it has expired
if not window_timestamp then
    -- No window exists yet, create a new one
    redis.call('SET', window_key, now, 'EX', window_size_sec)
    redis.call('SET', counter_key, 1, 'EX', window_size_sec)
    return { 1, max_requests - 1, now + window_size_sec }
end

-- Convert values to numbers
window_timestamp = tonumber(window_timestamp)
counter = counter and tonumber(counter) or 0

-- Calculate when the current window expires
local window_expires = window_timestamp + window_size_sec

-- Check if window has expired (allow a small buffer for clock skew)
if now >= window_expires then
    -- Window has expired, start a new one
    redis.call('SET', window_key, now, 'EX', window_size_sec)
    redis.call('SET', counter_key, 1, 'EX', window_size_sec)
    return { 1, max_requests - 1, now + window_size_sec }
end

-- Window is still active, check if we're within limits
if counter < max_requests then
    -- Still within limit, increment and return
    redis.call('INCR', counter_key)
    -- Ensure both keys have the same expiration
    local remaining_ttl = window_expires - now
    if remaining_ttl > 0 then
        redis.call('EXPIRE', counter_key, remaining_ttl)
        redis.call('EXPIRE', window_key, remaining_ttl)
    end
    return { 1, max_requests - counter - 1, window_expires }
else
    -- Rate limit exceeded
    return { 0, 0, window_expires }
end
