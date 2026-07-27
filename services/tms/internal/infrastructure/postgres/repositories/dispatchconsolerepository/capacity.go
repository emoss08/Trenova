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

// ListBoardDrivers returns the capacity rail: active drivers the organization allows to
// be assigned. Drivers temporarily flagged unavailable for dispatch are deliberately
// still returned — a dispatcher needs to see why someone is idle, and the eligibility
// engine is what stops the assignment from going through.
func (r *repository) ListBoardDrivers(
	ctx context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardDriver, error) {
	cols := buncolgen.WorkerColumns
	entities := make([]*repositories.BoardDriver, 0, 128)

	const tractorJoin = `LEFT JOIN LATERAL (
		SELECT trac.id,
			trac.code,
			trac.equipment_type_id,
			trac.fleet_code_id,
			trac.status
		FROM tractors AS trac
		WHERE trac.primary_worker_id = wrk.id
			AND trac.organization_id = wrk.organization_id
			AND trac.business_unit_id = wrk.business_unit_id
		ORDER BY (trac.status = 'Available') DESC, trac.code ASC
		LIMIT 1
	) AS wtrac ON TRUE`

	const openAssignmentJoin = `LEFT JOIN LATERAL (
		SELECT COUNT(*)::int AS open_assignments
		FROM assignments AS a
		JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id
		WHERE a.primary_worker_id = wrk.id
			AND a.organization_id = wrk.organization_id
			AND a.business_unit_id = wrk.business_unit_id
			AND a.archived_at IS NULL
			AND sm.status IN ('Assigned', 'InTransit')
	) AS oa ON TRUE`

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
		ColumnExpr("COALESCE(fc.code, '') AS fleet_code_name").
		ColumnExpr("COALESCE(ust.abbreviation, '') AS state_abbr").
		ColumnExpr("COALESCE(wtrac.id, '') AS tractor_id").
		ColumnExpr("COALESCE(wtrac.code, '') AS tractor_code").
		ColumnExpr("COALESCE(wtrac.equipment_type_id, '') AS tractor_type_id").
		ColumnExpr("COALESCE(wtrac.fleet_code_id, '') AS tractor_fleet_id").
		ColumnExpr("COALESCE(wtrac.status = 'Available', FALSE) AS tractor_status_ok").
		ColumnExpr("COALESCE(oa.open_assignments, 0) AS open_assignments").
		Join("LEFT JOIN fleet_codes AS fc ON "+
			buncolgen.FleetCodeColumns.ID.EqColumn(cols.FleetCodeID)).
		Join("LEFT JOIN us_states AS ust ON "+
			buncolgen.UsStateColumns.ID.EqColumn(cols.StateID)).
		Join(tractorJoin).
		Join(openAssignmentJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerScopeTenant(sq, filter.TenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive).
				Where(cols.CanBeAssigned.IsTrue())

			if len(filter.WorkerIDs) > 0 {
				sq = sq.Where(cols.ID.In(), bun.List(filter.WorkerIDs))
			}
			if len(filter.FleetCodeIDs) > 0 {
				sq = sq.Where(cols.FleetCodeID.In(), bun.List(filter.FleetCodeIDs))
			}
			if filter.Query != "" {
				pattern := "%" + filter.Query + "%"
				sq = sq.WhereGroup(" AND ", func(g *bun.SelectQuery) *bun.SelectQuery {
					return g.Where(cols.FirstName.ILike(), pattern).
						WhereOr(cols.LastName.ILike(), pattern).
						WhereOr("wtrac.code ILIKE ?", pattern)
				})
			}
			return sq
		}).
		Order(cols.LastName.OrderAsc()).
		Order(cols.FirstName.OrderAsc())

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list dispatch board drivers: %w", err)
	}

	return entities, nil
}

// ListWorkerCommitments returns the assignments each driver already holds inside the
// planning horizon. The console draws these as timeline blocks and the eligibility
// engine uses them to refuse a double booking.
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

	const windowJoin = `JOIN LATERAL (
		SELECT MIN(stp.scheduled_window_start) AS window_start,
			MAX(COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)) AS window_end
		FROM stops AS stp
		WHERE stp.shipment_move_id = sm.id
			AND stp.organization_id = sm.organization_id
			AND stp.business_unit_id = sm.business_unit_id
			AND stp.status <> 'Canceled'
	) AS win ON TRUE`

	const destinationJoin = `LEFT JOIN LATERAL (
		SELECT stp.location_id,
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

	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr(asnCols.ShipmentMoveID.As("move_id")).
		ColumnExpr(asnCols.TrailerID.Expr("COALESCE({}, '') AS trailer_id")).
		ColumnExpr(moveCols.ShipmentID.As("shipment_id")).
		ColumnExpr(moveCols.Status.As("move_status")).
		ColumnExpr("COALESCE(sp.pro_number, '') AS pro_number").
		ColumnExpr("COALESCE(win.window_start, 0) AS window_start").
		ColumnExpr("COALESCE(win.window_end, 0) AS window_end").
		ColumnExpr("COALESCE(dest.location_id, '') AS destination_id").
		ColumnExpr("COALESCE(dest.city, '') AS destination_city").
		ColumnExpr("COALESCE(dest.state_abbr, '') AS destination_st").
		ColumnExpr("dest.latitude AS dest_latitude").
		ColumnExpr("dest.longitude AS dest_longitude").
		Join("JOIN shipment_moves AS sm ON "+moveCols.ID.EqColumn(asnCols.ShipmentMoveID)).
		Join("LEFT JOIN shipments AS sp ON "+
			buncolgen.ShipmentColumns.ID.EqColumn(moveCols.ShipmentID)).
		Join(windowJoin).
		Join(destinationJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.ArchivedAt.IsNull()).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(moveCols.Status.In(), bun.List([]shipment.MoveStatus{
					shipment.MoveStatusAssigned,
					shipment.MoveStatusInTransit,
				}))

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

// ListWorkerTimeOff returns approved PTO overlapping the horizon. Only approved time off
// counts: a requested-but-unapproved day should not silently remove capacity.
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

// ListWorkerWorkload is the rolling utilization behind the load-balancing factor.
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

	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr("COUNT(*)::int AS move_count").
		ColumnExpr("COALESCE(SUM(sm.distance), 0)::float8 AS total_miles").
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0)::float8 AS total_revenue").
		ColumnExpr("COALESCE(MAX("+asnCols.UpdatedAt.Qualified()+"), 0) AS last_ended_at").
		Join("JOIN shipment_moves AS sm ON "+moveCols.ID.EqColumn(asnCols.ShipmentMoveID)).
		Join("LEFT JOIN shipments AS sp ON "+
			buncolgen.ShipmentColumns.ID.EqColumn(moveCols.ShipmentID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs))
			if req.Since > 0 {
				sq = sq.Where(asnCols.CreatedAt.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(asnCols.PrimaryWorkerID.Qualified())

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker workload: %w", err)
	}

	return entities, nil
}

// ListWorkerLaneExperience counts how many times each driver has already carried each
// customer's freight. Dispatchers weigh this heavily and nothing else in the system
// captures it.
func (r *repository) ListWorkerLaneExperience(
	ctx context.Context,
	req *repositories.ListLaneExperienceRequest,
) ([]*repositories.WorkerLaneExperience, error) {
	if len(req.WorkerIDs) == 0 || len(req.CustomerIDs) == 0 {
		return []*repositories.WorkerLaneExperience{}, nil
	}

	asnCols := buncolgen.AssignmentColumns
	moveCols := buncolgen.ShipmentMoveColumns
	shipCols := buncolgen.ShipmentColumns
	entities := make([]*repositories.WorkerLaneExperience, 0, len(req.WorkerIDs))

	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr(shipCols.CustomerID.As("customer_id")).
		ColumnExpr("COUNT(*)::int AS move_count").
		Join("JOIN shipment_moves AS sm ON "+moveCols.ID.EqColumn(asnCols.ShipmentMoveID)).
		Join("JOIN shipments AS sp ON "+shipCols.ID.EqColumn(moveCols.ShipmentID)).
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

// ListWorkersByIDs hydrates full worker records with profiles for the eligibility engine,
// which needs licence, medical, and endorsement data the board projection omits.
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

// ListEquipmentByIDs loads the power units and trailers a candidate set references, in
// two queries rather than one per assignment preview.
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
