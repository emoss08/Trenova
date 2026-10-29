package shipmentpreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
)

type ShipmentReader interface {
	Get(ctx context.Context, req *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error)
}

type ShipmentWriter interface {
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
}
