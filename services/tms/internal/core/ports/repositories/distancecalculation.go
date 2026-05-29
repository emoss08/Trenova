package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type DistanceCalculationRepository interface {
	CreateRun(ctx context.Context, entity *distancecalculation.Run) error
	UpdateMoveDistance(ctx context.Context, move *shipment.ShipmentMove) error
}
