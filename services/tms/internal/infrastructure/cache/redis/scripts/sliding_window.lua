-- Sliding window rate limiting script with more precise tracking
-- KEYS[1]: rate limit key (sorted set of timestamps)
-- ARGV[1]: max requests
-- ARGV[2]: window size in seconds
-- ARGV[3]: current timestamp

local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window_size_sec = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Calculate the cutoff time for the sliding window
local cutoff = now - window_size_sec

-- Remove expired timestamps
redis.call('ZREMRANGEBYSCORE', key, 0, cutoff)

-- Count current requests in window
local count = redis.call('ZCARD', key)

-- Check if under limit
if count < max_requests then
    -- Add current timestamp with score = timestamp
    -- Use a unique identifier (timestamp + random) to avoid collisions
    local unique_id = now .. ':' .. math.random()
    redis.call('ZADD', key, now, unique_id)

    -- Set expiration to ensure cleanup (a bit longer than window size)
    redis.call('EXPIRE', key, window_size_sec + 10)

    -- Calculate reset time - either when the window will be full or when oldest entry expires
    local reset_time
    if count > 0 then
        -- Get oldest timestamp to determine when one slot will free up
        local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')[2]
        reset_time = tonumber(oldest) + window_size_sec
    else
        -- No previous requests, so reset time is when this request expires
        reset_time = now + window_size_sec
    end

    return { 1, max_requests - count - 1, reset_time }
else
    -- Get oldest timestamp to calculate when one slot will free up
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')[2]
    local reset_time = tonumber(oldest) + window_size_sec

    return { 0, 0, reset_time }
end
