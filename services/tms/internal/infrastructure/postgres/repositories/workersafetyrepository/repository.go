package workersafetyrepository

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

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.WorkerSafetyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-safety-repository"),
	}
}

func (r *repository) ListEvents(
	ctx context.Context,
	req *repositories.ListWorkerSafetyEventsRequest,
) ([]*worker.WorkerSafetyEvent, error) {
	cols := buncolgen.WorkerSafetyEventColumns
	entities := make([]*worker.WorkerSafetyEvent, 0, 16)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerSafetyEventScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if req.Since > 0 {
				sq = sq.Where(cols.OccurredAt.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.OccurredAt.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc())
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerSafetyEventRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerSafetyEventRelations.RecordedBy).
			Relation(buncolgen.WorkerSafetyEventRelations.ClosedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker safety events", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetEventByID(
	ctx context.Context,
	req *repositories.GetWorkerSafetyEventByIDRequest,
) (*worker.WorkerSafetyEvent, error) {
	entity := new(worker.WorkerSafetyEvent)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerSafetyEventScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerSafetyEventColumns.ID.Eq(), req.ID)
		})
	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerSafetyEventRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerSafetyEventRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerSafetyEvent")
	}

	return entity, nil
}

func (r *repository) CreateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
) (*worker.WorkerSafetyEvent, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create worker safety event", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateEvent(
	ctx context.Context,
	entity *worker.WorkerSafetyEvent,
) (*worker.WorkerSafetyEvent, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerSafetyEventColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update worker safety event", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerSafetyEvent", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteEvent(
	ctx context.Context,
	req *repositories.GetWorkerSafetyEventByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerSafetyEvent)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerSafetyEventScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerSafetyEventColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete worker safety event", zap.Error(err))
		return err
	}
	return dberror.CheckRowsAffected(results, "WorkerSafetyEvent", req.ID.String())
}

func (r *repository) ListActions(
	ctx context.Context,
	req *repositories.ListWorkerDisciplinaryActionsRequest,
) ([]*worker.WorkerDisciplinaryAction, error) {
	cols := buncolgen.WorkerDisciplinaryActionColumns
	entities := make([]*worker.WorkerDisciplinaryAction, 0, 8)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerDisciplinaryActionScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
		}).
		Order(cols.IssuedAt.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc())
	if req.IncludeEvent {
		q = q.Relation(buncolgen.WorkerDisciplinaryActionRelations.SafetyEvent)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerDisciplinaryActionRelations.IssuedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list disciplinary actions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetActionByID(
	ctx context.Context,
	req *repositories.GetWorkerDisciplinaryActionByIDRequest,
) (*worker.WorkerDisciplinaryAction, error) {
	entity := new(worker.WorkerDisciplinaryAction)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerDisciplinaryActionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerDisciplinaryActionColumns.ID.Eq(), req.ID)
		})
	if req.IncludeEvent {
		q = q.Relation(buncolgen.WorkerDisciplinaryActionRelations.SafetyEvent)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerDisciplinaryAction")
	}

	return entity, nil
}

func (r *repository) CreateAction(
	ctx context.Context,
	entity *worker.WorkerDisciplinaryAction,
) (*worker.WorkerDisciplinaryAction, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create disciplinary action", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateAction(
	ctx context.Context,
	entity *worker.WorkerDisciplinaryAction,
) (*worker.WorkerDisciplinaryAction, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerDisciplinaryActionColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update disciplinary action", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerDisciplinaryAction", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListRecognitions(
	ctx context.Context,
	req *repositories.ListWorkerRecognitionsRequest,
) ([]*worker.WorkerRecognition, error) {
	cols := buncolgen.WorkerRecognitionColumns
	entities := make([]*worker.WorkerRecognition, 0, 8)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerRecognitionScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if req.VisibleOnly {
				sq = sq.Where(cols.VisibleToWorker.Eq(), true)
			}
			return sq
		}).
		Order(cols.OccurredAt.OrderDesc())
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerRecognitionRelations.AwardedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list recognitions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetRecognitionByID(
	ctx context.Context,
	req *repositories.GetWorkerRecognitionByIDRequest,
) (*worker.WorkerRecognition, error) {
	entity := new(worker.WorkerRecognition)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerRecognitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerRecognitionColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerRecognition")
	}

	return entity, nil
}

func (r *repository) CreateRecognition(
	ctx context.Context,
	entity *worker.WorkerRecognition,
) (*worker.WorkerRecognition, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create recognition", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteRecognition(
	ctx context.Context,
	req *repositories.GetWorkerRecognitionByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerRecognition)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerRecognitionScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerRecognitionColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete recognition", zap.Error(err))
		return err
	}
	return dberror.CheckRowsAffected(results, "WorkerRecognition", req.ID.String())
}

// ListWorkersWithLapsedPoints returns one row per worker whose points expired
// inside the window, plus anyone whose disciplinary action lapsed there. Both
// change the scorecard on a date rather than on a write, so the nightly sweep
// is the only thing that will notice.
func (r *repository) ListWorkersWithLapsedPoints(
	ctx context.Context,
	req *repositories.ListWorkersWithLapsedPointsRequest,
) ([]repositories.WorkerTenantRef, error) {
	ecols := buncolgen.WorkerSafetyEventColumns
	acols := buncolgen.WorkerDisciplinaryActionColumns

	events := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerSafetyEvent)(nil)).
		ColumnExpr("? AS worker_id", bun.Ident(ecols.WorkerID.String())).
		ColumnExpr("? AS organization_id", bun.Ident(ecols.OrganizationID.String())).
		ColumnExpr("? AS business_unit_id", bun.Ident(ecols.BusinessUnitID.String())).
		Where(ecols.PointsExpireAt.Gte(), req.Since).
		Where(ecols.PointsExpireAt.Lt(), req.Until)

	actions := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerDisciplinaryAction)(nil)).
		ColumnExpr("? AS worker_id", bun.Ident(acols.WorkerID.String())).
		ColumnExpr("? AS organization_id", bun.Ident(acols.OrganizationID.String())).
		ColumnExpr("? AS business_unit_id", bun.Ident(acols.BusinessUnitID.String())).
		Where(acols.ExpiresAt.Gte(), req.Since).
		Where(acols.ExpiresAt.Lt(), req.Until)

	refs := make([]repositories.WorkerTenantRef, 0, 32)
	q := r.db.DBForContext(ctx).
		NewSelect().
		With("lapsed", events.UnionAll(actions)).
		Table("lapsed").
		ColumnExpr("DISTINCT worker_id, organization_id, business_unit_id").
		OrderExpr("worker_id")
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if err := q.Scan(ctx, &refs); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Worker")
	}
	return refs, nil
}
