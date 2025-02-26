-- Sliding window rate limiting script
-- KEYS[1]: rate limit key
-- ARGV[1]: max requests
-- ARGV[2]: window size in seconds
-- ARGV[3]: current timestamp

local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window_size_sec = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove expired timestamps
redis.call('ZREMRANGEBYSCORE', key, 0, now - window_size_sec)

-- Count current requests in window
local count = redis.call('ZCARD', key)

-- Check if under limit
if count < max_requests then
    -- Add current timestamp with score = timestamp
    redis.call('ZADD', key, now, now .. ':' .. math.random())
    -- Set expiration to ensure cleanup
    redis.call('EXPIRE', key, window_size_sec)
    return { 1, max_requests - count - 1, now + window_size_sec }
else
    -- Get oldest timestamp to calculate reset time
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')[2]
    local reset_time = tonumber(oldest) + window_size_sec
    return { 0, 0, reset_time }
end
