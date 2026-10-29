package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type SettlementControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*settlementcontrol.SettlementControl, error)
	ListAutoGenerate(ctx context.Context) ([]*settlementcontrol.SettlementControl, error)
	ListAll(ctx context.Context) ([]*settlementcontrol.SettlementControl, error)
	Update(
		ctx context.Context,
		entity *settlementcontrol.SettlementControl,
	) (*settlementcontrol.SettlementControl, error)
}
