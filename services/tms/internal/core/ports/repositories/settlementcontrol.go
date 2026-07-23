package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type SettlementControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*tenant.SettlementControl, error)
	ListAutoGenerate(ctx context.Context) ([]*tenant.SettlementControl, error)
	ListAll(ctx context.Context) ([]*tenant.SettlementControl, error)
	Update(
		ctx context.Context,
		entity *tenant.SettlementControl,
	) (*tenant.SettlementControl, error)
}
