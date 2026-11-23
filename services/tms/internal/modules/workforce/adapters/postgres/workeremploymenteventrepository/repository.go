package workeremploymenteventrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment/employmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) workeremploymentports.WorkerEmploymentEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-employment-event-repository"),
	}
}

func eventLoads(includeDocument, includeActors bool) []keel.Loader {
	e := employmentdb.WorkerEmploymentEvents

	return append(
		listquery.When(includeDocument, e.Document()),
		listquery.When(includeActors, e.RecordedBy(), e.AmendedBy())...,
	)
}

func eventsOf(
	req *workeremploymentports.ListWorkerEmploymentEventsRequest,
) keel.Query[*employment.WorkerEmploymentEvent] {
	e := employmentdb.WorkerEmploymentEvents

	return e.Where(
		e.WorkerID.Eq(req.WorkerID),
		keel.When(len(req.Kinds) > 0, e.Kind.In(req.Kinds...)),
	)
}

func listQuery(
	req *workeremploymentports.ListWorkerEmploymentEventsRequest,
) keel.Query[*employment.WorkerEmploymentEvent] {
	e := employmentdb.WorkerEmploymentEvents
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}

	return eventsOf(req).
		Load(eventLoads(req.IncludeDocument, req.IncludeActors)...).
		Order(e.EffectiveAt.Desc(), e.CreatedAt.Desc()).
		Limit(limit)
}

func timelineKeyset() *listquery.Keyset {
	e := employmentdb.WorkerEmploymentEvents

	return &listquery.Keyset{
		Orders: []keel.Order{e.EffectiveAt.Desc(), e.CreatedAt.Desc(), e.ID.Desc()},
		Shape: []pagination.CursorSortField{
			{Field: "effectiveAt", Direction: string(dbtype.SortDirectionDesc)},
			{Field: "createdAt", Direction: string(dbtype.SortDirectionDesc)},
			{Field: "id", Direction: string(dbtype.SortDirectionDesc)},
		},
	}
}

func (r *repository) ListPage(
	ctx context.Context,
	req *workeremploymentports.ListWorkerEmploymentEventsRequest,
) (*pagination.CursorListResult[*employment.WorkerEmploymentEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(
			ctx context.Context,
		) (*pagination.CursorListResult[*employment.WorkerEmploymentEvent], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				eventsOf(req),
				&listquery.Request[employment.WorkerEmploymentEvent]{
					Filter: &pagination.QueryOptions{TenantInfo: req.TenantInfo},
					Cursor: req.Cursor,
					Shape: func(
						q keel.Query[*employment.WorkerEmploymentEvent],
					) keel.Query[*employment.WorkerEmploymentEvent] {
						return q.Load(eventLoads(req.IncludeDocument, req.IncludeActors)...)
					},
					Keyset: timelineKeyset(),
				},
			)
			if err != nil {
				return nil, fmt.Errorf("page worker employment events: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *workeremploymentports.ListWorkerEmploymentEventsRequest,
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

func (r *repository) Summarize(
	ctx context.Context,
	req *workeremploymentports.SummarizeWorkerEmploymentEventsRequest,
) (*workeremploymentports.EmploymentEventSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workeremploymentports.EmploymentEventSummary, error) {
			e := employmentdb.WorkerEmploymentEvents
			summary, err := keel.IntoOne[workeremploymentports.EmploymentEventSummary](
				ctx,
				r.db.Keel(ctx),
				e.Pick(
					keel.Count(),
					e.EffectiveAt.Max().Filter(e.Kind.In(
						employment.EmploymentEventHired,
						employment.EmploymentEventRehired,
					)),
				).Where(e.WorkerID.Eq(req.WorkerID)),
			)
			if err != nil {
				return nil, fmt.Errorf("summarize worker employment events: %w", err)
			}

			return &summary, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workeremploymentports.GetWorkerEmploymentEventByIDRequest,
) (*employment.WorkerEmploymentEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*employment.WorkerEmploymentEvent, error) {
			e := employmentdb.WorkerEmploymentEvents
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
			if err := employmentdb.WorkerEmploymentEvents.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
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
			if err := employmentdb.WorkerEmploymentEvents.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update worker employment event", zap.Error(err))
				return nil, eventErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
