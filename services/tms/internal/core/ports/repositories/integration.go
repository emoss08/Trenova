package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
)

type IntegrationRepository interface {
	ListByTenant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*integration.Integration, error)
	GetByType(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*integration.Integration, error)
	Upsert(
		ctx context.Context,
		entity *integration.Integration,
	) (*integration.Integration, error)
}
