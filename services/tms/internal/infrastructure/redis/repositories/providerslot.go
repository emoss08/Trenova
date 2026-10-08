package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const providerSlotPrefix = "ai:slots"

var errIncompleteSlotRequest = errors.New("provider slot request is incomplete")

var providerSlotAcquireScript = redis.NewScript(`
local now = redis.call('TIME')
local nowMs = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local limit = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', nowMs)
if redis.call('ZSCORE', KEYS[1], ARGV[3]) then
  redis.call('ZADD', KEYS[1], nowMs + ttl, ARGV[3])
  return 1
end
if redis.call('ZCARD', KEYS[1]) >= limit then
  return 0
end
redis.call('ZADD', KEYS[1], nowMs + ttl, ARGV[3])
if redis.call('PTTL', KEYS[1]) < ttl then
  redis.call('PEXPIRE', KEYS[1], ttl)
end
return 1
`)

var providerSlotRefreshScript = redis.NewScript(`
if not redis.call('ZSCORE', KEYS[1], ARGV[2]) then
  return 0
end
local now = redis.call('TIME')
local nowMs = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local ttl = tonumber(ARGV[1])
redis.call('ZADD', KEYS[1], 'XX', nowMs + ttl, ARGV[2])
if redis.call('PTTL', KEYS[1]) < ttl then
  redis.call('PEXPIRE', KEYS[1], ttl)
end
return 1
`)

type ProviderSlotParams struct {
	fx.In

	Client *redis.Client
}

type providerSlotRepository struct {
	client *redis.Client
}

func NewProviderSlotRepository(p ProviderSlotParams) repositories.ProviderSlotRepository {
	return &providerSlotRepository{client: p.Client}
}

func providerSlotKey(providerID pulid.ID) string {
	return providerSlotPrefix + ":" + providerID.String()
}

func (r *providerSlotRepository) Acquire(
	ctx context.Context,
	req repositories.AcquireProviderSlotRequest,
) (bool, error) {
	if req.ProviderID.IsNil() || req.Token == "" || req.Limit <= 0 || req.TTL <= 0 {
		return false, fmt.Errorf("acquire provider slot: %w", errIncompleteSlotRequest)
	}

	acquired, err := providerSlotAcquireScript.Run(
		ctx,
		r.client,
		[]string{providerSlotKey(req.ProviderID)},
		req.Limit,
		req.TTL.Milliseconds(),
		req.Token,
	).Int()
	if err != nil {
		return false, fmt.Errorf("acquire provider slot: %w", err)
	}

	return acquired == 1, nil
}

func (r *providerSlotRepository) Refresh(
	ctx context.Context,
	req repositories.ProviderSlotRequest,
) (bool, error) {
	if req.ProviderID.IsNil() || req.Token == "" || req.TTL <= 0 {
		return false, fmt.Errorf("refresh provider slot: %w", errIncompleteSlotRequest)
	}

	held, err := providerSlotRefreshScript.Run(
		ctx,
		r.client,
		[]string{providerSlotKey(req.ProviderID)},
		req.TTL.Milliseconds(),
		req.Token,
	).Int()
	if err != nil {
		return false, fmt.Errorf("refresh provider slot: %w", err)
	}

	return held == 1, nil
}

func (r *providerSlotRepository) Release(
	ctx context.Context,
	req repositories.ProviderSlotRequest,
) error {
	if req.ProviderID.IsNil() || req.Token == "" {
		return nil
	}

	if err := r.client.ZRem(ctx, providerSlotKey(req.ProviderID), req.Token).Err(); err != nil {
		return fmt.Errorf("release provider slot: %w", err)
	}

	return nil
}
