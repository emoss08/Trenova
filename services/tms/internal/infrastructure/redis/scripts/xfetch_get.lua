-- XFetch algorithm for probabilistic early expiration
-- Prevents cache stampedes by probabilistically returning nil before actual expiration
-- Based on: https://www.vldb.org/pvldb/vol8/p886-vattani.pdf
--
-- KEYS[1]: data key to get
-- ARGV[1]: recomputation time in milliseconds (how long it takes to rebuild cache)
-- ARGV[2]: beta parameter (typically 1.0, higher = more aggressive early expiration)
-- Returns: data value or nil (with early expiration flag)

local data_key = KEYS[1]
local delta = tonumber(ARGV[1]) / 1000  -- Convert ms to seconds
local beta = tonumber(ARGV[2]) or 1.0

-- Get the data and its TTL
local data = redis.call('JSON.GET', data_key, '.')
if not data then
    return {false, nil}  -- No data found
end

-- Get remaining TTL
local ttl = redis.call('TTL', data_key)
if ttl < 0 then
    return {false, nil}  -- Key doesn't exist or has no TTL
end

-- Calculate XFetch probability
-- P(expire early) = delta * beta * log(random()) / ttl
-- This ensures probability increases as we approach actual expiration
local random_value = math.random()
local xfetch_value = delta * beta * math.log(random_value)

-- If xfetch_value >= -ttl, trigger early expiration
if xfetch_value >= -ttl then
    -- Return nil to trigger cache refresh
    -- But mark it as probabilistic expiration, not actual missing data
    return {true, nil}  -- true indicates early expiration triggered
end

-- Return the cached data
return {false, data}