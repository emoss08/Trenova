package delayshipmentworkflow

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

func DelayShipments(db *bun.DB) func(worker.HatchetContext, *types.DelayShipmentsInput) (*types.DelayShipmentsOutput, error) {
	return func(hc worker.HatchetContext, _ *types.DelayShipmentsInput) (*types.DelayShipmentsOutput, error) {
		var queryStopsResults types.QueryStopsOutput
		if err := hc.StepOutput("query-stops", &queryStopsResults); err != nil {
			return nil, err
		}

		if len(queryStopsResults.PastDueStops) == 0 {
			hc.Log("no past due stops found, no shipments to delay")
			return &types.DelayShipmentsOutput{DelayedShipments: 0}, nil
		}

		// * Get unique shipment IDs from the moves
		type ShipmentMove struct {
			ShipmentID     string `bun:"shipment_id"`
			OrganizationID string `bun:"organization_id"`
		}

		// * Collect moveIDs from past due stops
		moveIDs := make([]string, 0, len(queryStopsResults.PastDueStops))
		for _, stop := range queryStopsResults.PastDueStops {
			moveIDs = append(moveIDs, stop.ShipmentMoveID)
		}

		hc.Log(fmt.Sprintf("querying for %d shipment moves", len(moveIDs)))

		// * Query to get the shipment IDs from the moves
		shipmentMoves := make([]ShipmentMove, 0)
		err := db.NewSelect().
			Table("shipment_moves").
			Column("shipment_id", "organization_id").
			Where("id IN (?)", bun.In(moveIDs)).
			Scan(hc, &shipmentMoves)
		if err != nil {
			hc.Log(fmt.Sprintf("error querying shipment moves: %v", err))
			return nil, err
		}

		hc.Log(fmt.Sprintf("found %d shipment moves", len(shipmentMoves)))

		if len(shipmentMoves) == 0 {
			hc.Log("no shipment moves found, no shipments to delay")
			return &types.DelayShipmentsOutput{DelayedShipments: 0}, nil
		}

		// Create a WITH clause for the shipment IDs and organization IDs
		// to use in the bulk update
		subquery := db.NewValues(&shipmentMoves)

		// Perform a bulk update for all shipments at once
		res, err := db.NewUpdate().
			With("_data", subquery).
			Table("shipments").
			Set("status = ?", "Delayed").
			Set("updated_at = ?", time.Now().Unix()).
			Where("id IN (SELECT shipment_id FROM _data)").
			Where("organization_id IN (SELECT organization_id FROM _data)").
			// Only update if currently in a status that can be changed to delayed
			Where("status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit')").
			Exec(hc)
		if err != nil {
			hc.Log(fmt.Sprintf("error updating shipments: %v", err))
			return nil, err
		}

		rowsAffected, _ := res.RowsAffected()
		hc.Log(fmt.Sprintf("updated %d shipments to 'Delayed' status", rowsAffected))

		return &types.DelayShipmentsOutput{
			DelayedShipments: int(rowsAffected),
		}, nil
	}
}
