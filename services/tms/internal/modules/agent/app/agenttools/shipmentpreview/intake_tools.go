package shipmentpreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
)

// shipmentWriter is the slice of the shipment service the intake tools use.
type ShipmentWriter interface {
	Get(ctx context.Context, req *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
}
