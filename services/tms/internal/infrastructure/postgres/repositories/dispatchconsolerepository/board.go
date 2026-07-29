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

const defaultBoardLimit = 500

const maxBoardLimit = 2000

const (
	assignedWorkerAlias  = "asn_wrk"
	assignedTractorAlias = "asn_trac"
	assignedTrailerAlias = "asn_tr"
)

var (
	asnWorkerID        = buncolgen.WorkerColumns.ID.WithAlias(assignedWorkerAlias)
	asnWorkerFirstName = buncolgen.WorkerColumns.FirstName.WithAlias(assignedWorkerAlias)
	asnWorkerLastName  = buncolgen.WorkerColumns.LastName.WithAlias(assignedWorkerAlias)
	asnWorkerOrgID     = buncolgen.WorkerColumns.OrganizationID.WithAlias(assignedWorkerAlias)
	asnWorkerBuID      = buncolgen.WorkerColumns.BusinessUnitID.WithAlias(assignedWorkerAlias)

	asnTractorID    = buncolgen.TractorColumns.ID.WithAlias(assignedTractorAlias)
	asnTractorCode  = buncolgen.TractorColumns.Code.WithAlias(assignedTractorAlias)
	asnTractorOrgID = buncolgen.TractorColumns.OrganizationID.WithAlias(assignedTractorAlias)
	asnTractorBuID  = buncolgen.TractorColumns.BusinessUnitID.WithAlias(assignedTractorAlias)

	asnTrailerID    = buncolgen.TrailerColumns.ID.WithAlias(assignedTrailerAlias)
	asnTrailerCode  = buncolgen.TrailerColumns.Code.WithAlias(assignedTrailerAlias)
	asnTrailerOrgID = buncolgen.TrailerColumns.OrganizationID.WithAlias(assignedTrailerAlias)
	asnTrailerBuID  = buncolgen.TrailerColumns.BusinessUnitID.WithAlias(assignedTrailerAlias)

	assignedWorkerJoin = "LEFT JOIN " + buncolgen.WorkerTable.As(assignedWorkerAlias) +
		" ON " + asnWorkerID.EqColumn(buncolgen.AssignmentColumns.PrimaryWorkerID) +
		" AND " + asnWorkerOrgID.EqColumn(buncolgen.AssignmentColumns.OrganizationID) +
		" AND " + asnWorkerBuID.EqColumn(buncolgen.AssignmentColumns.BusinessUnitID)

	assignedTractorJoin = "LEFT JOIN " + buncolgen.TractorTable.As(assignedTractorAlias) +
		" ON " + asnTractorID.EqColumn(buncolgen.AssignmentColumns.TractorID) +
		" AND " + asnTractorOrgID.EqColumn(buncolgen.AssignmentColumns.OrganizationID) +
		" AND " + asnTractorBuID.EqColumn(buncolgen.AssignmentColumns.BusinessUnitID)

	assignedTrailerJoin = "LEFT JOIN " + buncolgen.TrailerTable.As(assignedTrailerAlias) +
		" ON " + asnTrailerID.EqColumn(buncolgen.AssignmentColumns.TrailerID) +
		" AND " + asnTrailerOrgID.EqColumn(buncolgen.AssignmentColumns.OrganizationID) +
		" AND " + asnTrailerBuID.EqColumn(buncolgen.AssignmentColumns.BusinessUnitID)
)

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
		ColumnExpr(buncolgen.Expr(
			"CASE WHEN {0} IS NULL THEN '' ELSE CONCAT({1}, ' ', {2}) END AS assigned_worker_name",
			asnWorkerID, asnWorkerFirstName, asnWorkerLastName,
		)).
		ColumnExpr(asnTractorCode.Expr("COALESCE({}, '') AS assigned_tractor_code")).
		ColumnExpr(asnTrailerCode.Expr("COALESCE({}, '') AS assigned_trailer_code")).
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
		Join(shipmentJoin).
		Join(customerJoin).
		Join(serviceTypeJoin).
		Join(assignmentJoin).
		Join(assignedWorkerJoin).
		Join(assignedTractorJoin).
		Join(assignedTrailerJoin)

	q = joinMoveStopEdges(q)
	q = joinMoveAggregates(q)

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.ShipmentMoveScopeTenant(sq, filter.TenantInfo)
		sq = applyMoveFilters(sq, filter)
		if !filter.IncludeCovered {
			sq = sq.Where(asnCols.ID.IsNull())
		}
		return sq
	}).
		OrderExpr("COALESCE(orig.window_start, 0) ASC").
		Order(moveCols.Sequence.OrderAsc()).
		Order(moveCols.ID.OrderAsc()).
		Limit(boardLimit(filter.Limit))

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list dispatch board moves: %w", err)
	}

	return entities, nil
}

func joinMoveStopEdges(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Join(stopEdgeLateral(originAlias, orderAscending), shipment.StopStatusCanceled).
		Join(stopEdgeLateral(destinationAlias, orderDescending), shipment.StopStatusCanceled)
}

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
			AND com.organization_id = sc.organization_id
			AND com.business_unit_id = sc.business_unit_id
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
		JOIN assignments AS pa ON pa.shipment_move_id = pm.id
			AND pa.organization_id = pm.organization_id
			AND pa.business_unit_id = pm.business_unit_id
			AND pa.archived_at IS NULL
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
