package workeremploymenteventrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var eventErrors = dberror.Mapping{Entity: "WorkerEmploymentEvent"}

const defaultPageSize = 500

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerEmploymentEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-employment-event-repository"),
	}
}

func eventLoads(includeDocument, includeActors bool) []keel.Loader {
	e := workerdb.WorkerEmploymentEvents

	return append(
		listquery.When(includeDocument, e.Document()),
		listquery.When(includeActors, e.RecordedBy(), e.AmendedBy())...,
	)
}

func listQuery(
	req *workerports.ListWorkerEmploymentEventsRequest,
) *keel.Query[*employment.WorkerEmploymentEvent] {
	e := workerdb.WorkerEmploymentEvents
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}

	order := []keel.Order{e.EffectiveAt.Desc(), e.CreatedAt.Desc()}
	if req.Ascending {
		order = []keel.Order{e.EffectiveAt.Asc(), e.CreatedAt.Asc()}
	}

	return e.Where(
		e.WorkerID.Eq(req.WorkerID),
		keel.When(len(req.Kinds) > 0, e.Kind.In(req.Kinds...)),
	).
		Load(eventLoads(req.IncludeDocument, req.IncludeActors)...).
		Order(order...).
		Limit(limit)
}

func (r *repository) List(
	ctx context.Context,
	req *workerports.ListWorkerEmploymentEventsRequest,
) ([]*employment.WorkerEmploymentEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*employment.WorkerEmploymentEvent, error) {
			entities, err := listQuery(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list worker employment events", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerports.GetWorkerEmploymentEventByIDRequest,
) (*employment.WorkerEmploymentEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*employment.WorkerEmploymentEvent, error) {
			e := workerdb.WorkerEmploymentEvents
			entity, err := e.Where(e.ID.Eq(req.ID)).
				Load(eventLoads(req.IncludeDocument, req.IncludeActors)...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, eventErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *employment.WorkerEmploymentEvent,
) (*employment.WorkerEmploymentEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*employment.WorkerEmploymentEvent, error) {
			if err := workerdb.WorkerEmploymentEvents.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to create worker employment event", zap.Error(err))
				return nil, eventErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *employment.WorkerEmploymentEvent,
) (*employment.WorkerEmploymentEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*employment.WorkerEmploymentEvent, error) {
			if err := workerdb.WorkerEmploymentEvents.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to update worker employment event", zap.Error(err))
				return nil, eventErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
