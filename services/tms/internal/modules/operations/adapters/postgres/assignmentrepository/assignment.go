//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package assignmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var assignmentErrors = dberror.Mapping{Entity: "Assignment"}

var activeMoveStatuses = []shipmentvocab.MoveStatus{
	shipmentvocab.MoveStatusInTransit,
	shipmentvocab.MoveStatusAssigned,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) assignmentports.AssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assignment-repository"),
	}
}

type assignmentQuery = keel.Query[*shipment.Assignment]

func liveAssignments() *assignmentQuery {
	a := shipmentdb.Assignments
	return a.Where(a.ArchivedAt.IsNull())
}

func liveOnMove(moveID pulid.ID) *assignmentQuery {
	return liveAssignments().Where(shipmentdb.Assignments.ShipmentMoveID.Eq(moveID))
}

func unassignable(entity *shipment.Assignment) *assignmentQuery {
	a := shipmentdb.Assignments
	return liveAssignments().Where(a.ID.Eq(entity.ID), a.Version.Eq(entity.Version))
}

func withResources(q *assignmentQuery) *assignmentQuery {
	a := shipmentdb.Assignments
	return q.Load(
		a.ShipmentMove(),
		a.Tractor(),
		a.Trailer(),
		a.PrimaryWorker(),
		a.SecondaryWorker(),
	)
}

func (r *repository) List(
	ctx context.Context,
	req *assignmentports.ListAssignmentsRequest,
) (*pagination.ListResult[*shipment.Assignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipment.Assignment], error) {
			a := shipmentdb.Assignments
			live := liveAssignments()

			var items []*shipment.Assignment
			var total keel.RowCount
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.ListInto(withResources(live.Clone()).
					Order(a.CreatedAt.Desc()).
					Limit(req.Filter.Pagination.SafeLimit()).
					Offset(req.Filter.Pagination.SafeOffset()), &items),
				keel.CountEstimateInto(live, pagination.ExactTotalUpTo, &total),
			)
			if err != nil {
				return nil, err
			}

			return listquery.Result(items, total), nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *assignmentports.GetAssignmentByIDRequest,
) (*shipment.Assignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		return r.get(ctx, req.AssignmentID)
	})
}

func (r *repository) get(ctx context.Context, id pulid.ID) (*shipment.Assignment, error) {
	a := shipmentdb.Assignments
	entity, err := withResources(a.Where(a.ID.Eq(id))).One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, assignmentErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) GetByMoveID(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.Assignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		entity, err := liveOnMove(moveID).First(ctx, r.db.Keel(ctx))
		return optional(entity, err)
	})
}

func (r *repository) GetMoveByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentMove, error) {
		move, err := shipmentdb.ShipmentMoves.Get(ctx, r.db.Keel(ctx), moveID)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Shipment move")
		}

		return move, nil
	})
}

func optional(entity *shipment.Assignment, err error) (*shipment.Assignment, error) {
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
		}
		return nil, err
	}

	return entity, nil
}

func (r *repository) FindInProgressByTractorID(
	ctx context.Context,
	_ pagination.TenantInfo,
	tractorID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findInProgressAssignment(
		ctx,
		shipmentdb.Assignments.TractorID,
		tractorID,
		excludeMoveID,
	)
}

func (r *repository) FindInProgressByTrailerID(
	ctx context.Context,
	_ pagination.TenantInfo,
	trailerID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findInProgressAssignment(
		ctx,
		shipmentdb.Assignments.TrailerID,
		trailerID,
		excludeMoveID,
	)
}

func (r *repository) FindInProgressByPrimaryWorkerID(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findInProgressAssignment(
		ctx,
		shipmentdb.Assignments.PrimaryWorkerID,
		workerID,
		excludeMoveID,
	)
}

func (r *repository) FindActiveByTractorID(
	ctx context.Context,
	_ pagination.TenantInfo,
	tractorID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findActiveAssignment(ctx, shipmentdb.Assignments.TractorID.Eq(tractorID))
}

func (r *repository) FindActiveByWorkerID(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*shipment.Assignment, error) {
	a := shipmentdb.Assignments
	return r.findActiveAssignment(
		ctx,
		keel.Or(a.PrimaryWorkerID.Eq(workerID), a.SecondaryWorkerID.Eq(workerID)),
	)
}

func (r *repository) findActiveAssignment(
	ctx context.Context,
	subject keel.Pred,
) (*shipment.Assignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		a := shipmentdb.Assignments
		sm := shipmentdb.ShipmentMoves
		entity, err := a.Join(sm, sm.ID.EqCol(a.ShipmentMoveID)).
			Where(a.ArchivedAt.IsNull(), subject, sm.Status.In(activeMoveStatuses...)).
			Order(
				keel.Desc(sm.Status.Eq(shipmentvocab.MoveStatusInTransit)),
				sm.CreatedAt.Desc(),
			).
			First(ctx, r.db.Keel(ctx))
		return optional(entity, err)
	})
}

func (r *repository) findInProgressAssignment(
	ctx context.Context,
	column keelkinds.PulidIDCol,
	equipmentID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		a := shipmentdb.Assignments
		sm := shipmentdb.ShipmentMoves
		entity, err := a.Join(sm, sm.ID.EqCol(a.ShipmentMoveID)).
			Where(
				a.ArchivedAt.IsNull(),
				column.Eq(equipmentID),
				sm.Status.Eq(shipmentvocab.MoveStatusInTransit),
				keel.When(!excludeMoveID.IsNil(), a.ShipmentMoveID.Ne(excludeMoveID)),
			).
			First(ctx, r.db.Keel(ctx))
		return optional(entity, err)
	})
}

func (r *repository) FindNearestActualEventByTractorID(
	ctx context.Context,
	req assignmentports.FindNearestActualTimelineEventRequest,
	tractorID pulid.ID,
) (*assignmentports.ActualTimelineEvent, error) {
	return r.findNearestActualEvent(ctx, req, shipmentdb.Assignments.TractorID, tractorID)
}

func (r *repository) FindNearestActualEventByPrimaryWorkerID(
	ctx context.Context,
	req assignmentports.FindNearestActualTimelineEventRequest,
	workerID pulid.ID,
) (*assignmentports.ActualTimelineEvent, error) {
	return r.findNearestActualEvent(ctx, req, shipmentdb.Assignments.PrimaryWorkerID, workerID)
}

func (r *repository) FindOverlappingActualWindowByTractorID(
	ctx context.Context,
	req assignmentports.FindOverlappingActualTimelineWindowRequest,
	tractorID pulid.ID,
) (*assignmentports.ActualTimelineWindow, error) {
	return r.findOverlappingActualWindow(ctx, shipmentdb.Assignments.TractorID, tractorID, req)
}

func (r *repository) FindOverlappingActualWindowByPrimaryWorkerID(
	ctx context.Context,
	req assignmentports.FindOverlappingActualTimelineWindowRequest,
	workerID pulid.ID,
) (*assignmentports.ActualTimelineWindow, error) {
	return r.findOverlappingActualWindow(ctx, shipmentdb.Assignments.PrimaryWorkerID, workerID, req)
}

func (r *repository) findNearestActualEvent(
	ctx context.Context,
	req assignmentports.FindNearestActualTimelineEventRequest,
	column keelkinds.PulidIDCol,
	resourceID pulid.ID,
) (*assignmentports.ActualTimelineEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*assignmentports.ActualTimelineEvent, error) {
			ev := actualTimelineEvents(column, resourceID)
			ts := ev.Col("timestamp")

			near := ts.Lte(req.Timestamp)
			order := keel.Desc(ts)
			if req.Direction == assignmentports.ActualTimelineDirectionNext {
				near = ts.Gte(req.Timestamp)
				order = keel.Asc(ts)
			}

			events, err := keel.Into[assignmentports.ActualTimelineEvent](ctx, r.db.Keel(ctx), ev.
				Query().
				With(ev).
				Pick(
					ts,
					ev.Col("event_type"),
					ev.Col("stop_id"),
					ev.Col("shipment_move_id"),
					ev.Col("shipment_id"),
					ev.Col("location_name"),
				).
				Where(
					near,
					keel.When(!req.ExcludeShipmentID.IsNil(),
						ev.Col("shipment_id").Ne(req.ExcludeShipmentID.String())),
				).
				Order(order).
				Limit(1))
			if err != nil {
				return nil, err
			}
			if len(events) == 0 {
				return nil, nil
			}

			return &events[0], nil
		},
	)
}

func (r *repository) findOverlappingActualWindow(
	ctx context.Context,
	column keelkinds.PulidIDCol,
	resourceID pulid.ID,
	req assignmentports.FindOverlappingActualTimelineWindowRequest,
) (*assignmentports.ActualTimelineWindow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*assignmentports.ActualTimelineWindow, error) {
			ev := actualTimelineEvents(column, resourceID)
			ts := ev.Col("timestamp")
			moveID := ev.Col("shipment_move_id")
			shipmentID := ev.Col("shipment_id")

			windows, err := keel.Into[assignmentports.ActualTimelineWindow](ctx, r.db.Keel(ctx), ev.
				Query().
				With(ev).
				Group(moveID, shipmentID).
				Pick(keel.Min(ts), keel.Max(ts), moveID, shipmentID).
				Where(keel.When(!req.ExcludeShipmentID.IsNil(),
					shipmentID.Ne(req.ExcludeShipmentID.String()))).
				Having(keel.Min(ts).Lte(req.Timestamp), keel.Max(ts).Gte(req.Timestamp)).
				Order(keel.Min(ts).Desc()).
				Limit(1))
			if err != nil {
				return nil, err
			}
			if len(windows) == 0 {
				return nil, nil
			}

			return &windows[0], nil
		},
	)
}

// actualTimelineEvents is every actual arrival and departure on the live
// assignments of one resource, as one row per event.
func actualTimelineEvents(column keelkinds.PulidIDCol, resourceID pulid.ID) *keel.CTETable {
	stopEvents := func(
		at func(*shipmentdb.StopsTable) keel.Int64Col,
		kind assignmentports.ActualTimelineEventType,
	) *keel.Query[*shipment.Assignment] {
		a := shipmentdb.Assignments
		sm := shipmentdb.ShipmentMoves
		st := shipmentdb.Stops
		loc := locationdb.Locations
		return a.
			Join(sm, sm.ID.EqCol(a.ShipmentMoveID)).
			Join(st, st.ShipmentMoveID.EqCol(sm.ID)).
			LeftJoin(loc, loc.ID.EqCol(st.LocationID)).
			Pick(
				at(st),
				keel.Text(string(kind)),
				st.ID,
				st.ShipmentMoveID,
				sm.ShipmentID,
				loc.Name.Coalesce(""),
			).
			Where(
				a.ArchivedAt.IsNull(),
				column.Eq(resourceID),
				sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
				st.Status.Ne(shipmentvocab.StopStatusCanceled),
				at(st).IsNotNull(),
			)
	}

	return keel.CTE(
		"actual_event",
		[]string{
			"timestamp",
			"event_type",
			"stop_id",
			"shipment_move_id",
			"shipment_id",
			"location_name",
		},
		keel.UnionAll(
			stopEvents(func(st *shipmentdb.StopsTable) keel.Int64Col { return st.ActualArrival },
				assignmentports.ActualTimelineEventTypeArrival),
			stopEvents(func(st *shipmentdb.StopsTable) keel.Int64Col { return st.ActualDeparture },
				assignmentports.ActualTimelineEventTypeDeparture),
		),
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		if err := shipmentdb.Assignments.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, assignmentErrors.Map(err, entity.ID.String())
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		a := shipmentdb.Assignments
		if err := a.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			a.PrimaryWorkerID,
			a.TractorID,
			a.TrailerID,
			a.SecondaryWorkerID,
			a.Status,
		); err != nil {
			return nil, fmt.Errorf(
				"update assignment: %w",
				assignmentErrors.Map(err, entity.ID.String()),
			)
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Unassign(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		now := timeutils.NowUnix()
		a := shipmentdb.Assignments
		n, err := unassignable(entity).Update(
			ctx,
			r.db.Keel(ctx),
			a.Status.To(shipmentvocab.AssignmentStatusCanceled),
			a.ArchivedAt.To(now),
		)
		if err != nil {
			return nil, fmt.Errorf("unassign assignment: %w", err)
		}
		if err = dberror.CheckAffected(n, "Assignment", entity.ID.String()); err != nil {
			return nil, err
		}

		return r.get(ctx, entity.ID)
	})
}
