package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const aiTuneUpFreshPrefix = "ai-control:tune-ups:computed"

type AITuneUpFreshnessParams struct {
	fx.In

	Client *redis.Client
}

type aiTuneUpFreshness struct {
	client *redis.Client
}

func NewAITuneUpFreshness(p AITuneUpFreshnessParams) repositories.AITuneUpFreshness {
	return &aiTuneUpFreshness{client: p.Client}
}

func AITuneUpFreshKey(tenantInfo pagination.TenantInfo) string {
	return strings.Join([]string{
		aiTuneUpFreshPrefix,
		tenantInfo.OrgID.String(),
		tenantInfo.BuID.String(),
	}, ":")
}

func (f *aiTuneUpFreshness) Fresh(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	count, err := f.client.Exists(ctx, AITuneUpFreshKey(tenantInfo)).Result()
	if err != nil {
		return false, fmt.Errorf("read ai tune-up freshness: %w", err)
	}
	return count > 0, nil
}

func (f *aiTuneUpFreshness) Mark(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	computedAt int64,
	ttl time.Duration,
) error {
	if err := f.client.Set(ctx, AITuneUpFreshKey(tenantInfo), strconv.FormatInt(computedAt, 10), ttl).Err(); err != nil {
		return fmt.Errorf("mark ai tune-ups computed: %w", err)
	}
	return nil
}

func (f *aiTuneUpFreshness) Clear(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := f.client.Del(ctx, AITuneUpFreshKey(tenantInfo)).Err(); err != nil {
		return fmt.Errorf("clear ai tune-up freshness: %w", err)
	}
	return nil
}
