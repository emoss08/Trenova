package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// defaultBoardLimit caps a single board render. A dispatcher cannot act on more than a
// few hundred moves at once, and an unbounded scan is how this endpoint would become the
// slowest query in the system.
const defaultBoardLimit = 500

const maxBoardLimit = 2000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.DispatchConsoleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-console-repository"),
	}
}

func boardLimit(requested int) int {
	switch {
	case requested <= 0:
		return defaultBoardLimit
	case requested > maxBoardLimit:
		return maxBoardLimit
	default:
		return requested
	}
}

// ListBoardMoves returns the coverage board in a single query. Origin and destination
// are pulled through lateral joins against the move's first and last stop rather than by
// loading every stop, so the cost stays proportional to the number of moves rather than
// to the number of stops beneath them.
func (r *repository) ListBoardMoves(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardMove, error) {
	moveCols := buncolgen.ShipmentMoveColumns
	shipCols := buncolgen.ShipmentColumns
	asnCols := buncolgen.AssignmentColumns

	entities := make([]*repositories.BoardMove, 0, boardLimit(filter.Limit))

	q := r.db.DB().NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(moveCols.ID.As("move_id")).
		ColumnExpr(moveCols.ShipmentID.As("shipment_id")).
		ColumnExpr(moveCols.Status.As("move_status")).
		ColumnExpr(moveCols.Sequence.As("sequence")).
		ColumnExpr(moveCols.Loaded.As("loaded")).
		ColumnExpr(moveCols.Distance.As("distance")).
		ColumnExpr(shipCols.ProNumber.As("pro_number")).
		ColumnExpr(shipCols.BOL.As("bol")).
		ColumnExpr(shipCols.Status.As("shipment_status")).
		ColumnExpr(shipCols.CustomerID.As("customer_id")).
		ColumnExpr(shipCols.ServiceTypeID.As("service_type_id")).
		ColumnExpr(shipCols.TractorTypeID.As("tractor_type_id")).
		ColumnExpr(shipCols.TrailerTypeID.As("trailer_type_id")).
		ColumnExpr(shipCols.TemperatureMin.As("temperature_min")).
		ColumnExpr(shipCols.TemperatureMax.As("temperature_max")).
		ColumnExpr(shipCols.TotalChargeAmount.Expr("{}::float8 AS revenue")).
		ColumnExpr(buncolgen.CustomerColumns.Name.As("customer_name")).
		ColumnExpr(buncolgen.ServiceTypeColumns.Code.As("service_type_code")).
		ColumnExpr(asnCols.ID.As("assignment_id")).
		ColumnExpr(asnCols.PrimaryWorkerID.As("assigned_worker_id")).
		ColumnExpr(asnCols.TractorID.As("assigned_tractor_id")).
		ColumnExpr(asnCols.TrailerID.As("assigned_trailer_id")).
		ColumnExpr(asnCols.AckStatus.As("assignment_ack_status")).
		ColumnExpr(
			"CASE WHEN asn_wrk.id IS NULL THEN '' " +
				"ELSE CONCAT(asn_wrk.first_name, ' ', asn_wrk.last_name) END AS assigned_worker_name",
		).
		ColumnExpr("COALESCE(asn_trac.code, '') AS assigned_tractor_code").
		ColumnExpr("COALESCE(asn_tr.code, '') AS assigned_trailer_code").
		ColumnExpr("COALESCE(mc.move_count, 0) AS move_count").
		ColumnExpr("COALESCE(hz.has_hazmat, FALSE) AS has_hazmat").
		ColumnExpr("COALESCE(hd.has_active_hold, FALSE) AS has_active_hold").
		ColumnExpr("COALESCE(prev.trailer_id, '') AS previous_trailer_id").
		ColumnExpr("COALESCE(orig.stop_id, '') AS origin_stop_id").
		ColumnExpr("COALESCE(orig.location_id, '') AS origin_location_id").
		ColumnExpr("COALESCE(orig.name, '') AS origin_name").
		ColumnExpr("COALESCE(orig.city, '') AS origin_city").
		ColumnExpr("COALESCE(orig.state_abbr, '') AS origin_state").
		ColumnExpr("orig.latitude AS origin_latitude").
		ColumnExpr("orig.longitude AS origin_longitude").
		ColumnExpr("COALESCE(orig.window_start, 0) AS origin_window_start").
		ColumnExpr("orig.window_end AS origin_window_end").
		ColumnExpr("orig.actual_arrival AS origin_actual_arrive").
		ColumnExpr("COALESCE(dest.stop_id, '') AS destination_stop_id").
		ColumnExpr("COALESCE(dest.location_id, '') AS destination_location_id").
		ColumnExpr("COALESCE(dest.name, '') AS destination_name").
		ColumnExpr("COALESCE(dest.city, '') AS destination_city").
		ColumnExpr("COALESCE(dest.state_abbr, '') AS destination_state").
		ColumnExpr("dest.latitude AS destination_latitude").
		ColumnExpr("dest.longitude AS destination_longitude").
		ColumnExpr("COALESCE(dest.window_start, 0) AS destination_window_start").
		ColumnExpr("dest.window_end AS destination_window_end").
		Join("JOIN shipments AS sp ON " + shipCols.ID.EqColumn(moveCols.ShipmentID) +
			" AND " + shipCols.OrganizationID.EqColumn(moveCols.OrganizationID) +
			" AND " + shipCols.BusinessUnitID.EqColumn(moveCols.BusinessUnitID)).
		Join("LEFT JOIN customers AS cus ON " +
			buncolgen.CustomerColumns.ID.EqColumn(shipCols.CustomerID)).
		Join("LEFT JOIN service_types AS st ON " +
			buncolgen.ServiceTypeColumns.ID.EqColumn(shipCols.ServiceTypeID)).
		Join("LEFT JOIN assignments AS a ON " + asnCols.ShipmentMoveID.EqColumn(moveCols.ID) +
			" AND " + asnCols.ArchivedAt.IsNull()).
		Join("LEFT JOIN workers AS asn_wrk ON asn_wrk.id = " + asnCols.PrimaryWorkerID.Qualified()).
		Join("LEFT JOIN tractors AS asn_trac ON asn_trac.id = " + asnCols.TractorID.Qualified()).
		Join("LEFT JOIN trailers AS asn_tr ON asn_tr.id = " + asnCols.TrailerID.Qualified())

	q = joinMoveStopEdges(q)
	q = joinMoveAggregates(q)

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.ShipmentMoveScopeTenant(sq, filter.TenantInfo)
		return applyBoardFilters(sq, filter)
	}).
		Order("COALESCE(orig.window_start, 0) ASC").
		Order(moveCols.Sequence.OrderAsc()).
		Limit(boardLimit(filter.Limit))

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list dispatch board moves: %w", err)
	}

	return entities, nil
}

// joinMoveStopEdges attaches the first pickup and last delivery of each move. LATERAL
// keeps this to one indexed lookup per move instead of loading and grouping every stop.
func joinMoveStopEdges(q *bun.SelectQuery) *bun.SelectQuery {
	const originJoin = `LEFT JOIN LATERAL (
		SELECT stp.id AS stop_id,
			stp.location_id,
			stp.scheduled_window_start AS window_start,
			stp.scheduled_window_end AS window_end,
			stp.actual_arrival,
			loc.name,
			loc.city,
			loc.latitude,
			loc.longitude,
			ust.abbreviation AS state_abbr
		FROM stops AS stp
		LEFT JOIN locations AS loc ON loc.id = stp.location_id
		LEFT JOIN us_states AS ust ON ust.id = loc.state_id
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> 'Canceled'
		ORDER BY stp.sequence ASC
		LIMIT 1
	) AS orig ON TRUE`

	const destinationJoin = `LEFT JOIN LATERAL (
		SELECT stp.id AS stop_id,
			stp.location_id,
			stp.scheduled_window_start AS window_start,
			stp.scheduled_window_end AS window_end,
			loc.name,
			loc.city,
			loc.latitude,
			loc.longitude,
			ust.abbreviation AS state_abbr
		FROM stops AS stp
		LEFT JOIN locations AS loc ON loc.id = stp.location_id
		LEFT JOIN us_states AS ust ON ust.id = loc.state_id
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> 'Canceled'
		ORDER BY stp.sequence DESC
		LIMIT 1
	) AS dest ON TRUE`

	return q.Join(originJoin).Join(destinationJoin)
}

// joinMoveAggregates attaches the per-shipment facts a dispatcher needs on the card:
// how many legs the shipment has, whether it carries hazmat, whether a hold blocks
// dispatch, and which trailer the preceding leg used so trailer continuity can be
// pre-filled on assignment.
func joinMoveAggregates(q *bun.SelectQuery) *bun.SelectQuery {
	const moveCountJoin = `LEFT JOIN LATERAL (
		SELECT COUNT(*)::int AS move_count
		FROM shipment_moves AS sib
		WHERE sib.shipment_id = sm.shipment_id
			AND sib.organization_id = sm.organization_id
			AND sib.business_unit_id = sm.business_unit_id
	) AS mc ON TRUE`

	const hazmatJoin = `LEFT JOIN LATERAL (
		SELECT TRUE AS has_hazmat
		FROM shipment_commodities AS sc
		JOIN commodities AS com ON com.id = sc.commodity_id
		WHERE sc.shipment_id = sm.shipment_id
			AND sc.organization_id = sm.organization_id
			AND sc.business_unit_id = sm.business_unit_id
			AND com.hazardous_material_id IS NOT NULL
		LIMIT 1
	) AS hz ON TRUE`

	const holdJoin = `LEFT JOIN LATERAL (
		SELECT TRUE AS has_active_hold
		FROM shipment_holds AS shh
		WHERE shh.shipment_id = sm.shipment_id
			AND shh.organization_id = sm.organization_id
			AND shh.business_unit_id = sm.business_unit_id
			AND shh.blocks_dispatch = TRUE
			AND shh.released_at IS NULL
		LIMIT 1
	) AS hd ON TRUE`

	const previousTrailerJoin = `LEFT JOIN LATERAL (
		SELECT pa.trailer_id
		FROM shipment_moves AS pm
		JOIN assignments AS pa ON pa.shipment_move_id = pm.id AND pa.archived_at IS NULL
		WHERE pm.shipment_id = sm.shipment_id
			AND pm.organization_id = sm.organization_id
			AND pm.business_unit_id = sm.business_unit_id
			AND pm.sequence < sm.sequence
			AND pa.trailer_id IS NOT NULL
		ORDER BY pm.sequence DESC
		LIMIT 1
	) AS prev ON TRUE`

	return q.Join(moveCountJoin).Join(hazmatJoin).Join(holdJoin).Join(previousTrailerJoin)
}

func applyBoardFilters(
	sq *bun.SelectQuery,
	filter *repositories.DispatchBoardFilter,
) *bun.SelectQuery {
	moveCols := buncolgen.ShipmentMoveColumns
	shipCols := buncolgen.ShipmentColumns
	asnCols := buncolgen.AssignmentColumns

	if len(filter.MoveStatuses) > 0 {
		sq = sq.Where(moveCols.Status.In(), bun.List(filter.MoveStatuses))
	} else {
		sq = sq.Where(moveCols.Status.In(), bun.List([]shipment.MoveStatus{
			shipment.MoveStatusNew,
			shipment.MoveStatusAssigned,
			shipment.MoveStatusInTransit,
		}))
	}

	if !filter.IncludeCovered {
		sq = sq.Where(asnCols.ID.IsNull())
	}

	if len(filter.MoveIDs) > 0 {
		sq = sq.Where(moveCols.ID.In(), bun.List(filter.MoveIDs))
	}
	if len(filter.CustomerIDs) > 0 {
		sq = sq.Where(shipCols.CustomerID.In(), bun.List(filter.CustomerIDs))
	}
	if len(filter.ServiceTypeIDs) > 0 {
		sq = sq.Where(shipCols.ServiceTypeID.In(), bun.List(filter.ServiceTypeIDs))
	}
	if filter.WindowStart > 0 {
		sq = sq.Where("COALESCE(dest.window_end, dest.window_start, orig.window_start) >= ?",
			filter.WindowStart)
	}
	if filter.WindowEnd > 0 {
		sq = sq.Where("COALESCE(orig.window_start, 0) <= ?", filter.WindowEnd)
	}
	if filter.Query != "" {
		pattern := "%" + filter.Query + "%"
		sq = sq.WhereGroup(" AND ", func(g *bun.SelectQuery) *bun.SelectQuery {
			return g.Where(shipCols.ProNumber.ILike(), pattern).
				WhereOr(shipCols.BOL.ILike(), pattern).
				WhereOr(buncolgen.CustomerColumns.Name.ILike(), pattern).
				WhereOr("orig.city ILIKE ?", pattern).
				WhereOr("dest.city ILIKE ?", pattern)
		})
	}

	return sq
}
