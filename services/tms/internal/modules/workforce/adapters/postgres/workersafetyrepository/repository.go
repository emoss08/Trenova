package workersafetyrepository

import (
	"context"
	"fmt"

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
	"github.com/emoss08/trenova/pkg/listquery"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var eventErrors = dberror.Mapping{Entity: "WorkerSafetyEvent"}

var actionErrors = dberror.Mapping{Entity: "WorkerDisciplinaryAction"}

var recognitionErrors = dberror.Mapping{Entity: "WorkerRecognition"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workersafetyports.WorkerSafetyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-safety-repository"),
	}
}

func eventLoads(includeDocument, includeActors, includeWorker bool) []keel.Loader {
	e := workersafetydb.WorkerSafetyEvents
	loads := make([]keel.Loader, 0, 4)
	loads = append(loads, listquery.When(includeDocument, e.Document())...)
	loads = append(loads, listquery.When(includeActors, e.RecordedBy(), e.ClosedBy())...)

	return append(loads, listquery.When(includeWorker, e.Worker())...)
}

func eventsQuery(
	req *workersafetyports.ListWorkerSafetyEventsRequest,
) *keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents
	q := e.Where(
		keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
		keel.When(req.OpenOnly, e.Status.In(
			workersafety.SafetyEventStatusOpen,
			workersafety.SafetyEventStatusUnderReview,
		)),
		keel.When(req.Since > 0, e.OccurredAt.Gte(req.Since)),
	).
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
			loads := listquery.When(req.IncludeEvent, a.SafetyEvent())
			loads = append(loads, listquery.When(req.IncludeActors, a.IssuedBy())...)

			entities, err := a.Where(a.WorkerID.Eq(req.WorkerID)).
				Load(loads...).
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
				Load(listquery.When(req.IncludeEvent, a.SafetyEvent())...).
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
			entities, err := rc.Where(
				rc.WorkerID.Eq(req.WorkerID),
				keel.When(req.VisibleOnly, rc.VisibleToWorker.IsTrue()),
			).
				Load(listquery.When(req.IncludeActors, rc.AwardedBy())...).
				Order(rc.OccurredAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list recognitions", zap.Error(err))
				return nil, fmt.Errorf("list recognitions: %w", err)
			}

			return entities, nil
		},
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

func lapsedQuery(req *workersafetyports.ListWorkersWithLapsedPointsRequest) *keel.Query[*keel.Row] {
	e := workersafetydb.WorkerSafetyEvents
	a := workerdisciplinedb.WorkerDisciplinaryActions

	events := e.AllTenants().
		Pick(e.WorkerID, e.OrganizationID, e.BusinessUnitID).
		Where(e.PointsExpireAt.Gte(req.Since), e.PointsExpireAt.Lt(req.Until))
	actions := a.AllTenants().
		Pick(a.WorkerID, a.OrganizationID, a.BusinessUnitID).
		Where(a.ExpiresAt.Gte(req.Since), a.ExpiresAt.Lt(req.Until))

	lapsed := keel.CTE(
		"lapsed",
		[]string{"worker_id", "organization_id", "business_unit_id"},
		keel.Union(events, actions),
	)
	q := lapsed.Query().
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
		refs, err := keel.Into[workerports.WorkerTenantRef](ctx, r.db.Keel(ctx), lapsedQuery(req))
		if err != nil {
			return nil, fmt.Errorf("list workers with lapsed safety points: %w", err)
		}

		return refs, nil
	})
}
