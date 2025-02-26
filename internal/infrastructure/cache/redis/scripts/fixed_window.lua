-- Fixed window rate limiting script
-- KEYS[1]: rate limit key
-- KEYS[2]: window key
-- ARGV[1]: max requests
-- ARGV[2]: window size in seconds
-- ARGV[3]: current timestamp

local key = KEYS[1]
local window_key = KEYS[2]
local max_requests = tonumber(ARGV[1])
local window_size_sec = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Check if window exists
local window = redis.call('GET', window_key)
local count = 0

if window then
    count = tonumber(redis.call('GET', key) or '0')
    local window_start = tonumber(window)
    local window_end = window_start + window_size_sec

    -- If window has expired, start a new one
    if now >= window_end then
        redis.call('SET', window_key, now, 'EX', window_size_sec)
        redis.call('SET', key, 1, 'EX', window_size_sec)
        return { 1, max_requests - 1, now + window_size_sec }
    end

    -- Check if under limit
    if count < max_requests then
        redis.call('INCR', key)
        return { 1, max_requests - count - 1, window_end }
    else
        return { 0, 0, window_end }
    end
else
    -- New window
    redis.call('SET', window_key, now, 'EX', window_size_sec)
    redis.call('SET', key, 1, 'EX', window_size_sec)
    return { 1, max_requests - 1, now + window_size_sec }
end
