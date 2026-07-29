package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var (
	moveFromAssignmentJoin = "JOIN " + buncolgen.ShipmentMoveTable.As(
		buncolgen.ShipmentMoveTable.Alias) +
		" ON " + buncolgen.ShipmentMoveColumns.ID.EqColumn(
		buncolgen.AssignmentColumns.ShipmentMoveID) +
		" AND " + buncolgen.ShipmentMoveColumns.OrganizationID.EqColumn(
		buncolgen.AssignmentColumns.OrganizationID) +
		" AND " + buncolgen.ShipmentMoveColumns.BusinessUnitID.EqColumn(
		buncolgen.AssignmentColumns.BusinessUnitID)

	fleetCodeJoin = "LEFT JOIN " + buncolgen.FleetCodeTable.As(buncolgen.FleetCodeTable.Alias) +
		" ON " + buncolgen.FleetCodeColumns.ID.EqColumn(buncolgen.WorkerColumns.FleetCodeID) +
		" AND " + buncolgen.FleetCodeColumns.OrganizationID.EqColumn(
		buncolgen.WorkerColumns.OrganizationID) +
		" AND " + buncolgen.FleetCodeColumns.BusinessUnitID.EqColumn(
		buncolgen.WorkerColumns.BusinessUnitID)

	workerStateJoin = "LEFT JOIN " + buncolgen.UsStateTable.As(buncolgen.UsStateTable.Alias) +
		" ON " + buncolgen.UsStateColumns.ID.EqColumn(buncolgen.WorkerColumns.StateID)
)

func (r *repository) ListBoardDrivers(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardDriver, error) {
	cols := buncolgen.WorkerColumns
	entities := make([]*repositories.BoardDriver, 0, boardLimit(filter.Limit))

	const tractorLateral = `LEFT JOIN LATERAL (
		SELECT trac.id,
			trac.code,
			trac.equipment_type_id,
			trac.fleet_code_id,
			trac.status
		FROM tractors AS trac
		WHERE trac.primary_worker_id = wrk.id
			AND trac.organization_id = wrk.organization_id
			AND trac.business_unit_id = wrk.business_unit_id
		ORDER BY (trac.status = ?) DESC, trac.code ASC
		LIMIT 1
	) AS wtrac ON TRUE`

	q := r.db.DB().NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr(cols.ID.As("worker_id")).
		ColumnExpr(cols.FirstName.As("first_name")).
		ColumnExpr(cols.LastName.As("last_name")).
		ColumnExpr(cols.Type.As("worker_type")).
		ColumnExpr(cols.DriverType.As("driver_type")).
		ColumnExpr(cols.FleetCodeID.As("fleet_code_id")).
		ColumnExpr(cols.City.As("city")).
		ColumnExpr(cols.PostalCode.As("postal_code")).
		ColumnExpr(cols.ProfilePicURL.As("profile_pic_url")).
		ColumnExpr(cols.AssignmentBlocked.Expr("COALESCE({}, '') AS assignment_note")).
		ColumnExpr(cols.AvailableForDispatch.As("available_for_dispatch")).
		ColumnExpr(buncolgen.FleetCodeColumns.Code.Expr("COALESCE({}, '') AS fleet_code_name")).
		ColumnExpr(buncolgen.UsStateColumns.Abbreviation.Expr("COALESCE({}, '') AS state_abbr")).
		ColumnExpr("COALESCE(wtrac.id, '') AS tractor_id").
		ColumnExpr("COALESCE(wtrac.code, '') AS tractor_code").
		ColumnExpr("COALESCE(wtrac.equipment_type_id, '') AS tractor_type_id").
		ColumnExpr("COALESCE(wtrac.fleet_code_id, '') AS tractor_fleet_id").
		ColumnExpr("COALESCE(wtrac.status = ?, FALSE) AS tractor_status_ok",
			domaintypes.EquipmentStatusAvailable).
		ColumnExpr("COALESCE(oa.open_assignments, 0) AS open_assignments").
		Join(fleetCodeJoin).
		Join(workerStateJoin).
		Join(tractorLateral, domaintypes.EquipmentStatusAvailable).
		Join(openAssignmentLateral, bun.List(openMoveStatuses)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerScopeTenant(sq, filter.TenantInfo)
			sq = applyDriverFilters(sq, filter)

			if filter.Query != "" {
				pattern := searchPattern(filter.Query)
				sq = sq.WhereGroup(" AND ", func(g *bun.SelectQuery) *bun.SelectQuery {
					return g.Where(cols.FirstName.ILike(), pattern).
						WhereOr(cols.LastName.ILike(), pattern).
						WhereOr("wtrac.code ILIKE ?", pattern)
				})
			}
			return sq
		}).
		Order(cols.LastName.OrderAsc()).
		Order(cols.FirstName.OrderAsc()).
		Order(cols.ID.OrderAsc()).
		Limit(boardLimit(filter.Limit))

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list dispatch board drivers: %w", err)
	}

	return entities, nil
}

func (r *repository) ListWorkerCommitments(
	ctx context.Context,
	req *repositories.ListWorkerWindowsRequest,
) ([]*repositories.WorkerCommitment, error) {
	if len(req.WorkerIDs) == 0 {
		return []*repositories.WorkerCommitment{}, nil
	}

	asnCols := buncolgen.AssignmentColumns
	moveCols := buncolgen.ShipmentMoveColumns
	entities := make([]*repositories.WorkerCommitment, 0, len(req.WorkerIDs))

	const windowLateral = `JOIN LATERAL (
		SELECT MIN(stp.scheduled_window_start) AS window_start,
			MAX(COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)) AS window_end
		FROM stops AS stp
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> ?
	) AS win ON TRUE`

	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr(asnCols.ShipmentMoveID.As("move_id")).
		ColumnExpr(asnCols.TrailerID.Expr("COALESCE({}, '') AS trailer_id")).
		ColumnExpr(moveCols.ShipmentID.As("shipment_id")).
		ColumnExpr(moveCols.Status.As("move_status")).
		ColumnExpr(buncolgen.ShipmentColumns.ProNumber.Expr("COALESCE({}, '') AS pro_number")).
		ColumnExpr("COALESCE(win.window_start, 0) AS window_start").
		ColumnExpr("COALESCE(win.window_end, 0) AS window_end").
		ColumnExpr("COALESCE(dest.location_id, '') AS destination_id").
		ColumnExpr("COALESCE(dest.city, '') AS destination_city").
		ColumnExpr("COALESCE(dest.state_abbr, '') AS destination_st").
		ColumnExpr("dest.latitude AS dest_latitude").
		ColumnExpr("dest.longitude AS dest_longitude").
		Join(moveFromAssignmentJoin).
		Join("LEFT "+shipmentJoin).
		Join(windowLateral, shipment.StopStatusCanceled).
		Join(stopEdgeLateral(destinationAlias, orderDescending), shipment.StopStatusCanceled).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.ArchivedAt.IsNull()).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(moveCols.Status.In(), bun.List(openMoveStatuses))

			if req.WindowEnd > 0 {
				sq = sq.Where("COALESCE(win.window_start, 0) <= ?", req.WindowEnd)
			}
			if req.WindowStart > 0 {
				sq = sq.Where("COALESCE(win.window_end, win.window_start, 0) >= ?", req.WindowStart)
			}
			return sq
		}).
		Order("win.window_start ASC")

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker commitments: %w", err)
	}

	return entities, nil
}

func (r *repository) ListWorkerTimeOff(
	ctx context.Context,
	req *repositories.ListWorkerWindowsRequest,
) ([]*repositories.WorkerTimeOff, error) {
	if len(req.WorkerIDs) == 0 {
		return []*repositories.WorkerTimeOff{}, nil
	}

	cols := buncolgen.WorkerPTOColumns
	entities := make([]*repositories.WorkerTimeOff, 0, len(req.WorkerIDs))

	q := r.db.DB().NewSelect().
		Model((*worker.WorkerPTO)(nil)).
		ColumnExpr(cols.WorkerID.As("worker_id")).
		ColumnExpr(cols.StartDate.As("start_date")).
		ColumnExpr(cols.EndDate.As("end_date")).
		ColumnExpr(cols.Type.As("pto_type")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerPTOScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), worker.PTOStatusApproved).
				Where(cols.WorkerID.In(), bun.List(req.WorkerIDs))

			if req.WindowEnd > 0 {
				sq = sq.Where(cols.StartDate.Lte(), req.WindowEnd)
			}
			if req.WindowStart > 0 {
				sq = sq.Where(cols.EndDate.Gte(), req.WindowStart)
			}
			return sq
		}).
		Order(cols.StartDate.OrderAsc())

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker time off: %w", err)
	}

	return entities, nil
}

func (r *repository) ListWorkerWorkload(
	ctx context.Context,
	req *repositories.ListWorkloadRequest,
) ([]*repositories.WorkerWorkload, error) {
	if len(req.WorkerIDs) == 0 {
		return []*repositories.WorkerWorkload{}, nil
	}

	asnCols := buncolgen.AssignmentColumns
	moveCols := buncolgen.ShipmentMoveColumns
	entities := make([]*repositories.WorkerWorkload, 0, len(req.WorkerIDs))

	const departureLateral = `LEFT JOIN LATERAL (
		SELECT stp.actual_departure
		FROM stops AS stp
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> ?
		ORDER BY stp.sequence DESC
		LIMIT 1
	) AS dep ON TRUE`

	perShipment := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr("COUNT(*)::int AS move_count").
		ColumnExpr(buncolgen.Expr(
			"COALESCE(SUM({0}), 0)::float8 AS total_miles", moveCols.Distance)).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(MAX({0}), 0)::float8 AS shipment_revenue",
			buncolgen.ShipmentColumns.TotalChargeAmount)).
		ColumnExpr(buncolgen.Expr(
			"MAX(COALESCE(dep.actual_departure, {0})) AS last_ended_at", asnCols.CreatedAt)).
		Join(moveFromAssignmentJoin).
		Join("LEFT "+shipmentJoin).
		Join(departureLateral, shipment.StopStatusCanceled).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(moveCols.Status.NotEq(), shipment.MoveStatusCanceled)
			if req.Since > 0 {
				sq = sq.Where(asnCols.CreatedAt.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(asnCols.PrimaryWorkerID.Qualified() + ", " + moveCols.ShipmentID.Qualified())

	q := r.db.DB().NewSelect().
		TableExpr("(?) AS wl", perShipment).
		ColumnExpr("wl.worker_id AS worker_id").
		ColumnExpr("SUM(wl.move_count)::int AS move_count").
		ColumnExpr("SUM(wl.total_miles)::float8 AS total_miles").
		ColumnExpr("SUM(wl.shipment_revenue)::float8 AS total_revenue").
		ColumnExpr("MAX(wl.last_ended_at) AS last_ended_at").
		GroupExpr("wl.worker_id")

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker workload: %w", err)
	}

	return entities, nil
}

func (r *repository) ListWorkerLaneExperience(
	ctx context.Context,
	req *repositories.ListLaneExperienceRequest,
) ([]*repositories.WorkerLaneExperience, error) {
	if len(req.WorkerIDs) == 0 || len(req.CustomerIDs) == 0 {
		return []*repositories.WorkerLaneExperience{}, nil
	}

	asnCols := buncolgen.AssignmentColumns
	shipCols := buncolgen.ShipmentColumns
	entities := make([]*repositories.WorkerLaneExperience, 0, len(req.WorkerIDs))

	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr(shipCols.CustomerID.As("customer_id")).
		ColumnExpr("COUNT(*)::int AS move_count").
		Join(moveFromAssignmentJoin).
		Join(shipmentJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(shipCols.CustomerID.In(), bun.List(req.CustomerIDs))
			if req.Since > 0 {
				sq = sq.Where(asnCols.CreatedAt.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(asnCols.PrimaryWorkerID.Qualified() + ", " + shipCols.CustomerID.Qualified())

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker lane experience: %w", err)
	}

	return entities, nil
}

func (r *repository) ListWorkersByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerIDs []pulid.ID,
) ([]*worker.Worker, error) {
	if len(workerIDs) == 0 {
		return []*worker.Worker{}, nil
	}

	cols := buncolgen.WorkerColumns
	entities := make([]*worker.Worker, 0, len(workerIDs))

	err := r.db.DB().NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerRelations.Profile).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, tenantInfo).
				Where(cols.ID.In(), bun.List(workerIDs))
		}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workers by ids: %w", err)
	}

	return entities, nil
}

func (r *repository) ListEquipmentByIDs(
	ctx context.Context,
	req *repositories.ListEquipmentByIDsRequest,
) ([]*tractor.Tractor, []*trailer.Trailer, error) {
	tractors := make([]*tractor.Tractor, 0, len(req.TractorIDs))
	trailers := make([]*trailer.Trailer, 0, len(req.TrailerIDs))

	if len(req.TractorIDs) > 0 {
		err := r.db.DB().NewSelect().
			Model(&tractors).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TractorScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.TractorColumns.ID.In(), bun.List(req.TractorIDs))
			}).
			Scan(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("list tractors by ids: %w", err)
		}
	}

	if len(req.TrailerIDs) > 0 {
		err := r.db.DB().NewSelect().
			Model(&trailers).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TrailerScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
			}).
			Scan(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("list trailers by ids: %w", err)
		}
	}

	return tractors, trailers, nil
}
