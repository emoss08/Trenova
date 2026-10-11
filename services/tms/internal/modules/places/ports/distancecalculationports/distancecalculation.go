package distancecalculationports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
)

type DistanceCalculationRepository interface {
	CreateRun(ctx context.Context, entity *distancecalculation.Run) error
	UpdateMoveDistance(ctx context.Context, move *shipment.ShipmentMove) error
}
