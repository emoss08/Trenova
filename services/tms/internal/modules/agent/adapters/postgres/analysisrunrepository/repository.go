package analysisrunrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis/analysisdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName  = "Analysis"
	defaultRuns = 50
	maxRuns     = 200
)

var runErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) analysisports.RunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-analysis-run-repository"),
	}
}

func (r *repository) Create(ctx context.Context, run *analysis.Run) (*analysis.Run, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*analysis.Run, error) {
		if err := analysisdb.Runs.Create(ctx, r.db.Keel(ctx), run); err != nil {
			r.l.Error("failed to create analysis run", zap.Error(err))
			return nil, fmt.Errorf("create analysis run: %w", err)
		}

		return run, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*analysis.Run, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*analysis.Run, error) {
		run, err := analysisdb.Runs.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, runErrors.Map(err, id.String())
		}

		return run, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req *analysisports.ListRunsRequest,
) ([]*analysis.Run, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultRuns
	}
	limit = min(limit, maxRuns)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*analysis.Run, error) {
		a := analysisdb.Runs
		runs, err := a.Pick(
			a.ID, a.Status, a.Title, a.Purpose, a.DurationMs, a.AgentDefinitionID,
			a.AgentRunID, a.ThreadID, a.RequestedByID, a.StartedAt, a.FinishedAt,
		).Where(keel.When(req.Before > 0, keel.Or(
			a.StartedAt.Lt(req.Before),
			keel.And(a.StartedAt.Eq(req.Before), a.ID.Lt(req.BeforeID)),
		))).
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
