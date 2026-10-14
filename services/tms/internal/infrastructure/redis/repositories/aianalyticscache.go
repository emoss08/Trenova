package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	aiAnalyticsPrefix    = "ai-analytics"
	aiAnalyticsEpochPart = "epoch"
	AIAnalyticsCacheTTL  = 2 * time.Minute
	aiAnalyticsEpochTTL  = 7 * 24 * time.Hour
)

type AIAnalyticsCacheParams struct {
	fx.In

	Client *redis.Client
}

type aiAnalyticsCache struct {
	client *redis.Client
}

func NewAIAnalyticsCache(p AIAnalyticsCacheParams) aianalyticsports.AnalyticsCache {
	return &aiAnalyticsCache{client: p.Client}
}

func AIAnalyticsEpochKey(tenant pagination.TenantInfo) string {
	return strings.Join([]string{
		aiAnalyticsPrefix,
		aiAnalyticsEpochPart,
		tenant.OrgID.String(),
		tenant.BuID.String(),
	}, ":")
}

func AIAnalyticsKey(key *aianalyticsports.CacheKey) string {
	req := key.Request
	return strings.Join([]string{
		aiAnalyticsPrefix,
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		strconv.FormatInt(key.Epoch, 10),
		string(req.Window),
		string(req.CompareBy),
		req.AgentDefinitionID.String(),
		string(req.Task),
		req.Timezone,
		key.Locale,
	}, ":")
}

func (c *aiAnalyticsCache) Epoch(ctx context.Context, tenant pagination.TenantInfo) (int64, error) {
	value, err := c.client.Get(ctx, AIAnalyticsEpochKey(tenant)).Int64()
	if err != nil {
		if redishelpers.IsRedisNil(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read ai analytics epoch: %w", err)
	}
	return value, nil
}

func (c *aiAnalyticsCache) BumpEpoch(ctx context.Context, tenant pagination.TenantInfo) error {
	key := AIAnalyticsEpochKey(tenant)
	pipe := c.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, aiAnalyticsEpochTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("bump ai analytics epoch: %w", err)
	}
	return nil
}

func (c *aiAnalyticsCache) Get(
	ctx context.Context,
	key *aianalyticsports.CacheKey,
	dest *aianalyticsports.Analytics,
) (bool, error) {
	err := redishelpers.GetStringJSON(ctx, c.client, AIAnalyticsKey(key), dest)
	if err == nil {
		return true, nil
	}
	if redishelpers.IsRedisNil(err) || errors.Is(err, redis.Nil) {
		return false, nil
	}
	return false, fmt.Errorf("read ai analytics: %w", err)
}

func (c *aiAnalyticsCache) Set(
	ctx context.Context,
	key *aianalyticsports.CacheKey,
	value *aianalyticsports.Analytics,
) error {
	if err := redishelpers.SetStringJSON(
		ctx,
		c.client,
		AIAnalyticsKey(key),
		value,
		AIAnalyticsCacheTTL,
	); err != nil {
		return fmt.Errorf("write ai analytics: %w", err)
	}
	return nil
}
