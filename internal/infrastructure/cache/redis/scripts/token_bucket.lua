-- Token bucket rate limiting script
-- KEYS[1]: tokens key
-- KEYS[2]: timestamp key
-- ARGV[1]: bucket size
-- ARGV[2]: refill rate
-- ARGV[3]: current timestamp

local key = KEYS[1]
local timestamp_key = KEYS[2]
local bucket_size = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Get or initialize bucket
local tokens = tonumber(redis.call('GET', key) or bucket_size)
local last_update = tonumber(redis.call('GET', timestamp_key) or now)

-- Calculate tokens to add based on time passed
local elapsed = now - last_update
local new_tokens = math.min(bucket_size, tokens + (elapsed * refill_rate))

-- Try to consume a token
if new_tokens >= 1 then
    -- Update bucket with one less token
    redis.call('SET', key, new_tokens - 1, 'EX', 86400) -- 24h expiry
    redis.call('SET', timestamp_key, now, 'EX', 86400)

    -- Calculate when bucket will be full again
    local time_to_full = (bucket_size - (new_tokens - 1)) / refill_rate
    local reset_time = now + time_to_full

    return { 1, math.floor(new_tokens - 1), math.floor(reset_time) }
else
    -- Calculate when next token will be available
    local time_to_next = (1 - new_tokens) / refill_rate
    local reset_time = now + time_to_next

    redis.call('SET', key, new_tokens, 'EX', 86400)
    redis.call('SET', timestamp_key, now, 'EX', 86400)

    return { 0, 0, math.floor(reset_time) }
end
