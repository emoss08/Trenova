package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	aiControlSummaryPrefix = "ai-control:summary"
	aiControlClaimPart     = "claims"
	aiControlClaimTTL      = 26 * time.Hour
	claimDayLayout         = "20060102"
)

type AIControlSummaryCacheParams struct {
	fx.In

	Client *redis.Client
}

type aiControlSummaryCache struct {
	client *redis.Client
	now    func() time.Time
}

func NewAIControlSummaryCache(p AIControlSummaryCacheParams) repositories.AIControlSummaryCache {
	return &aiControlSummaryCache{client: p.Client, now: time.Now}
}

func AIControlSummaryKey(key *repositories.AIControlSummaryKey) string {
	return strings.Join([]string{
		aiControlSummaryPrefix,
		key.TenantInfo.OrgID.String(),
		key.TenantInfo.BuID.String(),
		string(key.Tab),
		key.FactsHash,
	}, ":")
}

func aiControlClaimKey(key *repositories.AIControlSummaryKey, day string) string {
	return strings.Join([]string{
		aiControlSummaryPrefix,
		aiControlClaimPart,
		key.TenantInfo.OrgID.String(),
		key.TenantInfo.BuID.String(),
		string(key.Tab),
		day,
	}, ":")
}

func (c *aiControlSummaryCache) Get(
	ctx context.Context,
	key *repositories.AIControlSummaryKey,
	dest *aicontrolsummary.Summary,
) (bool, error) {
	err := redishelpers.GetStringJSON(ctx, c.client, AIControlSummaryKey(key), dest)
	if err == nil {
		return true, nil
	}
	if redishelpers.IsRedisNil(err) || errors.Is(err, redis.Nil) {
		return false, nil
	}
	return false, fmt.Errorf("read ai control summary: %w", err)
}

func (c *aiControlSummaryCache) Set(
	ctx context.Context,
	key *repositories.AIControlSummaryKey,
	value *aicontrolsummary.Summary,
	ttl time.Duration,
) error {
	encoded, err := jsonutils.ToJSONString(value)
	if err != nil {
		return fmt.Errorf("encode ai control summary: %w", err)
	}
	if err = c.client.Set(ctx, AIControlSummaryKey(key), encoded, ttl).Err(); err != nil {
		return fmt.Errorf("write ai control summary: %w", err)
	}
	return nil
}

func (c *aiControlSummaryCache) ClaimNarration(
	ctx context.Context,
	key *repositories.AIControlSummaryKey,
	perDay int,
) (bool, error) {
	claim := aiControlClaimKey(key, c.now().UTC().Format(claimDayLayout))
	pipe := c.client.TxPipeline()
	count := pipe.Incr(ctx, claim)
	pipe.Expire(ctx, claim, aiControlClaimTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("claim ai control narration: %w", err)
	}
	return count.Val() <= int64(perDay), nil
}
