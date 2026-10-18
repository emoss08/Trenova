package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/reportcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
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

func NewRunRepository(p RunParams) reportports.ReportRunRepository {
	return &runRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-run-repository"),
	}
}

func (r *runRepository) Create(
	ctx context.Context,
	entity *report.ReportRun,
) (*report.ReportRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportRun, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			r.l.Error("failed to create report run", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *runRepository) Update(
	ctx context.Context,
	entity *report.ReportRun,
) (*report.ReportRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportRun, error) {
		log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

		ov := entity.Version
		entity.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(reportcol.ReportRunColumns.Version.Eq(), ov).
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
	})
}

func (r *runRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportRunRequest,
) (*report.ReportRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportRun, error) {
		entity := new(report.ReportRun)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = reportcol.ReportRunScopeTenant(sq, req.TenantInfo).
					Where(reportcol.ReportRunColumns.ID.Eq(), req.RunID)
				if !req.RequestedBy.IsNil() {
					sq = sq.Where(reportcol.ReportRunColumns.RequestedByID.Eq(), req.RequestedBy)
				}
				return sq
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "ReportRun")
		}

		return entity, nil
	})
}

func (r *runRepository) List(
	ctx context.Context,
	req *reportports.ListReportRunsRequest,
) ([]*report.ReportRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportRun, error) {
		cols := reportcol.ReportRunColumns

		entities := make([]*report.ReportRun, 0)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(reportcol.ReportRunApplyTenant(req.TenantInfo))

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
			q = q.Limit(int64(req.Limit))
		}
		if req.Offset > 0 {
			q = q.Offset(int64(req.Offset))
		}

		if err := q.Order(cols.CreatedAt.OrderDesc()).Scan(ctx); err != nil {
			r.l.Error("failed to list report runs", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *runRepository) CountActive(
	ctx context.Context,
	req *reportports.CountActiveReportRunsRequest,
) (*reportports.ActiveReportRunCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*reportports.ActiveReportRunCounts, error) {
			cols := reportcol.ReportRunColumns

			var counts reportports.ActiveReportRunCounts
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
				Apply(reportcol.ReportRunApplyTenant(req.TenantInfo)).
				Scan(ctx, &counts.Running, &counts.Queued)
			if err != nil {
				r.l.Error("failed to count active report runs", zap.Error(err))
				return nil, err
			}

			return &counts, nil
		},
	)
}

func (r *runRepository) ListStale(
	ctx context.Context,
	req *reportports.ListStaleReportRunsRequest,
) ([]*report.ReportRun, error) {
	ctx = dbscope.WithSystem(ctx, "list stalled report runs across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportRun, error) {
		cols := reportcol.ReportRunColumns

		entities := make([]*report.ReportRun, 0)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(cols.Status.In(), bun.List(req.Statuses)).
			Where(cols.UpdatedAt.Lt(), req.UpdatedBeforeUnix).
			Order(cols.UpdatedAt.OrderAsc())

		if req.Limit > 0 {
			q = q.Limit(int64(req.Limit))
		}

		if err := q.Scan(ctx); err != nil {
			r.l.Error("failed to list stale report runs", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *runRepository) ListExpired(
	ctx context.Context,
	req *reportports.ListExpiredReportRunsRequest,
) ([]*report.ReportRun, error) {
	ctx = dbscope.WithSystem(ctx, "list expired report artifacts across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportRun, error) {
		cols := reportcol.ReportRunColumns

		entities := make([]*report.ReportRun, 0)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(cols.Status.Eq(), report.RunStatusSucceeded).
			Where(cols.ArtifactExpiresAt.IsNotNull()).
			Where(cols.ArtifactExpiresAt.Lt(), req.CutoffUnix).
			Order(cols.ArtifactExpiresAt.OrderAsc())

		if req.Limit > 0 {
			q = q.Limit(int64(req.Limit))
		}

		if err := q.Scan(ctx); err != nil {
			r.l.Error("failed to list expired report runs", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
