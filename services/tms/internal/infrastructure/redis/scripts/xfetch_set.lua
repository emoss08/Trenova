-- XFetch set with metadata for probabilistic expiration
-- Stores data along with recomputation time for XFetch algorithm
--
-- KEYS[1]: data key to set
-- KEYS[2]: metadata key (stores recomputation time)
-- ARGV[1]: data value (JSON)
-- ARGV[2]: TTL in seconds
-- ARGV[3]: recomputation time in milliseconds
-- Returns: OK

local data_key = KEYS[1]
local meta_key = KEYS[2]
local data_value = ARGV[1]
local ttl = tonumber(ARGV[2])
local recompute_time = ARGV[3]

-- Set the data with TTL
redis.call('JSON.SET', data_key, '.', data_value)
redis.call('EXPIRE', data_key, ttl)

-- Store recomputation time metadata
-- This is used by xfetch_get to calculate early expiration probability
redis.call('SET', meta_key, recompute_time, 'EX', ttl + 60)  -- Metadata lives slightly longer

return "OK"