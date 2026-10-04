package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

type QuotaCountRequest struct {
	TenantInfo  pagination.TenantInfo
	Meter       platformcatalog.MeterKey
	WindowStart int64
	WindowEnd   int64
}

type QuotaCounterRepository interface {
	Supports(meter platformcatalog.MeterKey) bool
	Lock(ctx context.Context, tenantInfo pagination.TenantInfo, meter platformcatalog.MeterKey) error
	Count(ctx context.Context, req *QuotaCountRequest) (int64, error)
	OrganizationTimezone(ctx context.Context, tenantInfo pagination.TenantInfo) (string, error)
}
