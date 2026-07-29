package dispatchconsolerepository

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

const (
	originAlias      = "orig"
	destinationAlias = "dest"
	orderAscending   = "ASC"
	orderDescending  = "DESC"
)

// openBoardStatuses are the moves that still need a dispatcher's attention. They are the
// board's default status filter, and they are also exempt from the window's lower bound:
// an overdue move is the most urgent thing on the board, not history to be hidden.
var openBoardStatuses = []shipment.MoveStatus{
	shipment.MoveStatusNew,
	shipment.MoveStatusAssigned,
	shipment.MoveStatusInTransit,
}

var openMoveStatuses = []shipment.MoveStatus{
	shipment.MoveStatusAssigned,
	shipment.MoveStatusInTransit,
}

var (
	shipmentJoin = "JOIN " + buncolgen.ShipmentTable.As(buncolgen.ShipmentTable.Alias) + " ON " +
		buncolgen.ShipmentColumns.ID.EqColumn(buncolgen.ShipmentMoveColumns.ShipmentID) +
		" AND " + buncolgen.ShipmentColumns.OrganizationID.EqColumn(
		buncolgen.ShipmentMoveColumns.OrganizationID) +
		" AND " + buncolgen.ShipmentColumns.BusinessUnitID.EqColumn(
		buncolgen.ShipmentMoveColumns.BusinessUnitID)

	customerJoin = "LEFT JOIN " + buncolgen.CustomerTable.As(buncolgen.CustomerTable.Alias) +
		" ON " + buncolgen.CustomerColumns.ID.EqColumn(buncolgen.ShipmentColumns.CustomerID) +
		" AND " + buncolgen.CustomerColumns.OrganizationID.EqColumn(
		buncolgen.ShipmentColumns.OrganizationID) +
		" AND " + buncolgen.CustomerColumns.BusinessUnitID.EqColumn(
		buncolgen.ShipmentColumns.BusinessUnitID)

	serviceTypeJoin = "LEFT JOIN " + buncolgen.ServiceTypeTable.As(
		buncolgen.ServiceTypeTable.Alias) +
		" ON " + buncolgen.ServiceTypeColumns.ID.EqColumn(
		buncolgen.ShipmentColumns.ServiceTypeID) +
		" AND " + buncolgen.ServiceTypeColumns.OrganizationID.EqColumn(
		buncolgen.ShipmentColumns.OrganizationID) +
		" AND " + buncolgen.ServiceTypeColumns.BusinessUnitID.EqColumn(
		buncolgen.ShipmentColumns.BusinessUnitID)

	assignmentJoin = "LEFT JOIN " + buncolgen.AssignmentTable.As(
		buncolgen.AssignmentTable.Alias) +
		" ON " + buncolgen.AssignmentColumns.ShipmentMoveID.EqColumn(
		buncolgen.ShipmentMoveColumns.ID) +
		" AND " + buncolgen.AssignmentColumns.OrganizationID.EqColumn(
		buncolgen.ShipmentMoveColumns.OrganizationID) +
		" AND " + buncolgen.AssignmentColumns.BusinessUnitID.EqColumn(
		buncolgen.ShipmentMoveColumns.BusinessUnitID) +
		" AND " + buncolgen.AssignmentColumns.ArchivedAt.IsNull()
)

const openAssignmentLateral = `LEFT JOIN LATERAL (
	SELECT COUNT(*)::int AS open_assignments
	FROM assignments AS a
	JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id
		AND sm.organization_id = a.organization_id
		AND sm.business_unit_id = a.business_unit_id
	WHERE a.primary_worker_id = wrk.id
		AND a.organization_id = wrk.organization_id
		AND a.business_unit_id = wrk.business_unit_id
		AND a.archived_at IS NULL
		AND sm.status IN (?)
) AS oa ON TRUE`

func stopEdgeLateral(alias, direction string) string {
	return `LEFT JOIN LATERAL (
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
			AND loc.organization_id = stp.organization_id
			AND loc.business_unit_id = stp.business_unit_id
		LEFT JOIN us_states AS ust ON ust.id = loc.state_id
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> ?
		ORDER BY stp.sequence ` + direction + `
		LIMIT 1
	) AS ` + alias + ` ON TRUE`
}

func searchPattern(term string) string {
	return "%" + stringutils.EscapeLikePattern(term) + "%"
}

func applyMoveFilters(
	sq *bun.SelectQuery,
	filter *repositories.DispatchBoardFilter,
) *bun.SelectQuery {
	moveCols := buncolgen.ShipmentMoveColumns
	shipCols := buncolgen.ShipmentColumns

	if len(filter.MoveStatuses) > 0 {
		sq = sq.Where(moveCols.Status.In(), bun.List(filter.MoveStatuses))
	} else {
		sq = sq.Where(moveCols.Status.In(), bun.List(openBoardStatuses))
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
	// The lower bound trims work the board has already finished with. Open moves are
	// exempt: a load whose delivery appointment lapsed two days ago is still uncovered,
	// still the dispatcher's problem, and would otherwise vanish from the board entirely.
	if filter.WindowStart > 0 {
		sq = sq.WhereGroup(" AND ", func(g *bun.SelectQuery) *bun.SelectQuery {
			return g.Where(
				"COALESCE(dest.window_end, dest.window_start, orig.window_start) >= ?",
				filter.WindowStart,
			).WhereOr(moveCols.Status.In(), bun.List(openBoardStatuses))
		})
	}
	if filter.WindowEnd > 0 {
		sq = sq.Where("COALESCE(orig.window_start, 0) <= ?", filter.WindowEnd)
	}
	if filter.Query != "" {
		pattern := searchPattern(filter.Query)
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

func applyDriverFilters(
	sq *bun.SelectQuery,
	filter *repositories.DispatchBoardFilter,
) *bun.SelectQuery {
	cols := buncolgen.WorkerColumns

	sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive).
		Where(cols.CanBeAssigned.IsTrue())

	if len(filter.WorkerIDs) > 0 {
		sq = sq.Where(cols.ID.In(), bun.List(filter.WorkerIDs))
	}
	if len(filter.FleetCodeIDs) > 0 {
		sq = sq.Where(cols.FleetCodeID.In(), bun.List(filter.FleetCodeIDs))
	}

	return sq
}
