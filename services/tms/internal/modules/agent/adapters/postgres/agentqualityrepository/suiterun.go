package agentqualityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality/agentqualitydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	suiteRunEntity         = "AgentSuiteRun"
	defaultPageLimit       = 25
	maxPageLimit           = 101
	defaultHistoryPerAgent = 30
	maxHistoryPerAgent     = 100
	maxLatestAgents        = 1000
	constraintOneRunning   = "uq_agent_suite_runs_one_running"
	constraintSweepKey     = "uq_agent_suite_runs_sweep_key"
	historyCTE             = "suite_history"
)

var suiteRunErrors = dberror.Mapping{Entity: suiteRunEntity}

type SuiteRunParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type suiteRunRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSuiteRuns(p SuiteRunParams) agentqualityports.AgentSuiteRunRepository {
	return &suiteRunRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentsuiterun-repository"),
	}
}

func clampLimit(limit, fallback, ceiling int) int {
	if limit <= 0 {
		return fallback
	}

	return min(limit, ceiling)
}

func (r *suiteRunRepository) Create(
	ctx context.Context,
	entity *agentqualitysuite.SuiteRun,
) (*agentqualitysuite.SuiteRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentqualitysuite.SuiteRun, error) {
		if err := agentqualitydb.SuiteRuns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			switch dberror.ExtractConstraintName(err) {
			case constraintOneRunning:
				return nil, agentqualityports.ErrSuiteRunAlreadyRunning
			case constraintSweepKey:
				return nil, agentqualityports.ErrSuiteRunSweepKeyTaken
			}
			r.l.Error("failed to create agent suite run", zap.Error(err))

			return nil, fmt.Errorf("create agent suite run: %w", err)
		}

		return entity, nil
	})
}

func (r *suiteRunRepository) Update(
	ctx context.Context,
	entity *agentqualitysuite.SuiteRun,
) (*agentqualitysuite.SuiteRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentqualitysuite.SuiteRun, error) {
		s := agentqualitydb.SuiteRuns
		err := s.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			s.Status,
			s.Fingerprint,
			s.FingerprintHash,
			s.FingerprintChanges,
			s.CasesTotal,
			s.CasesPassed,
			s.CasesFailed,
			s.CasesSkipped,
			s.HardFailures,
			s.DeterministicScore,
			s.JudgeScore,
			s.QualityScore,
			s.BaselineScore,
			s.BaselineRunID,
			s.Regression,
			s.CostUSD,
			s.FinishedAt,
			s.Comments,
			s.WorkflowID,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, suiteRunErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent suite run: %w", err)
		}

		return entity, nil
	})
}

func (r *suiteRunRepository) GetByID(
	ctx context.Context,
	req agentqualityports.GetAgentSuiteRunRequest,
) (*agentqualitysuite.SuiteRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentqualitysuite.SuiteRun, error) {
		entity, err := agentqualitydb.SuiteRuns.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, suiteRunErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *suiteRunRepository) GetBySweepKey(
	ctx context.Context,
	req agentqualityports.GetAgentSuiteRunBySweepKeyRequest,
) (*agentqualitysuite.SuiteRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentqualitysuite.SuiteRun, error) {
		s := agentqualitydb.SuiteRuns
		entity, err := s.Where(s.SweepKey.Eq(req.SweepKey)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, suiteRunErrors.Map(err, req.SweepKey)
		}

		return entity, nil
	})
}

func (r *suiteRunRepository) Last(
	ctx context.Context,
	req agentqualityports.LastAgentSuiteRunRequest,
) (*agentqualitysuite.SuiteRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentqualitysuite.SuiteRun, error) {
		s := agentqualitydb.SuiteRuns
		entity, err := s.Where(
			s.AgentDefinitionID.Eq(req.AgentDefinitionID),
			keel.When(len(req.Statuses) > 0, s.Status.In(req.Statuses...)),
			keel.When(req.Before > 0, s.StartedAt.Lte(req.Before)),
			keel.When(req.ExcludeID.IsNotNil(), s.ID.Ne(req.ExcludeID)),
		).
			Order(s.StartedAt.Desc(), s.ID.Desc()).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read the last agent suite run: %w", err)
		}

		return entity, nil
	})
}

func (r *suiteRunRepository) ListConnection(
	ctx context.Context,
	req *agentqualityports.ListAgentSuiteRunConnectionRequest,
) (*pagination.CursorListResult[*agentqualitysuite.SuiteRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentqualitysuite.SuiteRun], error) {
			s := agentqualitydb.SuiteRuns
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				s.Where(keel.When(
					req.AgentDefinitionID.IsNotNil(),
					s.AgentDefinitionID.Eq(req.AgentDefinitionID),
				)),
				&listquery.Request[agentqualitysuite.SuiteRun]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent suite runs", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func historyQuery(
	req *agentqualityports.AgentSuiteRunHistoryRequest,
	perAgent int,
) *keel.Query[*agentqualitysuite.SuiteRun] {
	s := agentqualitydb.SuiteRuns
	ranked := agentqualitydb.SuiteRuns.As("ranked")
	history := keel.CTE(historyCTE, []string{"id", "history_rank"}, ranked.
		Pick(
			ranked.ID,
			keel.RowNumber().Over(
				[]keel.Expr{ranked.AgentDefinitionID},
				ranked.StartedAt.Desc(),
				ranked.ID.Desc(),
			),
		).
		Where(
			ranked.AgentDefinitionID.In(req.AgentDefinitionIDs...),
			ranked.Status.In(scoredStatuses()...),
			ranked.QualityScore.IsNotNull(),
			ranked.StartedAt.Gte(req.Since),
		))

	return s.With(history).
		Join(history, history.Col("id").EqCol(s.ID)).
		Where(history.Col("history_rank").Lte(perAgent)).
		Order(s.AgentDefinitionID.Asc(), s.StartedAt.Asc(), s.ID.Asc())
}

func (r *suiteRunRepository) History(
	ctx context.Context,
	req agentqualityports.AgentSuiteRunHistoryRequest,
) ([]*agentqualitysuite.SuiteRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentqualitysuite.SuiteRun, error) {
		if len(req.AgentDefinitionIDs) == 0 {
			return []*agentqualitysuite.SuiteRun{}, nil
		}

		perAgent := clampLimit(req.PerAgent, defaultHistoryPerAgent, maxHistoryPerAgent)
		runs, err := historyQuery(&req, perAgent).
			Limit(len(req.AgentDefinitionIDs)*perAgent).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent quality history: %w", err)
		}

		return runs, nil
	})
}

func scoredStatuses() []agentqualitysuite.SuiteRunStatus {
	return []agentqualitysuite.SuiteRunStatus{
		agentqualitysuite.SuiteRunStatusCompleted,
		agentqualitysuite.SuiteRunStatusBudgetStopped,
	}
}

func (r *suiteRunRepository) Latest(
	ctx context.Context,
	req agentqualityports.LatestAgentSuiteRunsRequest,
) ([]*agentqualitysuite.SuiteRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentqualitysuite.SuiteRun, error) {
		s := agentqualitydb.SuiteRuns
		runs, err := s.DistinctOn(s.AgentDefinitionID).
			Where(
				keel.When(
					len(req.AgentDefinitionIDs) > 0,
					s.AgentDefinitionID.In(req.AgentDefinitionIDs...),
				),
				keel.When(len(req.Statuses) > 0, s.Status.In(req.Statuses...)),
			).
			Order(s.AgentDefinitionID.Asc(), s.StartedAt.Desc(), s.ID.Desc()).
			Limit(clampLimit(req.Limit, maxLatestAgents, maxLatestAgents)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read each agent's latest suite run: %w", err)
		}

		return runs, nil
	})
}

func (r *suiteRunRepository) Overview(
	ctx context.Context,
	req agentqualityports.AgentSuiteRunOverviewRequest,
) (*agentqualityports.AgentSuiteRunOverview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentqualityports.AgentSuiteRunOverview, error) {
			s := agentqualitydb.SuiteRuns
			overview, err := keel.IntoOne[agentqualityports.AgentSuiteRunOverview](
				ctx,
				r.db.Keel(ctx),
				s.Pick(
					keel.Count().Filter(s.Status.In(scoredStatuses()...)),
					keel.Count().Filter(s.Regression.IsTrue()),
				).Where(s.StartedAt.Gte(req.Since)),
			)
			if err != nil {
				return nil, fmt.Errorf("summarize agent suite runs: %w", err)
			}

			return &overview, nil
		},
	)
}

func (r *suiteRunRepository) ListAgents(
	ctx context.Context,
	req agentqualityports.ListAgentQualityAgentsQuery,
) (*agentqualityports.AgentQualityAgentRows, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentqualityports.AgentQualityAgentRows, error) {
			d := agentdefinitiondb.Definitions
			limit := clampLimit(req.Limit, defaultPageLimit, maxPageLimit-1)

			var (
				rows  []agentqualityports.AgentQualityAgentRow
				total keel.RowCount
			)
			items := []keel.BatchItem{
				keel.IntoSlice(d.
					Pick(d.ID, d.Name, d.Enabled).
					Order(d.Name.Asc(), d.ID.Asc()).
					Limit(limit+1).
					Offset(max(req.Offset, 0)), &rows),
			}
			if req.IncludeTotalCount {
				items = append(
					items,
					keel.CountEstimateInto(d.Query(), pagination.ExactTotalUpTo, &total),
				)
			}
			if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
				return nil, fmt.Errorf("list agents for quality: %w", err)
			}

			result := &agentqualityports.AgentQualityAgentRows{}
			if req.IncludeTotalCount {
				result.TotalCount = new(int(total.N))
				result.TotalCountExact = total.Exact
			}
			if len(rows) > limit {
				result.HasNextPage = true
				rows = rows[:limit]
			}
			result.Items = make([]*agentqualityports.AgentQualityAgentRow, 0, len(rows))
			for i := range rows {
				result.Items = append(result.Items, &rows[i])
			}

			return result, nil
		},
	)
}
