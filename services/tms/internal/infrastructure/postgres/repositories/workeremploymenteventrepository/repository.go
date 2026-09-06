package workeremploymenteventrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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

func New(p Params) repositories.WorkerEmploymentEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-employment-event-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListWorkerEmploymentEventsRequest,
) ([]*worker.WorkerEmploymentEvent, error) {
	cols := buncolgen.WorkerEmploymentEventColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}

	entities := make([]*worker.WorkerEmploymentEvent, 0, 16)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerEmploymentEventScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if len(req.Kinds) > 0 {
				sq = sq.Where(cols.Kind.In(), bun.In(req.Kinds))
			}
			return sq
		}).
		Limit(limit)
	if req.Ascending {
		q = q.Order(cols.EffectiveAt.OrderAsc()).Order(cols.CreatedAt.OrderAsc())
	} else {
		q = q.Order(cols.EffectiveAt.OrderDesc()).Order(cols.CreatedAt.OrderDesc())
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerEmploymentEventRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerEmploymentEventRelations.RecordedBy).
			Relation(buncolgen.WorkerEmploymentEventRelations.AmendedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker employment events", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerEmploymentEventByIDRequest,
) (*worker.WorkerEmploymentEvent, error) {
	entity := new(worker.WorkerEmploymentEvent)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerEmploymentEventScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerEmploymentEventColumns.ID.Eq(), req.ID)
		})
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerEmploymentEventRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerEmploymentEventRelations.RecordedBy).
			Relation(buncolgen.WorkerEmploymentEventRelations.AmendedBy)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerEmploymentEvent")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.WorkerEmploymentEvent,
) (*worker.WorkerEmploymentEvent, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create worker employment event", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerEmploymentEvent,
) (*worker.WorkerEmploymentEvent, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerEmploymentEventColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update worker employment event", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerEmploymentEvent", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
