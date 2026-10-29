package shipmentlifecycle

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func uncancelRequestRequired() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("request", errortypes.ErrRequired, "Uncancel request is required")

	return multiErr
}

func uncancelRestores(
	original, restored *shipment.Shipment,
) ([]shipmentports.MoveStatusRestore, []shipmentports.StopStatusRestore) {
	moves := make([]shipmentports.MoveStatusRestore, 0, len(restored.Moves))
	stops := make([]shipmentports.StopStatusRestore, 0)

	for moveIndex, move := range restored.Moves {
		if move == nil || moveIndex >= len(original.Moves) || original.Moves[moveIndex] == nil {
			continue
		}

		before := original.Moves[moveIndex]
		if before.IsCanceled() && !move.IsCanceled() {
			moves = append(moves, shipmentports.MoveStatusRestore{
				MoveID: move.ID,
				Status: move.Status,
			})
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil || stopIndex >= len(before.Stops) || before.Stops[stopIndex] == nil {
				continue
			}

			if before.Stops[stopIndex].IsCanceled() && !stop.IsCanceled() {
				stops = append(stops, shipmentports.StopStatusRestore{
					StopID: stop.ID,
					Status: stop.Status,
				})
			}
		}
	}

	return moves, stops
}
