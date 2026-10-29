package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type CarrierSettlementControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*carriersettlementcontrol.CarrierSettlementControl, error)
	ListAutoGenerate(
		ctx context.Context,
	) ([]*carriersettlementcontrol.CarrierSettlementControl, error)
	Update(
		ctx context.Context,
		entity *carriersettlementcontrol.CarrierSettlementControl,
	) (*carriersettlementcontrol.CarrierSettlementControl, error)
}
