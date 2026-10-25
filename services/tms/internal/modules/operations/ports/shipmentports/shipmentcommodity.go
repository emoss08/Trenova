package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

type ShipmentCommodityRepository interface {
	SyncForShipment(
		ctx context.Context,
		entity *shipment.Shipment,
	) error
}
