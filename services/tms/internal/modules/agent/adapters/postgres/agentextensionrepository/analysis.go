package agentextensionrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentextensioncol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	analysisRunEntityName   = "Analysis"
	analysisChartEntityName = "Analysis chart"
	defaultAnalysisRuns     = 50
	maxAnalysisRuns         = 200
)

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
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(run).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to create analysis run", zap.Error(err))
			return nil, fmt.Errorf("create analysis run: %w", err)
		}

		return run, nil
	})
}

func (r *analysisRunRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.AnalysisRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisRun, error) {
		run := new(agentextension.AnalysisRun)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(run).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentextensioncol.AnalysisRunScopeTenant(sq, tenantInfo).
					Where(agentextensioncol.AnalysisRunColumns.ID.Eq(), id)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, analysisRunEntityName)
		}

		return run, nil
	})
}

func (r *analysisRunRepository) Update(
	ctx context.Context,
	run *agentextension.AnalysisRun,
) (*agentextension.AnalysisRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisRun, error) {
		ov := run.Version
		run.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(run).
			WherePK().
			Where(agentextensioncol.AnalysisRunColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update analysis run", zap.Error(err))
			return nil, fmt.Errorf("update analysis run: %w", err)
		}
		if err = dberror.CheckRowsAffected(result, analysisRunEntityName, run.ID.String()); err != nil {
			return nil, err
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
		runs := make([]*agentextension.AnalysisRun, 0, limit)
		cols := agentextensioncol.AnalysisRunColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&runs).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = agentextensioncol.AnalysisRunScopeTenant(sq, req.TenantInfo)
				if req.Before > 0 {
					sq = sq.Where(cols.StartedAt.Lt(), req.Before)
				}
				return sq
			}).
			Order(cols.StartedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(limit).
			Scan(ctx)
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
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&charts).
			Exec(ctx); err != nil {
			r.l.Error("failed to save analysis charts", zap.Error(err))
			return fmt.Errorf("save analysis charts: %w", err)
		}

		return nil
	})
}

func (r *analysisRunRepository) GetChart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
	position int,
) (*agentextension.AnalysisChart, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.AnalysisChart, error) {
		chart := new(agentextension.AnalysisChart)
		cols := agentextensioncol.AnalysisChartColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(chart).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentextensioncol.AnalysisChartScopeTenant(sq, tenantInfo).
					Where(cols.RunID.Eq(), runID).
					Where(cols.Position.Eq(), position)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err,
				analysisChartEntityName+" "+strconv.Itoa(position))
		}

		return chart, nil
	})
}
