package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
)

var (
	stopFromMoveJoin = "JOIN " + buncolgen.StopTable.As(buncolgen.StopTable.Alias) +
		" ON " + buncolgen.StopColumns.ShipmentMoveID.EqColumn(
		buncolgen.ShipmentMoveColumns.ID) +
		" AND " + buncolgen.StopColumns.OrganizationID.EqColumn(
		buncolgen.ShipmentMoveColumns.OrganizationID) +
		" AND " + buncolgen.StopColumns.BusinessUnitID.EqColumn(
		buncolgen.ShipmentMoveColumns.BusinessUnitID)

	assignmentFromFailureJoin = "JOIN " + buncolgen.AssignmentTable.As(
		buncolgen.AssignmentTable.Alias) +
		" ON " + buncolgen.AssignmentColumns.ShipmentMoveID.EqColumn(
		buncolgen.ServiceFailureColumns.ShipmentMoveID) +
		" AND " + buncolgen.AssignmentColumns.OrganizationID.EqColumn(
		buncolgen.ServiceFailureColumns.OrganizationID) +
		" AND " + buncolgen.AssignmentColumns.BusinessUnitID.EqColumn(
		buncolgen.ServiceFailureColumns.BusinessUnitID)

	reasonCodeFromFailureJoin = "JOIN " + buncolgen.ReasonCodeTable.As(
		buncolgen.ReasonCodeTable.Alias) +
		" ON " + buncolgen.ReasonCodeColumns.ID.EqColumn(
		buncolgen.ServiceFailureColumns.ReasonCodeID) +
		" AND " + buncolgen.ReasonCodeColumns.OrganizationID.EqColumn(
		buncolgen.ServiceFailureColumns.OrganizationID) +
		" AND " + buncolgen.ReasonCodeColumns.BusinessUnitID.EqColumn(
		buncolgen.ServiceFailureColumns.BusinessUnitID)
)

func (r *repository) ListWorkerOnTimeStats(
	ctx context.Context,
	req *repositories.ListWorkloadRequest,
) ([]*repositories.WorkerOnTimeStats, error) {
	if len(req.WorkerIDs) == 0 {
		return []*repositories.WorkerOnTimeStats{}, nil
	}

	asnCols := buncolgen.AssignmentColumns
	stopCols := buncolgen.StopColumns

	arrivals := make([]*repositories.WorkerOnTimeStats, 0, len(req.WorkerIDs))
	arrivalsQuery := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr("COUNT(*)::int AS completed_stops").
		ColumnExpr(buncolgen.Expr(
			"COUNT(*) FILTER (WHERE {0} <= COALESCE({1}, {2}))::int AS on_time_stops",
			stopCols.ActualArrival,
			stopCols.ScheduledWindowEnd,
			stopCols.ScheduledWindowStart,
		)).
		Join(moveFromAssignmentJoin).
		Join(stopFromMoveJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(stopCols.Status.Eq(), shipment.StopStatusCompleted).
				Where(stopCols.Type.In(),
					bun.List([]shipment.StopType{
						shipment.StopTypeDelivery,
						shipment.StopTypeSplitDelivery,
					})).
				Where(stopCols.ActualArrival.IsNotNull()).
				Where(stopCols.ScheduledWindowStart.Gt(), 0)
			if req.Since > 0 {
				sq = sq.Where(stopCols.ActualArrival.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(asnCols.PrimaryWorkerID.Qualified())

	if err := arrivalsQuery.Scan(ctx, &arrivals); err != nil {
		return nil, fmt.Errorf("list worker on-time arrivals: %w", err)
	}

	sfCols := buncolgen.ServiceFailureColumns

	failures := make([]*repositories.WorkerOnTimeStats, 0, len(req.WorkerIDs))
	failuresQuery := r.db.DB().NewSelect().
		Model((*servicefailure.ServiceFailure)(nil)).
		ColumnExpr(asnCols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr("COUNT(*)::int AS driver_fault_failures").
		Join(assignmentFromFailureJoin).
		Join(reasonCodeFromFailureJoin).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ServiceFailureScopeTenant(sq, req.TenantInfo).
				Where(asnCols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs)).
				Where(sfCols.Status.NotEq(), servicefailure.StatusVoided).
				Where(buncolgen.ReasonCodeColumns.Category.Eq(),
					servicefailure.ReasonCategoryDriver)
			if req.Since > 0 {
				sq = sq.Where(sfCols.DetectedAt.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(asnCols.PrimaryWorkerID.Qualified())

	if err := failuresQuery.Scan(ctx, &failures); err != nil {
		return nil, fmt.Errorf("list worker driver-fault failures: %w", err)
	}

	merged := make(map[string]*repositories.WorkerOnTimeStats, len(arrivals))
	for _, row := range arrivals {
		merged[row.WorkerID.String()] = row
	}
	for _, row := range failures {
		if existing, ok := merged[row.WorkerID.String()]; ok {
			existing.DriverFaultFailures = row.DriverFaultFailures
			continue
		}
		merged[row.WorkerID.String()] = row
	}

	entities := make([]*repositories.WorkerOnTimeStats, 0, len(merged))
	for _, row := range merged {
		entities = append(entities, row)
	}
	return entities, nil
}

func (r *repository) ListWorkerAcceptanceStats(
	ctx context.Context,
	req *repositories.ListWorkloadRequest,
) ([]*repositories.WorkerAcceptanceStats, error) {
	if len(req.WorkerIDs) == 0 {
		return []*repositories.WorkerAcceptanceStats{}, nil
	}

	cols := buncolgen.AssignmentColumns

	entities := make([]*repositories.WorkerAcceptanceStats, 0, len(req.WorkerIDs))
	q := r.db.DB().NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(cols.PrimaryWorkerID.As("worker_id")).
		ColumnExpr(buncolgen.CountFilter("accepted", cols.AckStatus.Eq()),
			shipment.AssignmentAckAccepted).
		ColumnExpr(buncolgen.CountFilter("declined", cols.AckStatus.Eq()),
			shipment.AssignmentAckDeclined).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(AVG({0} - {1}) FILTER (WHERE {2} IN (?, ?) AND {0} IS NOT NULL AND {0} >= {1}), 0)::float8 AS avg_ack_latency_seconds",
			cols.AckAt,
			cols.CreatedAt,
			cols.AckStatus,
		), shipment.AssignmentAckAccepted, shipment.AssignmentAckDeclined).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.PrimaryWorkerID.In(), bun.List(req.WorkerIDs))
			if req.Since > 0 {
				sq = sq.Where(cols.CreatedAt.Gte(), req.Since)
			}
			return sq
		}).
		GroupExpr(cols.PrimaryWorkerID.Qualified())

	if err := q.Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("list worker acceptance stats: %w", err)
	}
	return entities, nil
}
