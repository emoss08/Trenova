-- Token bucket rate limiting script with precise refill calculations
-- KEYS[1]: tokens key
-- KEYS[2]: timestamp key
-- ARGV[1]: bucket size (maximum tokens)
-- ARGV[2]: refill rate (tokens per second)
-- ARGV[3]: current timestamp

local token_key = KEYS[1]
local timestamp_key = KEYS[2]
local bucket_size = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Get or initialize bucket
local tokens = redis.call('GET', token_key)
local last_update = redis.call('GET', timestamp_key)

if not tokens then
    -- Initialize to full bucket
    tokens = bucket_size
else
    tokens = tonumber(tokens)
end

if not last_update then
    last_update = now
else
    last_update = tonumber(last_update)
end

-- Calculate elapsed time since last update
local elapsed = math.max(0, now - last_update)

-- Calculate tokens to add based on time passed (with precise float calculation)
local new_tokens = math.min(bucket_size, tokens + (elapsed * refill_rate))

-- Try to consume a token
if new_tokens >= 1 then
    -- Update bucket with one less token
    local remaining_tokens = new_tokens - 1

    -- Calculate TTL based on how long until bucket is full again
    local ttl = math.ceil((bucket_size - remaining_tokens) / refill_rate) + 60

    -- Ensure minimum TTL
    ttl = math.max(ttl, 60)

    -- Update the tokens and timestamp
    redis.call('SET', token_key, remaining_tokens, 'EX', ttl)
    redis.call('SET', timestamp_key, now, 'EX', ttl)

    -- Calculate when bucket will be full again for reset time
    local time_to_full = (bucket_size - remaining_tokens) / refill_rate
    local reset_time = now + time_to_full

    return { 1, math.floor(remaining_tokens), math.ceil(reset_time) }
else
    -- Calculate when next token will be available
    local time_to_next = (1 - new_tokens) / refill_rate
    local reset_time = now + time_to_next

    -- Update timestamp but keep tokens the same
    redis.call('SET', token_key, new_tokens, 'EX', 60)
    redis.call('SET', timestamp_key, now, 'EX', 60)

    return { 0, 0, math.ceil(reset_time) }
end
