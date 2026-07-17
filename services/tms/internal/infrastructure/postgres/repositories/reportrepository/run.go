package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RunParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type runRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRunRepository(p RunParams) repositories.ReportRunRepository {
	return &runRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-run-repository"),
	}
}

func (r *runRepository) Create(
	ctx context.Context,
	entity *report.ReportRun,
) (*report.ReportRun, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to create report run", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *runRepository) Update(
	ctx context.Context,
	entity *report.ReportRun,
) (*report.ReportRun, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ReportRunColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update report run", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "ReportRun", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *runRepository) GetByID(
	ctx context.Context,
	req *repositories.GetReportRunRequest,
) (*report.ReportRun, error) {
	entity := new(report.ReportRun)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReportRunScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReportRunColumns.ID.Eq(), req.RunID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReportRun")
	}

	return entity, nil
}

func (r *runRepository) List(
	ctx context.Context,
	req *repositories.ListReportRunsRequest,
) ([]*report.ReportRun, error) {
	cols := buncolgen.ReportRunColumns

	entities := make([]*report.ReportRun, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.ReportRunApplyTenant(req.TenantInfo))

	if !req.DefinitionID.IsNil() {
		q = q.Where(cols.DefinitionID.Eq(), req.DefinitionID)
	}
	if !req.RequestedBy.IsNil() {
		q = q.Where(cols.RequestedByID.Eq(), req.RequestedBy)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	if err := q.Order(cols.CreatedAt.OrderDesc()).Scan(ctx); err != nil {
		r.l.Error("failed to list report runs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *runRepository) CountActive(
	ctx context.Context,
	req *repositories.CountActiveReportRunsRequest,
) (*repositories.ActiveReportRunCounts, error) {
	cols := buncolgen.ReportRunColumns

	var counts repositories.ActiveReportRunCounts
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*report.ReportRun)(nil)).
		ColumnExpr(
			buncolgen.CountFilter("running", cols.Status.Eq()),
			report.RunStatusRunning,
		).
		ColumnExpr(
			buncolgen.CountFilter("queued", cols.Status.Eq()),
			report.RunStatusQueued,
		).
		Apply(buncolgen.ReportRunApplyTenant(req.TenantInfo)).
		Scan(ctx, &counts.Running, &counts.Queued)
	if err != nil {
		r.l.Error("failed to count active report runs", zap.Error(err))
		return nil, err
	}

	return &counts, nil
}

func (r *runRepository) ListStale(
	ctx context.Context,
	req *repositories.ListStaleReportRunsRequest,
) ([]*report.ReportRun, error) {
	cols := buncolgen.ReportRunColumns

	entities := make([]*report.ReportRun, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.List(req.Statuses)).
		Where(cols.UpdatedAt.Lt(), req.UpdatedBeforeUnix).
		Order(cols.UpdatedAt.OrderAsc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list stale report runs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *runRepository) ListExpired(
	ctx context.Context,
	req *repositories.ListExpiredReportRunsRequest,
) ([]*report.ReportRun, error) {
	cols := buncolgen.ReportRunColumns

	entities := make([]*report.ReportRun, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(cols.Status.Eq(), report.RunStatusSucceeded).
		Where(cols.ArtifactExpiresAt.IsNotNull()).
		Where(cols.ArtifactExpiresAt.Lt(), req.CutoffUnix).
		Order(cols.ArtifactExpiresAt.OrderAsc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list expired report runs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
