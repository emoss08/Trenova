/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
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
