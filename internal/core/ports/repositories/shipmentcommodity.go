package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type CommodityDeletionRequest struct {
	ExistingCommodityMap map[pulid.ID]*shipment.ShipmentCommodity
	UpdatedCommodityIDs  map[pulid.ID]struct{}
	CommoditiesToDelete  []*shipment.ShipmentCommodity
}

type ShipmentCommodityRepository interface {
	HandleCommodityOperations(
		ctx context.Context,
		tx bun.IDB,
		shp *shipment.Shipment,
		isCreate bool,
	) error
}
