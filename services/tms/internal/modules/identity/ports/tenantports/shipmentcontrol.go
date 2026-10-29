package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetShipmentControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ShipmentControlRepository interface {
	Get(
		ctx context.Context,
		req GetShipmentControlRequest,
	) (*shipmentcontrol.ShipmentControl, error)
	Update(
		ctx context.Context,
		sc *shipmentcontrol.ShipmentControl,
	) (*shipmentcontrol.ShipmentControl, error)
	ListDetentionEngineTenants(
		ctx context.Context,
		limit int,
	) ([]pagination.TenantInfo, error)
}
