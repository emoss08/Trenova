package shipmentrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/shared/pulid"
)

func previousRateOrigins(originLocationID pulid.ID) *keel.Query[*shipment.ShipmentMove] {
	firstMove := shipmentdb.ShipmentMoves.As("first_move")
	originStop := shipmentdb.Stops.As("origin_stop")

	return firstMove.Pick(firstMove.ShipmentID).
		Join(originStop, originStop.ShipmentMoveID.EqCol(firstMove.ID)).
		Where(
			firstMove.Sequence.Eq(0),
			originStop.Sequence.Eq(0),
			originStop.Type.In(shipment.StopTypePickup, shipment.StopTypeSplitPickup),
			originStop.LocationID.Eq(originLocationID),
		)
}

func previousRateDestinations(destinationLocationID pulid.ID) *keel.Query[*shipment.ShipmentMove] {
	lastMove := shipmentdb.ShipmentMoves.As("last_move")
	deliveryStop := shipmentdb.Stops.As("delivery_stop")
	otherMove := shipmentdb.ShipmentMoves.As("sm3")
	otherStop := shipmentdb.Stops.As("stp3")

	return lastMove.Pick(lastMove.ShipmentID).
		Join(deliveryStop, deliveryStop.ShipmentMoveID.EqCol(lastMove.ID)).
		Where(
			lastMove.Sequence.EqCol(keel.Scalar(
				otherMove.Pick(otherMove.Sequence.Max()).
					Where(otherMove.ShipmentID.EqCol(lastMove.ShipmentID)),
			)),
			deliveryStop.Sequence.EqCol(keel.Scalar(
				otherStop.Pick(otherStop.Sequence.Max()).
					Where(otherStop.ShipmentMoveID.EqCol(lastMove.ID)),
			)),
			deliveryStop.Type.In(shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery),
			deliveryStop.LocationID.Eq(destinationLocationID),
		)
}
