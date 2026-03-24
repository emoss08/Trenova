package shipmentrepository

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func buildPreviousRatesCTEs(
	dba bun.IDB,
	originLocationID pulid.ID,
	destinationLocationID pulid.ID,
) (originCTE, destinationCTE *bun.SelectQuery) {
	originCTE = dba.NewSelect().
		Column("first_move.shipment_id").
		TableExpr("shipment_moves AS first_move").
		Join("JOIN stops AS origin_stop ON origin_stop.shipment_move_id = first_move.id").
		Where("first_move.sequence = 0").
		Where("origin_stop.sequence = 0").
		Where("origin_stop.type IN (?)", bun.List([]shipment.StopType{
			shipment.StopTypePickup,
			shipment.StopTypeSplitPickup,
		})).
		Where("origin_stop.location_id = ?", originLocationID)

	destinationCTE = dba.NewSelect().
		Column("last_move.shipment_id").
		TableExpr("shipment_moves AS last_move").
		Join("JOIN stops AS delivery_stop ON delivery_stop.shipment_move_id = last_move.id").
		Where("last_move.sequence = (SELECT MAX(sm3.sequence) FROM shipment_moves AS sm3 WHERE sm3.shipment_id = last_move.shipment_id)").
		Where("delivery_stop.sequence = (SELECT MAX(stp3.sequence) FROM stops AS stp3 WHERE stp3.shipment_move_id = last_move.id)").
		Where("delivery_stop.type IN (?)", bun.List([]shipment.StopType{
			shipment.StopTypeDelivery,
			shipment.StopTypeSplitDelivery,
		})).
		Where("delivery_stop.location_id = ?", destinationLocationID)

	return originCTE, destinationCTE
}
