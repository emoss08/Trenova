package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/uptrace/bun"
)

type ShipmentAdditionalChargeRepository interface {
	SyncForShipment(
		ctx context.Context,
		tx bun.IDB,
		entity *shipment.Shipment,
	) error
}
