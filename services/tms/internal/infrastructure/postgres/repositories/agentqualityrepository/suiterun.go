package agentqualityrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	suiteRunEntity          = "AgentSuiteRun"
	defaultPageLimit        = 25
	maxPageLimit            = 101
	defaultHistoryPerAgent  = 30
	maxHistoryPerAgent      = 100
	maxLatestAgents         = 1000
	constraintOneRunning    = "uq_agent_suite_runs_one_running"
	constraintSweepKey      = "uq_agent_suite_runs_sweep_key"
	historyRankColumnAlias  = "history_rank"
	historyDerivedTableName = "asr"
)

type SuiteRunParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type suiteRunRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSuiteRuns(p SuiteRunParams) repositories.AgentSuiteRunRepository {
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
	entity *agentquality.SuiteRun,
) (*agentquality.SuiteRun, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		switch dberror.ExtractConstraintName(err) {
		case constraintOneRunning:
			return nil, repositories.ErrSuiteRunAlreadyRunning
		case constraintSweepKey:
			return nil, repositories.ErrSuiteRunSweepKeyTaken
		}
		r.l.Error("failed to create agent suite run", zap.Error(err))

		return nil, fmt.Errorf("create agent suite run: %w", err)
	}

	return entity, nil
}

func (r *suiteRunRepository) Update(
	ctx context.Context,
	entity *agentquality.SuiteRun,
) (*agentquality.SuiteRun, error) {
	cols := buncolgen.SuiteRunColumns
	ov := entity.Version
	entity.Version++
	entity.UpdatedAt = timeutils.NowUnix()

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SuiteRunScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.Fingerprint.Set(), entity.Fingerprint).
		Set(cols.FingerprintHash.Set(), entity.FingerprintHash).
		Set(cols.FingerprintChanges.Set(), entity.FingerprintChanges).
		Set(cols.CasesTotal.Set(), entity.CasesTotal).
		Set(cols.CasesPassed.Set(), entity.CasesPassed).
		Set(cols.CasesFailed.Set(), entity.CasesFailed).
		Set(cols.CasesSkipped.Set(), entity.CasesSkipped).
		Set(cols.HardFailures.Set(), entity.HardFailures).
		Set(cols.DeterministicScore.Set(), entity.DeterministicScore).
		Set(cols.JudgeScore.Set(), entity.JudgeScore).
		Set(cols.QualityScore.Set(), entity.QualityScore).
		Set(cols.BaselineScore.Set(), entity.BaselineScore).
		Set(cols.BaselineRunID.Set(), entity.BaselineRunID).
		Set(cols.Regression.Set(), entity.Regression).
		Set(cols.CostUSD.Set(), entity.CostUSD).
		Set(cols.FinishedAt.Set(), entity.FinishedAt).
		Set(cols.Comments.Set(), bun.NullZero(entity.Comments)).
		Set(cols.WorkflowID.Set(), bun.NullZero(entity.WorkflowID)).
		Set(cols.UpdatedAt.Set(), entity.UpdatedAt).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent suite run: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update agent suite run rows: %w", err)
	}
	if rows == 0 {
		return nil, dberror.CreateVersionMismatchError(suiteRunEntity, entity.ID.String())
	}

	return entity, nil
}

func (r *suiteRunRepository) GetByID(
	ctx context.Context,
	req repositories.GetAgentSuiteRunRequest,
) (*agentquality.SuiteRun, error) {
	cols := buncolgen.SuiteRunColumns
	entity := new(agentquality.SuiteRun)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, suiteRunEntity)
	}

	return entity, nil
}

func (r *suiteRunRepository) GetBySweepKey(
	ctx context.Context,
	req repositories.GetAgentSuiteRunBySweepKeyRequest,
) (*agentquality.SuiteRun, error) {
	cols := buncolgen.SuiteRunColumns
	entity := new(agentquality.SuiteRun)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo).
				Where(cols.SweepKey.Eq(), req.SweepKey)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, suiteRunEntity)
	}

	return entity, nil
}

func (r *suiteRunRepository) Last(
	ctx context.Context,
	req repositories.LastAgentSuiteRunRequest,
) (*agentquality.SuiteRun, error) {
	cols := buncolgen.SuiteRunColumns
	entity := new(agentquality.SuiteRun)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}
			if req.Before > 0 {
				sq = sq.Where(cols.StartedAt.Lte(), req.Before)
			}
			if req.ExcludeID.IsNotNil() {
				sq = sq.Where(cols.ID.NotEq(), req.ExcludeID)
			}

			return sq
		}).
		Order(cols.StartedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the last agent suite run: %w", err)
	}

	return entity, nil
}

func (r *suiteRunRepository) List(
	ctx context.Context,
	req repositories.ListAgentSuiteRunsRequest,
) (*repositories.AgentSuiteRunPage, error) {
	cols := buncolgen.SuiteRunColumns
	limit := clampLimit(req.Limit, defaultPageLimit, maxPageLimit-1)
	dba := r.db.DBForContext(ctx)

	scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo)
		if req.AgentDefinitionID.IsNotNil() {
			sq = sq.Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
		}
		if len(req.Statuses) > 0 {
			sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
		}

		return sq
	}

	page := &repositories.AgentSuiteRunPage{}
	if req.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*agentquality.SuiteRun)(nil)).
			WhereGroup(" AND ", scope).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count agent suite runs: %w", err)
		}
		page.TotalCount = &total
	}

	items := make([]*agentquality.SuiteRun, 0, limit+1)
	if err := dba.NewSelect().
		Model(&items).
		WhereGroup(" AND ", scope).
		Order(cols.StartedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit + 1).
		Offset(max(req.Offset, 0)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list agent suite runs: %w", err)
	}

	if len(items) > limit {
		page.HasNextPage = true
		items = items[:limit]
	}
	page.Items = items

	return page, nil
}

func (r *suiteRunRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentSuiteRunConnectionRequest,
) (*pagination.CursorListResult[*agentquality.SuiteRun], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*agentquality.SuiteRun)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.SuiteRunTable.Alias,
					req.Filter,
					(*agentquality.SuiteRun)(nil),
				)

				return suiteRunScope(sq, req).
					Apply(buncolgen.SuiteRunApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count agent suite runs", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agentquality.SuiteRun]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*agentquality.SuiteRun) *bun.SelectQuery {
				return dba.NewSelect().Model(entities)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					suiteRunScope(sq, req),
					buncolgen.SuiteRunTable.Alias,
					req.Filter,
					req.Cursor,
					(*agentquality.SuiteRun)(nil),
				)
			},
		})
	if err != nil {
		log.Error("failed to list agent suite runs", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func suiteRunScope(
	sq *bun.SelectQuery,
	req *repositories.ListAgentSuiteRunConnectionRequest,
) *bun.SelectQuery {
	if req.AgentDefinitionID.IsNil() {
		return sq
	}

	return sq.Where(buncolgen.SuiteRunColumns.AgentDefinitionID.Eq(), req.AgentDefinitionID)
}

func (r *suiteRunRepository) History(
	ctx context.Context,
	req repositories.AgentSuiteRunHistoryRequest,
) ([]*agentquality.SuiteRun, error) {
	if len(req.AgentDefinitionIDs) == 0 {
		return []*agentquality.SuiteRun{}, nil
	}

	cols := buncolgen.SuiteRunColumns
	perAgent := clampLimit(req.PerAgent, defaultHistoryPerAgent, maxHistoryPerAgent)
	dba := r.db.DBForContext(ctx)

	ranked := dba.NewSelect().
		Model((*agentquality.SuiteRun)(nil)).
		ColumnExpr(buncolgen.SuiteRunTable.All()).
		ColumnExpr(buncolgen.Expr(
			"row_number() OVER (PARTITION BY {0} ORDER BY {1} DESC, {2} DESC) AS "+
				historyRankColumnAlias,
			cols.AgentDefinitionID,
			cols.StartedAt,
			cols.ID,
		)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.In(), bun.List(req.AgentDefinitionIDs)).
				Where(cols.Status.In(), bun.List(scoredStatuses())).
				Where(cols.QualityScore.IsNotNull()).
				Where(cols.StartedAt.Gte(), req.Since)
		})

	runs := make([]*agentquality.SuiteRun, 0, len(req.AgentDefinitionIDs)*perAgent)
	if err := dba.NewSelect().
		Model(&runs).
		ModelTableExpr("(?) AS ?", ranked, bun.Ident(historyDerivedTableName)).
		Column(buncolgen.SuiteRunInsertableColumns...).
		Where("? <= ?", bun.Ident(historyRankColumnAlias), perAgent).
		Order(cols.AgentDefinitionID.OrderAsc(), cols.StartedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent quality history: %w", err)
	}

	return runs, nil
}

func scoredStatuses() []agentquality.SuiteRunStatus {
	return []agentquality.SuiteRunStatus{
		agentquality.SuiteRunStatusCompleted,
		agentquality.SuiteRunStatusBudgetStopped,
	}
}

func (r *suiteRunRepository) Latest(
	ctx context.Context,
	req repositories.LatestAgentSuiteRunsRequest,
) ([]*agentquality.SuiteRun, error) {
	cols := buncolgen.SuiteRunColumns
	limit := clampLimit(req.Limit, maxLatestAgents, maxLatestAgents)

	runs := make([]*agentquality.SuiteRun, 0, min(limit, len(req.AgentDefinitionIDs)+1))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&runs).
		DistinctOn(cols.AgentDefinitionID.Qualified()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo)
			if len(req.AgentDefinitionIDs) > 0 {
				sq = sq.Where(cols.AgentDefinitionID.In(), bun.List(req.AgentDefinitionIDs))
			}
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}

			return sq
		}).
		Order(cols.AgentDefinitionID.OrderAsc(), cols.StartedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read each agent's latest suite run: %w", err)
	}

	return runs, nil
}

func (r *suiteRunRepository) Overview(
	ctx context.Context,
	req repositories.AgentSuiteRunOverviewRequest,
) (*repositories.AgentSuiteRunOverview, error) {
	cols := buncolgen.SuiteRunColumns
	overview := new(repositories.AgentSuiteRunOverview)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*agentquality.SuiteRun)(nil)).
		ColumnExpr(buncolgen.CountFilter("runs", cols.Status.In()), bun.List(scoredStatuses())).
		ColumnExpr(buncolgen.CountFilter("regressions", cols.Regression.IsTrue())).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SuiteRunScopeTenant(sq, req.TenantInfo).
				Where(cols.StartedAt.Gte(), req.Since)
		}).
		Scan(ctx, overview); err != nil {
		return nil, fmt.Errorf("summarize agent suite runs: %w", err)
	}

	return overview, nil
}

func (r *suiteRunRepository) ListAgents(
	ctx context.Context,
	req repositories.ListAgentQualityAgentsRequest,
) (*repositories.AgentQualityAgentRows, error) {
	cols := buncolgen.DefinitionColumns
	limit := clampLimit(req.Limit, defaultPageLimit, maxPageLimit-1)
	dba := r.db.DBForContext(ctx)

	scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
		return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo)
	}

	result := &repositories.AgentQualityAgentRows{}
	if req.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*agentdefinition.Definition)(nil)).
			WhereGroup(" AND ", scope).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count agents: %w", err)
		}
		result.TotalCount = &total
	}

	rows := make([]*repositories.AgentQualityAgentRow, 0, limit+1)
	if err := dba.NewSelect().
		Model((*agentdefinition.Definition)(nil)).
		ColumnExpr(cols.ID.As("id")).
		ColumnExpr(cols.Name.As("name")).
		ColumnExpr(cols.Enabled.As("enabled")).
		WhereGroup(" AND ", scope).
		OrderExpr(cols.Name.OrderAsc()).
		OrderExpr(cols.ID.OrderAsc()).
		Limit(limit+1).
		Offset(max(req.Offset, 0)).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("list agents for quality: %w", err)
	}

	if len(rows) > limit {
		result.HasNextPage = true
		rows = rows[:limit]
	}
	result.Items = rows

	return result, nil
}
