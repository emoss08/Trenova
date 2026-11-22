package slotstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/shared/redis"
)

type Bound struct {
	Key   string
	Limit int
}

type Claim struct {
	Bounds []Bound
	Token  string
	TTL    time.Duration
}

var ErrIncompleteClaim = errors.New(
	"a slot claim needs at least one bound with a limit, a token and a TTL",
)

var acquireScript = redis.NewScript(`
local now = redis.call('TIME')
local nowMs = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local ttl = tonumber(ARGV[1])
local token = ARGV[2]
for i, key in ipairs(KEYS) do
  redis.call('ZREMRANGEBYSCORE', key, '-inf', nowMs)
end
for i, key in ipairs(KEYS) do
  if not redis.call('ZSCORE', key, token) then
    if redis.call('ZCARD', key) >= tonumber(ARGV[2 + i]) then
      return 0
    end
  end
end
for i, key in ipairs(KEYS) do
  redis.call('ZADD', key, nowMs + ttl, token)
  if redis.call('PTTL', key) < ttl then
    redis.call('PEXPIRE', key, ttl)
  end
end
return 1
`)

var refreshScript = redis.NewScript(`
local now = redis.call('TIME')
local nowMs = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local ttl = tonumber(ARGV[1])
local token = ARGV[2]
for i, key in ipairs(KEYS) do
  if not redis.call('ZSCORE', key, token) then
    return 0
  end
end
for i, key in ipairs(KEYS) do
  redis.call('ZADD', key, 'XX', nowMs + ttl, token)
  if redis.call('PTTL', key) < ttl then
    redis.call('PEXPIRE', key, ttl)
  end
end
return 1
`)

type Store struct {
	client *redis.Client
}

func New(client *redis.Client) *Store {
	return &Store{client: client}
}

func (c Claim) valid() bool {
	if c.Token == "" || c.TTL <= 0 || len(c.Bounds) == 0 {
		return false
	}
	for _, bound := range c.Bounds {
		if bound.Key == "" || bound.Limit <= 0 {
			return false
		}
	}
	return true
}

func (c Claim) keys() []string {
	keys := make([]string, len(c.Bounds))
	for i, bound := range c.Bounds {
		keys[i] = bound.Key
	}
	return keys
}

func (s *Store) Acquire(ctx context.Context, claim Claim) (bool, error) {
	if !claim.valid() {
		return false, fmt.Errorf("acquire slot: %w", ErrIncompleteClaim)
	}
	args := make([]any, 0, 2+len(claim.Bounds))
	args = append(args, claim.TTL.Milliseconds(), claim.Token)
	for _, bound := range claim.Bounds {
		args = append(args, strconv.Itoa(bound.Limit))
	}
	acquired, err := acquireScript.Run(ctx, s.client, claim.keys(), args...).Int()
	if err != nil {
		return false, fmt.Errorf("acquire slot: %w", err)
	}
	return acquired == 1, nil
}

func (s *Store) Refresh(ctx context.Context, claim Claim) (bool, error) {
	if claim.Token == "" || claim.TTL <= 0 || len(claim.Bounds) == 0 {
		return false, fmt.Errorf("refresh slot: %w", ErrIncompleteClaim)
	}
	held, err := refreshScript.Run(
		ctx,
		s.client,
		claim.keys(),
		claim.TTL.Milliseconds(),
		claim.Token,
	).Int()
	if err != nil {
		return false, fmt.Errorf("refresh slot: %w", err)
	}
	return held == 1, nil
}

func (s *Store) Release(ctx context.Context, claim Claim) error {
	if claim.Token == "" || len(claim.Bounds) == 0 {
		return nil
	}
	_, err := s.client.Pipelined(ctx, func(pipe *redis.Pipeline) error {
		for _, key := range claim.keys() {
			pipe.ZRem(ctx, key, claim.Token)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("release slot: %w", err)
	}
	return nil
}
