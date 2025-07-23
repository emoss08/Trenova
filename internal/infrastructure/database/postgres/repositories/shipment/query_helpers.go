package shipment

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// buildDelayedShipmentsCTEs builds the CTEs for finding delayed shipments
func buildDelayedShipmentsCTEs(
	dba bun.IDB,
	currentTime int64,
) (*bun.SelectQuery, *bun.SelectQuery) {
	// CTE to find stops that should be marked as delayed
	stopCte := dba.NewSelect().
		Column("stp.shipment_move_id").
		TableExpr("stops stp").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.status NOT IN (?)", bun.In([]shipment.StopStatus{
					shipment.StopStatusCompleted,
					shipment.StopStatusCanceled,
				})).
				Where("stp.actual_departure IS NULL").
				Where("stp.planned_departure < ?", currentTime)
		})

	// CTE to find moves with delayed stops
	moveCte := dba.NewSelect().
		ColumnExpr("DISTINCT sm.shipment_id").
		TableExpr("shipment_moves sm").
		Where("sm.id IN (SELECT shipment_move_id FROM stop_cte)").
		Where("sm.status NOT IN (?)", bun.In([]shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		}))

	return stopCte, moveCte
}

// buildPreviousRatesCTEs builds the CTEs for finding shipments with matching origin and destination
func buildPreviousRatesCTEs(
	dba bun.IDB,
	originLocationID, destinationLocationID pulid.ID,
) (*bun.SelectQuery, *bun.SelectQuery) {
	// CTE to find shipments with matching origin location
	originCTE := dba.NewSelect().
		Column("first_move.shipment_id").
		TableExpr("shipment_moves first_move").
		Join("JOIN stops origin_stop ON origin_stop.shipment_move_id = first_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("first_move.sequence = 0").
				Where("origin_stop.sequence = 0").
				Where("origin_stop.type IN (?)", bun.In([]shipment.StopType{
					shipment.StopTypePickup,
					shipment.StopTypeSplitPickup,
				})).
				Where("origin_stop.location_id = ?", originLocationID)
		})

	// CTE to find shipments with matching destination location
	destCTE := dba.NewSelect().
		Column("last_move.shipment_id").
		TableExpr("shipment_moves last_move").
		Join("JOIN stops delivery_stop ON delivery_stop.shipment_move_id = last_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("last_move.sequence = (SELECT MAX(sm3.sequence) FROM shipment_moves sm3 WHERE sm3.shipment_id = last_move.shipment_id)").
				Where("delivery_stop.sequence = (SELECT MAX(stp3.sequence) FROM stops stp3 WHERE stp3.shipment_move_id = last_move.id)").
				Where("delivery_stop.location_id = ?", destinationLocationID).
				Where("delivery_stop.type IN (?)", bun.In([]shipment.StopType{
					shipment.StopTypeDelivery,
					shipment.StopTypeSplitDelivery,
				}))
		})

	return originCTE, destCTE
}

// buildDateRangeFilter builds a filter for shipments within a date range
func buildDateRangeFilter(
	sq *bun.SelectQuery,
	orgID pulid.ID,
	startDate, endDate int64,
) *bun.SelectQuery {
	return sq.
		Where("sp.organization_id = ?", orgID).
		Where("sp.created_at >= ?", startDate).
		Where("sp.created_at <= ?", endDate)
}

// buildBulkUpdateQuery builds a query for bulk updating shipment status
func buildBulkUpdateQuery(
	tx bun.Tx,
	shipmentIDs []pulid.ID,
	status shipment.Status,
) *bun.UpdateQuery {
	return tx.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("status = ?", status).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("sp.id IN (?)", bun.In(shipmentIDs))
}
