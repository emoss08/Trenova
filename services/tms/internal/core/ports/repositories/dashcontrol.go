package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type DashControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*tenant.DashControl, error)
	Update(
		ctx context.Context,
		entity *tenant.DashControl,
	) (*tenant.DashControl, error)
}
