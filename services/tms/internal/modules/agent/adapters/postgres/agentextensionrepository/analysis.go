package agentextensionrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	analysisRunEntityName   = "Analysis"
	analysisChartEntityName = "Analysis chart"
	defaultAnalysisRuns     = 50
	maxAnalysisRuns         = 200
)

var analysisRunErrors = dberror.Mapping{Entity: analysisRunEntityName}

type analysisRunRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAnalysisRunRepository(p Params) agentextensionports.AnalysisRunRepository {
	return &analysisRunRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-analysis-run-repository"),
	}
}

func (r *analysisRunRepository) Create(
	ctx context.Context,
	run *agentextension.AnalysisRun,
) (*agentextension.AnalysisRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisRun, error) {
		if err := agentextensiondb.AnalysisRuns.Create(ctx, r.db.Keel(ctx), run); err != nil {
			r.l.Error("failed to create analysis run", zap.Error(err))
			return nil, fmt.Errorf("create analysis run: %w", err)
		}

		return run, nil
	})
}

func (r *analysisRunRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.AnalysisRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisRun, error) {
		run, err := agentextensiondb.AnalysisRuns.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, analysisRunErrors.Map(err, id.String())
		}

		return run, nil
	})
}

func (r *analysisRunRepository) Update(
	ctx context.Context,
	run *agentextension.AnalysisRun,
) (*agentextension.AnalysisRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisRun, error) {
		if err := agentextensiondb.AnalysisRuns.Save(ctx, r.db.Keel(ctx), run); err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, analysisRunErrors.Map(err, run.ID.String())
			}
			r.l.Error("failed to update analysis run", zap.Error(err))
			return nil, fmt.Errorf("update analysis run: %w", err)
		}

		return run, nil
	})
}

func (r *analysisRunRepository) List(
	ctx context.Context,
	req *agentextensionports.ListAnalysisRunsRequest,
) ([]*agentextension.AnalysisRun, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultAnalysisRuns
	}
	limit = min(limit, maxAnalysisRuns)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.AnalysisRun, error) {
		a := agentextensiondb.AnalysisRuns
		runs, err := a.Where(keel.When(req.Before > 0, a.StartedAt.Lt(req.Before))).
			Order(a.StartedAt.Desc(), a.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list analysis runs", zap.Error(err))
			return nil, fmt.Errorf("list analysis runs: %w", err)
		}

		return runs, nil
	})
}

func (r *analysisRunRepository) SaveCharts(
	ctx context.Context,
	charts []*agentextension.AnalysisChart,
) error {
	if len(charts) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := agentextensiondb.AnalysisCharts.CreateMany(
			ctx,
			r.db.Keel(ctx),
			charts,
		); err != nil {
			r.l.Error("failed to save analysis charts", zap.Error(err))
			return fmt.Errorf("save analysis charts: %w", err)
		}

		return nil
	})
}

func (r *analysisRunRepository) GetChart(
	ctx context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
	position int,
) (*agentextension.AnalysisChart, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisChart, error) {
		c := agentextensiondb.AnalysisCharts
		chart, err := c.Where(c.RunID.Eq(runID), c.Position.Eq(int64(position))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err,
				analysisChartEntityName+" "+strconv.Itoa(position))
		}

		return chart, nil
	})
}
