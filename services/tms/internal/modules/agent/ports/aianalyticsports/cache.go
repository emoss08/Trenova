package aianalyticsports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type CacheKey struct {
	Request *AnalyticsRequest
	Epoch   int64
	Locale  string
	Buckets uint64
}

type AnalyticsCache interface {
	Epoch(ctx context.Context, tenant pagination.TenantInfo) (int64, error)
	BumpEpoch(ctx context.Context, tenant pagination.TenantInfo) error
	Get(ctx context.Context, key *CacheKey, dest *Analytics) (bool, error)
	Set(ctx context.Context, key *CacheKey, value *Analytics) error
	Palette(
		ctx context.Context,
		tenant pagination.TenantInfo,
		epoch int64,
	) ([]DeploymentKey, bool, error)
	SetPalette(
		ctx context.Context,
		tenant pagination.TenantInfo,
		epoch int64,
		keys []DeploymentKey,
	) error
}
