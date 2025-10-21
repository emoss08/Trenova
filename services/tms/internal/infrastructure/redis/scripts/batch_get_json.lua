-- Batch get JSON values from Redis
-- KEYS: array of keys to fetch
-- ARGV: none
-- Returns: array of JSON strings (nil for missing keys)

local results = {}

for i, key in ipairs(KEYS) do
    local value = redis.call('JSON.GET', key, '.')
    if value then
        results[i] = value
    else
        results[i] = false
    end
end

return results