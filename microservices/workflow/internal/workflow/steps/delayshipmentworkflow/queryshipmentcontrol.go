package delayshipmentworkflow

import (
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

func QueryShipmentControls(db *bun.DB) func(worker.HatchetContext, *types.QueryShipmentControlsInput) (*types.QueryShipmentControlsOutput, error) {
	return func(hc worker.HatchetContext, _ *types.QueryShipmentControlsInput) (*types.QueryShipmentControlsOutput, error) {
		scResults := make([]types.ShipmentControlResults, 0)
		err := db.NewSelect().
			Table("shipment_controls").
			ColumnExpr("organization_id").
			ColumnExpr("auto_delay_shipments").
			ColumnExpr("auto_delay_shipments_threshold").
			Where("auto_delay_shipments = ?", true).
			Scan(hc, &scResults)
		if err != nil {
			return nil, err
		}

		return &types.QueryShipmentControlsOutput{
			Organizations: scResults,
		}, nil
	}
}
