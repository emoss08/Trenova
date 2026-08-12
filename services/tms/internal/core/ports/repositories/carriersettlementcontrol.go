package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type CarrierSettlementControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*tenant.CarrierSettlementControl, error)
	ListAutoGenerate(ctx context.Context) ([]*tenant.CarrierSettlementControl, error)
	Update(
		ctx context.Context,
		entity *tenant.CarrierSettlementControl,
	) (*tenant.CarrierSettlementControl, error)
}
