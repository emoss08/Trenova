package reportrepository

import (
	"context"

	"github.com/bytedance/sonic"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var runErrors = dberror.Mapping{Entity: "ReportRun"}

type activeRunCounts struct {
	Running int64
	Queued  int64
}

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
		if err := reportdb.ReportRuns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
		if err := reportdb.ReportRuns.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update report run",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *runRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportRunRequest,
) (*report.ReportRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportRun, error) {
		run := reportdb.ReportRuns
		entity, err := run.Where(
			run.ID.Eq(req.RunID),
			keel.When(!req.RequestedBy.IsNil(), run.RequestedByID.Eq(req.RequestedBy)),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, runErrors.Map(err, req.RunID.String())
		}

		return entity, nil
	})
}

func (r *runRepository) LatestComparable(
	ctx context.Context,
	req *reportports.GetComparableReportRunRequest,
) (*report.ReportRun, error) {
	params, err := comparableParams(req.Params)
	if err != nil {
		return nil, err
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportRun, error) {
		run := reportdb.ReportRuns
		entity, firstErr := run.Pick(run.ID, run.CompletedAt, run.RowCount, run.Sample).
			Where(
				run.RequestedByID.Eq(req.RequestedBy),
				run.Status.Eq(report.RunStatusSucceeded),
				run.Sample.IsNotNull(),
				keel.Raw("COALESCE(?, '{}'::jsonb) = ?::jsonb", run.Params, params),
				run.ID.Ne(req.ExcludeRunID),
				run.CompletedAt.Lt(req.CompletedBefore),
				keel.When(req.DefinitionID.IsNil(), run.DefinitionID.IsNull()),
				keel.When(req.DefinitionID.IsNil(), run.CannedKey.Eq(req.CannedKey)),
				keel.When(!req.DefinitionID.IsNil(), run.DefinitionID.Eq(req.DefinitionID)),
				keel.When(!req.DefinitionID.IsNil(), run.CannedKey.IsNull()),
			).
			Order(run.CompletedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if firstErr != nil {
			return nil, runErrors.Map(firstErr, req.ExcludeRunID.String())
		}

		return entity, nil
	})
}

func comparableParams(params map[string]any) (string, error) {
	if len(params) == 0 {
		return "{}", nil
	}

	return sonic.MarshalString(params)
}

func (r *runRepository) CountActive(
	ctx context.Context,
	_ *reportports.CountActiveReportRunsRequest,
) (*reportports.ActiveReportRunCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*reportports.ActiveReportRunCounts, error) {
			run := reportdb.ReportRuns
			counts, err := keel.IntoOne[activeRunCounts](
				ctx,
				r.db.Keel(ctx),
				run.Pick(
					keel.Count().Filter(run.Status.Eq(report.RunStatusRunning)),
					keel.Count().Filter(run.Status.Eq(report.RunStatusQueued)),
				),
			)
			if err != nil {
				r.l.Error("failed to count active report runs", zap.Error(err))
				return nil, err
			}

			return &reportports.ActiveReportRunCounts{
				Running: int(counts.Running),
				Queued:  int(counts.Queued),
			}, nil
		},
	)
}

func (r *runRepository) ListStale(
	ctx context.Context,
	req *reportports.ListStaleReportRunsRequest,
) ([]*report.ReportRun, error) {
	ctx = dbscope.WithSystem(ctx, "list stalled report runs across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportRun, error) {
		run := reportdb.ReportRuns
		q := run.AllTenants().
			Omit(run.Sample).
			Where(run.Status.In(req.Statuses...), run.UpdatedAt.Lt(req.UpdatedBeforeUnix)).
			Order(run.UpdatedAt.Asc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
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
		run := reportdb.ReportRuns
		q := run.AllTenants().
			Omit(run.Sample).
			Where(
				run.Status.Eq(report.RunStatusSucceeded),
				run.ArtifactExpiresAt.IsNotNull(),
				run.ArtifactExpiresAt.Lt(req.CutoffUnix),
			).
			Order(run.ArtifactExpiresAt.Asc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list expired report runs", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
