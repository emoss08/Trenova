package delayshipmentworkflow

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

func QueryStops(
	db *bun.DB,
) func(worker.HatchetContext, *types.QueryStopsInput) (*types.QueryStopsOutput, error) {
	return func(hc worker.HatchetContext, _ *types.QueryStopsInput) (*types.QueryStopsOutput, error) {
		var querySCResults types.QueryShipmentControlsOutput
		if err := hc.StepOutput("get-shipment-controls", &querySCResults); err != nil {
			return nil, err
		}

		if len(querySCResults.Organizations) == 0 {
			hc.Log("no organizations have auto-delay enabled, skipping")
			return &types.QueryStopsOutput{}, nil
		}

		now := time.Now().Unix()
		orgIDs, orgThresholdMap := buildOrgThresholdMap(querySCResults.Organizations, now)

		query := db.NewSelect().
			Table("stops").
			ColumnExpr("id AS stop_id").
			ColumnExpr("organization_id").
			ColumnExpr("shipment_move_id").
			ColumnExpr("planned_arrival").
			ColumnExpr("planned_departure").
			Where("organization_id IN (?)", bun.In(orgIDs)).
			Where("actual_arrival IS NULL").
			Where("status IN ('New', 'InTransit')")

		if len(orgIDs) > 0 {
			query = buildThresholdConditions(query, orgIDs, orgThresholdMap)
		}

		pastDueStops := make([]types.StopResults, 0)
		if err := query.Scan(hc, &pastDueStops); err != nil {
			hc.Log(fmt.Sprintf("error querying stops: %v", err))
			return nil, err
		}

		hc.Log(
			fmt.Sprintf(
				"found %d past due stops across all configured organizations",
				len(pastDueStops),
			),
		)

		return &types.QueryStopsOutput{
			PastDueStops: pastDueStops,
		}, nil
	}
}

func buildOrgThresholdMap(
	orgs []types.ShipmentControlResults,
	now int64,
) ([]string, map[string]int64) {
	orgIDs := make([]string, 0, len(orgs))
	orgThresholdMap := make(map[string]int64)

	for _, org := range orgs {
		if org.AutoDelayShipments {
			orgIDs = append(orgIDs, org.OrganizationID)
			thresholdInSeconds := int64(org.AutoDelayShipmentsThreshold) * 60
			orgThresholdMap[org.OrganizationID] = now - thresholdInSeconds
		}
	}
	return orgIDs, orgThresholdMap
}

func buildThresholdConditions(
	q *bun.SelectQuery,
	orgIDs []string,
	orgThresholdMap map[string]int64,
) *bun.SelectQuery {
	return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		sq := q
		for i, orgID := range orgIDs {
			thresholdTime := orgThresholdMap[orgID]
			condition := `(
				organization_id = ? AND 
				(
					(planned_departure IS NOT NULL AND planned_departure < ?) OR
					(planned_arrival < ?)
				)
			)`

			if i == 0 {
				sq = sq.Where(condition, orgID, thresholdTime, thresholdTime)
			} else {
				sq = sq.WhereOr(condition, orgID, thresholdTime, thresholdTime)
			}
		}
		return sq
	})
}
