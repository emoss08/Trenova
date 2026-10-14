package aianalyticsports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type CacheKey struct {
	Request *AnalyticsRequest
	Epoch   int64
	Locale  string
}

type AnalyticsCache interface {
	Epoch(ctx context.Context, tenant pagination.TenantInfo) (int64, error)
	BumpEpoch(ctx context.Context, tenant pagination.TenantInfo) error
	Get(ctx context.Context, key *CacheKey, dest *Analytics) (bool, error)
	Set(ctx context.Context, key *CacheKey, value *Analytics) error
}
