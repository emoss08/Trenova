package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/uptrace/bun"
)

type ShipmentCommodityRepository interface {
	SyncForShipment(
		ctx context.Context,
		tx bun.IDB,
		entity *shipment.Shipment,
	) error
}
