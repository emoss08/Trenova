package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
)

type AITuneUpFreshness interface {
	Fresh(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	Mark(ctx context.Context, tenantInfo pagination.TenantInfo, computedAt int64, ttl time.Duration) error
	Clear(ctx context.Context, tenantInfo pagination.TenantInfo) error
}
