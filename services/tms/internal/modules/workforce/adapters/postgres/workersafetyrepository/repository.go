package workersafetyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline/workerdisciplinedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety/workersafetydb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var eventErrors = dberror.Mapping{Entity: "WorkerSafetyEvent"}

var actionErrors = dberror.Mapping{Entity: "WorkerDisciplinaryAction"}

var recognitionErrors = dberror.Mapping{Entity: "WorkerRecognition"}

type Params struct {
	fx.In

	Entitlements entitlementports.Filter
	DB           *postgres.Connection
	Logger       *zap.Logger
}

type repository struct {
	entitlements entitlementports.Filter
	db           *postgres.Connection
	l            *zap.Logger
}

func New(p Params) workersafetyports.WorkerSafetyRepository {
	return &repository{
		entitlements: p.Entitlements,
		db:           p.DB,
		l:            p.Logger.Named("postgres.worker-safety-repository"),
	}
}

func eventLoads(includeDocument, includeActors, includeWorker bool) []keel.Loader {
	e := workersafetydb.WorkerSafetyEvents
	loads := make([]keel.Loader, 0, 4)
	loads = append(loads, listquery.When(includeDocument, e.Document())...)
	loads = append(loads, listquery.When(includeActors, e.RecordedBy(), e.ClosedBy())...)

	return append(loads, listquery.When(includeWorker, e.Worker())...)
}

func eventsOf(
	req *workersafetyports.ListWorkerSafetyEventsRequest,
) keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents

	return e.Where(
		keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
		keel.When(req.OpenOnly, e.Status.In(
			workersafety.SafetyEventStatusOpen,
			workersafety.SafetyEventStatusUnderReview,
		)),
		keel.When(req.Since > 0, e.OccurredAt.Gte(req.Since)),
		keel.When(req.DecisiveAt > 0, keel.Or(
			e.OccurredAt.Gte(req.DecisiveSince),
			e.Status.Ne(workersafety.SafetyEventStatusClosed),
			keel.And(
				e.Points.Gt(0),
				keel.Or(
					e.PointsExpireAt.IsNull(),
					e.PointsExpireAt.Lte(0),
					e.PointsExpireAt.Gt(req.DecisiveAt),
				),
			),
		)),
	)
}

func (r *repository) LastCountingEventAt(
	ctx context.Context,
	req *workersafetyports.LastCountingSafetyEventRequest,
) (*int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*int64, error) {
		e := workersafetydb.WorkerSafetyEvents
		last, err := keel.IntoOne[*int64](
			ctx,
			r.db.Keel(ctx),
			e.Pick(e.OccurredAt.Max()).Where(
				e.WorkerID.Eq(req.WorkerID),
				e.Kind.In(workersafety.CountingKinds()...),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("read last counting safety event: %w", err)
		}

		return last, nil
	})
}

func eventsQuery(
	req *workersafetyports.ListWorkerSafetyEventsRequest,
) keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents
	q := eventsOf(req).
		Load(eventLoads(req.IncludeDocument, req.IncludeActors, req.IncludeWorker)...).
		Order(e.OccurredAt.Desc(), e.CreatedAt.Desc())
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	return q
}

func (r *repository) ListEvents(
	ctx context.Context,
	req *workersafetyports.ListWorkerSafetyEventsRequest,
) ([]*workersafety.WorkerSafetyEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workersafety.WorkerSafetyEvent, error) {
			entities, err := eventsQuery(req).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list worker safety events", zap.Error(err))
				return nil, fmt.Errorf("list worker safety events: %w", err)
			}

			return entities, nil
		},
	)
}

func newestFirst(at keel.Order, atField string, created, id keel.Order) *listquery.Keyset {
	desc := string(dbtype.SortDirectionDesc)

	return &listquery.Keyset{
		Orders: []keel.Order{at, created, id},
		Shape: []pagination.CursorSortField{
			{Field: atField, Direction: desc},
			{Field: "createdAt", Direction: desc},
			{Field: "id", Direction: desc},
		},
	}
}

func (r *repository) GetEventByID(
	ctx context.Context,
	req *workersafetyports.GetWorkerSafetyEventByIDRequest,
) (*workersafety.WorkerSafetyEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workersafety.WorkerSafetyEvent, error) {
		e := workersafetydb.WorkerSafetyEvents
		entity, err := e.Where(e.ID.Eq(req.ID)).
			Load(eventLoads(req.IncludeDocument, false, req.IncludeWorker)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, eventErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateEvent(
	ctx context.Context,
	entity *workersafety.WorkerSafetyEvent,
) (*workersafety.WorkerSafetyEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workersafety.WorkerSafetyEvent, error) {
			if err := workersafetydb.WorkerSafetyEvents.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create worker safety event", zap.Error(err))
				return nil, eventErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateEvent(
	ctx context.Context,
	entity *workersafety.WorkerSafetyEvent,
) (*workersafety.WorkerSafetyEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workersafety.WorkerSafetyEvent, error) {
			if err := workersafetydb.WorkerSafetyEvents.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update worker safety event", zap.Error(err))
				return nil, eventErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) DeleteEvent(
	ctx context.Context,
	req *workersafetyports.GetWorkerSafetyEventByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		e := workersafetydb.WorkerSafetyEvents
		rows, err := e.Where(e.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete worker safety event", zap.Error(err))
			return eventErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(rows, eventErrors.Entity, req.ID.String())
	})
}

func (r *repository) ListActions(
	ctx context.Context,
	req *workersafetyports.ListWorkerDisciplinaryActionsRequest,
) ([]*workerdiscipline.WorkerDisciplinaryAction, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerdiscipline.WorkerDisciplinaryAction, error) {
			a := workerdisciplinedb.WorkerDisciplinaryActions
			entities, err := actionsOf(req).
				Load(actionLoads(req)...).
				Order(a.IssuedAt.Desc(), a.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list disciplinary actions", zap.Error(err))
				return nil, fmt.Errorf("list disciplinary actions: %w", err)
			}

			return entities, nil
		},
	)
}

func actionsOf(
	req *workersafetyports.ListWorkerDisciplinaryActionsRequest,
) keel.Query[*workerdiscipline.WorkerDisciplinaryAction] {
	a := workerdisciplinedb.WorkerDisciplinaryActions

	return a.Where(
		a.WorkerID.Eq(req.WorkerID),
		keel.When(req.ActiveOnly, a.Status.Eq(workerdiscipline.DisciplinaryStatusActive)),
	)
}

func actionLoads(req *workersafetyports.ListWorkerDisciplinaryActionsRequest) []keel.Loader {
	a := workerdisciplinedb.WorkerDisciplinaryActions
	loads := listquery.When(req.IncludeEvent, a.SafetyEvent())

	return append(loads, listquery.When(req.IncludeActors, a.IssuedBy())...)
}

func (r *repository) ActionsPage(
	ctx context.Context,
	req *workersafetyports.ListWorkerDisciplinaryActionsRequest,
) (*pagination.CursorListResult[*workerdiscipline.WorkerDisciplinaryAction], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(
			ctx context.Context,
		) (*pagination.CursorListResult[*workerdiscipline.WorkerDisciplinaryAction], error) {
			a := workerdisciplinedb.WorkerDisciplinaryActions
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				actionsOf(req),
				&listquery.Request[workerdiscipline.WorkerDisciplinaryAction]{
					Filter: &pagination.QueryOptions{TenantInfo: req.TenantInfo},
					Cursor: req.Cursor,
					Shape: func(
						q keel.Query[*workerdiscipline.WorkerDisciplinaryAction],
					) keel.Query[*workerdiscipline.WorkerDisciplinaryAction] {
						return q.Load(actionLoads(req)...)
					},
					Keyset: newestFirst(a.IssuedAt.Desc(), "issuedAt", a.CreatedAt.Desc(), a.ID.Desc()),
				},
			)
			if err != nil {
				return nil, fmt.Errorf("page disciplinary actions: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetActionByID(
	ctx context.Context,
	req *workersafetyports.GetWorkerDisciplinaryActionByIDRequest,
) (*workerdiscipline.WorkerDisciplinaryAction, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerdiscipline.WorkerDisciplinaryAction, error) {
			a := workerdisciplinedb.WorkerDisciplinaryActions
			entity, err := a.Where(a.ID.Eq(req.ID)).
				Load(actionLoads(&workersafetyports.ListWorkerDisciplinaryActionsRequest{
					IncludeEvent:  req.IncludeEvent,
					IncludeActors: req.IncludeActors,
				})...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, actionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateAction(
	ctx context.Context,
	entity *workerdiscipline.WorkerDisciplinaryAction,
) (*workerdiscipline.WorkerDisciplinaryAction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerdiscipline.WorkerDisciplinaryAction, error) {
			err := workerdisciplinedb.WorkerDisciplinaryActions.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create disciplinary action", zap.Error(err))
				return nil, actionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateAction(
	ctx context.Context,
	entity *workerdiscipline.WorkerDisciplinaryAction,
) (*workerdiscipline.WorkerDisciplinaryAction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerdiscipline.WorkerDisciplinaryAction, error) {
			err := workerdisciplinedb.WorkerDisciplinaryActions.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update disciplinary action", zap.Error(err))
				return nil, actionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListRecognitions(
	ctx context.Context,
	req *workersafetyports.ListWorkerRecognitionsRequest,
) ([]*workerdiscipline.WorkerRecognition, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerdiscipline.WorkerRecognition, error) {
			rc := workerdisciplinedb.WorkerRecognitions
			q := recognitionsOf(req).
				Load(listquery.When(req.IncludeActors, rc.AwardedBy())...).
				Order(rc.OccurredAt.Desc(), rc.CreatedAt.Desc(), rc.ID.Desc())
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}
			entities, err := q.All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list recognitions", zap.Error(err))
				return nil, fmt.Errorf("list recognitions: %w", err)
			}

			return entities, nil
		},
	)
}

func recognitionsOf(
	req *workersafetyports.ListWorkerRecognitionsRequest,
) keel.Query[*workerdiscipline.WorkerRecognition] {
	rc := workerdisciplinedb.WorkerRecognitions

	return rc.Where(
		rc.WorkerID.Eq(req.WorkerID),
		keel.When(req.VisibleOnly, rc.VisibleToWorker.IsTrue()),
		keel.When(req.Since > 0, rc.OccurredAt.Gte(req.Since)),
	)
}

func (r *repository) GetRecognitionByID(
	ctx context.Context,
	req *workersafetyports.GetWorkerRecognitionByIDRequest,
) (*workerdiscipline.WorkerRecognition, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerdiscipline.WorkerRecognition, error) {
			entity, err := workerdisciplinedb.WorkerRecognitions.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, recognitionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateRecognition(
	ctx context.Context,
	entity *workerdiscipline.WorkerRecognition,
) (*workerdiscipline.WorkerRecognition, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerdiscipline.WorkerRecognition, error) {
			if err := workerdisciplinedb.WorkerRecognitions.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create recognition", zap.Error(err))
				return nil, recognitionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) DeleteRecognition(
	ctx context.Context,
	req *workersafetyports.GetWorkerRecognitionByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rc := workerdisciplinedb.WorkerRecognitions
		rows, err := rc.Where(rc.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete recognition", zap.Error(err))
			return recognitionErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(rows, recognitionErrors.Entity, req.ID.String())
	})
}

func lapsedQuery(
	req *workersafetyports.ListWorkersWithLapsedPointsRequest,
	eventsEntitled keel.Pred,
	actionsEntitled keel.Pred,
) keel.Query[*keel.Row] {
	e := workersafetydb.WorkerSafetyEvents
	a := workerdisciplinedb.WorkerDisciplinaryActions

	events := e.AllTenants().
		Pick(e.WorkerID, e.OrganizationID, e.BusinessUnitID).
		Where(eventsEntitled, e.PointsExpireAt.Gte(req.Since), e.PointsExpireAt.Lt(req.Until))
	actions := a.AllTenants().
		Pick(a.WorkerID, a.OrganizationID, a.BusinessUnitID).
		Where(actionsEntitled, a.ExpiresAt.Gte(req.Since), a.ExpiresAt.Lt(req.Until))

	lapsed := keel.CTE(
		"lapsed",
		[]string{"worker_id", "organization_id", "business_unit_id"},
		keel.Union(events, actions),
	)
	q := keel.RowsOf(lapsed.TableRef()).
		With(lapsed).
		Pick(
			lapsed.Col("worker_id"),
			lapsed.Col("organization_id"),
			lapsed.Col("business_unit_id"),
		).
		Order(lapsed.Col("worker_id").Asc())
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	return q
}

func (r *repository) ListWorkersWithLapsedPoints(
	ctx context.Context,
	req *workersafetyports.ListWorkersWithLapsedPointsRequest,
) ([]workerports.WorkerTenantRef, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list workers with lapsed safety points across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.WorkerTenantRef, error) {
		eventsEntitled, err := r.entitlements.ForWork(
			ctx,
			workersafetydb.WorkerSafetyEvents.OrganizationID.Col,
		)
		if err != nil {
			return nil, err
		}
		actionsEntitled, err := r.entitlements.ForWork(
			ctx,
			workerdisciplinedb.WorkerDisciplinaryActions.OrganizationID.Col,
		)
		if err != nil {
			return nil, err
		}
		refs, err := keel.Into[workerports.WorkerTenantRef](
			ctx,
			r.db.Keel(ctx),
			lapsedQuery(req, eventsEntitled, actionsEntitled),
		)
		if err != nil {
			return nil, fmt.Errorf("list workers with lapsed safety points: %w", err)
		}

		return refs, nil
	})
}
