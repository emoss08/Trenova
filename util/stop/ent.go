package stop

import (
	"context"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/shipmentmove"
	"github.com/emoss08/trenova/ent/stop"
	"github.com/google/uuid"
)

func GetShipmentMoveByStop(
	ctx context.Context, client *ent.Client, stopID uuid.UUID,
) (*ent.ShipmentMove, error) {
	shipmentMove, err := client.ShipmentMove.Query().
		Where(
			shipmentmove.HasMoveStopsWith(
				stop.IDEQ(stopID),
			),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	return shipmentMove, nil
}
