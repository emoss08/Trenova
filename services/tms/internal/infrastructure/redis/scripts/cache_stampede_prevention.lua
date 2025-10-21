-- Cache stampede prevention with distributed locking
-- KEYS[1]: lock key
-- KEYS[2]: data key
-- ARGV[1]: lock timeout in seconds
-- ARGV[2]: data value (already serialized)
-- ARGV[3]: data TTL in seconds
-- Returns: 1 if lock acquired and data set, 0 if lock not acquired

local lock_key = KEYS[1]
local data_key = KEYS[2]
local lock_timeout = tonumber(ARGV[1])
local data_value = ARGV[2]
local data_ttl = tonumber(ARGV[3])

local lock_acquired = redis.call('SET', lock_key, '1', 'NX', 'EX', lock_timeout)

if lock_acquired then
    if data_value then
        redis.call('SETEX', data_key, data_ttl, data_value)
    end
    redis.call('DEL', lock_key)
    return 1
else
    return 0
end