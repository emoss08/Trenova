package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
)

const (
	fieldsPerDecision = 4
	argsPerRequest    = 3
)

var gcraScript = redis.NewScript(`
local now_parts = redis.call('TIME')
local now = now_parts[1] * 1000000 + now_parts[2]
local n = #KEYS
local results = {}
local all_allowed = true

for i = 1, n do
  local base = (i - 1) * 3
  local interval = tonumber(ARGV[base + 1])
  local burst = tonumber(ARGV[base + 2])
  local cost = tonumber(ARGV[base + 3])

  local stored = redis.call('GET', KEYS[i])
  local tat = now
  if stored then
    tat = tonumber(stored)
    if tat < now then tat = now end
  end

  local new_tat = tat + interval * cost
  local allow_at = new_tat - interval * burst
  local diff = now - allow_at

  if diff < 0 then
    all_allowed = false
    results[i] = {0, 0, -diff, tat - now, tat}
  else
    results[i] = {1, math.floor(diff / interval), 0, new_tat - now, new_tat}
  end
end

local out = {}
for i = 1, n do
  local r = results[i]
  if all_allowed then
    local ttl_ms = math.ceil(r[4] / 1000)
    if ttl_ms < 1 then ttl_ms = 1 end
    redis.call('SET', KEYS[i], string.format('%.0f', r[5]), 'PX', ttl_ms)
  end
  out[#out + 1] = r[1]
  out[#out + 1] = r[2]
  out[#out + 1] = r[3]
  out[#out + 1] = r[4]
end
return out
`)

type RedisStore struct {
	client    redis.Scripter
	keyPrefix string
}

func NewRedisStore(client redis.Scripter, keyPrefix string) (*RedisStore, error) {
	if client == nil {
		return nil, ErrMissingClient
	}
	return &RedisStore{client: client, keyPrefix: keyPrefix}, nil
}

func (s *RedisStore) Check(
	ctx context.Context,
	requests []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	if err := validateRequests(requests); err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, nil
	}

	keys := make([]string, len(requests))
	args := make([]any, 0, len(requests)*argsPerRequest)
	for i := range requests {
		keys[i] = s.key(requests[i].Key)
		args = append(args,
			emissionInterval(requests[i].Policy),
			requests[i].Policy.Burst,
			max(requests[i].Cost, 1),
		)
	}

	raw, err := gcraScript.Run(ctx, s.client, keys, args...).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("ratelimit: run gcra script: %w", err)
	}
	if len(raw) != len(requests)*fieldsPerDecision {
		return nil, fmt.Errorf(
			"%w: expected %d values, got %d",
			ErrScriptResponse,
			len(requests)*fieldsPerDecision,
			len(raw),
		)
	}

	decisions := make([]repositories.RateLimitDecision, len(requests))
	for i := range requests {
		base := i * fieldsPerDecision
		decisions[i] = repositories.RateLimitDecision{
			Allowed:    raw[base] == 1,
			Limit:      requests[i].Policy.Burst,
			Remaining:  int(raw[base+1]),
			RetryAfter: time.Duration(raw[base+2]) * time.Microsecond,
			ResetAfter: time.Duration(raw[base+3]) * time.Microsecond,
		}
	}

	return decisions, nil
}

func (s *RedisStore) key(key string) string {
	if s.keyPrefix == "" {
		return key
	}
	return s.keyPrefix + ":" + key
}
