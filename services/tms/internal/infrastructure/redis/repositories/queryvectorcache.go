package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/vectorutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const queryVectorPrefix = "retrieval-query-vector:v1"

type QueryVectorCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type queryVectorCache struct {
	client *redis.Client
	l      *zap.Logger
}

type cachedQueryVector struct {
	ModelKey   string `json:"m"`
	Dimensions int    `json:"d"`
	Vector     []byte `json:"v"`
}

func NewQueryVectorCache(p QueryVectorCacheParams) agentports.QueryVectorCache {
	return &queryVectorCache{
		client: p.Client,
		l:      p.Logger.Named("repository.query-vector-cache"),
	}
}

func (c *queryVectorCache) key(key agentports.QueryVectorCacheKey) string {
	return fmt.Sprintf(
		"%s:%s:%s:%s:%s",
		queryVectorPrefix,
		key.TenantInfo.OrgID,
		key.TenantInfo.BuID,
		key.ModelKey,
		key.TextHash,
	)
}

func (c *queryVectorCache) Get(
	ctx context.Context,
	key agentports.QueryVectorCacheKey,
) (agentports.QueryVector, bool, error) {
	raw, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return agentports.QueryVector{}, false, nil
		}

		return agentports.QueryVector{}, false, fmt.Errorf("read query vector: %w", err)
	}

	var cached cachedQueryVector
	if err = sonic.Unmarshal(raw, &cached); err != nil {
		c.drop(ctx, key, err)

		return agentports.QueryVector{}, false, nil
	}
	vector, err := vectorutils.Unpack(cached.Vector)
	if err != nil || cached.ModelKey != key.ModelKey ||
		!airetrieval.IsAllowedDimension(cached.Dimensions) || len(vector) != cached.Dimensions {
		c.drop(ctx, key, err)

		return agentports.QueryVector{}, false, nil
	}

	return agentports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   cached.ModelKey,
		Dimensions: cached.Dimensions,
	}, true, nil
}

func (c *queryVectorCache) Set(
	ctx context.Context,
	key agentports.QueryVectorCacheKey,
	vector agentports.QueryVector,
	ttl time.Duration,
) error {
	if !vector.Usable() {
		return nil
	}
	payload, err := sonic.Marshal(cachedQueryVector{
		ModelKey:   vector.ModelKey,
		Dimensions: vector.Dimensions,
		Vector:     vectorutils.Pack(vector.Vector),
	})
	if err != nil {
		return fmt.Errorf("encode query vector: %w", err)
	}
	if err = c.client.Set(ctx, c.key(key), payload, ttl).Err(); err != nil {
		return fmt.Errorf("write query vector: %w", err)
	}

	return nil
}

func (c *queryVectorCache) drop(ctx context.Context, key agentports.QueryVectorCacheKey, cause error) {
	c.l.Warn("dropping an unreadable cached query vector", zap.Error(cause))
	if err := c.client.Del(ctx, c.key(key)).Err(); err != nil {
		c.l.Warn("could not drop an unreadable cached query vector", zap.Error(err))
	}
}
