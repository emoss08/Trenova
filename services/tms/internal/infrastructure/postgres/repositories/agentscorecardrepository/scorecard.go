package agentscorecardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/shopspring/decimal"
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

func New(p Params) repositories.AgentScorecardRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-scorecard-repository"),
	}
}

// runJoin ties a record back to the agent that made it. Proposals and
// exceptions carry a run, and only the run knows which agent definition it
// belongs to.
var runJoin = "JOIN " + buncolgen.AgentRunTable.Name + " AS " + buncolgen.AgentRunTable.Alias +
	" ON " + buncolgen.AgentRunColumns.ID.EqColumn(buncolgen.AgentProposalColumns.RunID)

// Aggregate answers the whole scorecard in four counted queries: the runs,
// the proposals grouped by tool, the exceptions, and what the model cost.
// None of them returns a row per record, so the cost of the page does not
// grow with how busy the agent has been.
func (r *repository) Aggregate(
	ctx context.Context,
	req repositories.ScorecardRequest,
) (*agent.ScorecardTotals, error) {
	totals := &agent.ScorecardTotals{
		CostUSD: decimal.Zero,
		ByTool:  []agent.ToolOutcomeCount{},
		Trend:   []agent.ScorecardPoint{},
	}

	for _, step := range []func(context.Context, repositories.ScorecardRequest, *agent.ScorecardTotals) error{
		r.runs,
		r.proposals,
		r.exceptions,
		r.usage,
	} {
		if err := step(ctx, req, totals); err != nil {
			return nil, err
		}
	}

	return totals, nil
}

func (r *repository) runs(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	db := r.db.DBForContext(ctx)

	run := buncolgen.AgentRunColumns
	var counted struct {
		Runs   int `bun:"runs"`
		Failed int `bun:"failed"`
	}

	err := db.NewSelect().
		Model((*agent.AgentRun)(nil)).
		ColumnExpr(buncolgen.Count("runs")).
		ColumnExpr(buncolgen.CountFilter("failed", run.Status.Eq()), agent.RunStatusFailed).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentRunScopeTenant(q, req.TenantInfo)
		}).
		Where(run.AgentDefinitionID.Eq(), req.AgentDefinitionID).
		Where(run.CreatedAt.Gte(), req.Since).
		Scan(ctx, &counted)
	if err != nil {
		r.l.Error("failed to count agent runs", zap.Error(err))

		return fmt.Errorf("count agent runs: %w", err)
	}

	totals.Runs = counted.Runs
	totals.RunsFailed = counted.Failed

	return nil
}

// proposals groups by tool, because "which of this agent's tools is being
// rejected" is the question a person actually has, and one total hides a bad
// tool inside nine good ones.
func (r *repository) proposals(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	db := r.db.DBForContext(ctx)

	prop := buncolgen.AgentProposalColumns
	run := buncolgen.AgentRunColumns

	var rows []struct {
		ToolName  string `bun:"tool_name"`
		Approved  int    `bun:"approved"`
		Modified  int    `bun:"modified"`
		Rejected  int    `bun:"rejected"`
		Executed  int    `bun:"executed"`
		Failed    int    `bun:"failed"`
		Pending   int    `bun:"pending"`
		Automatic int    `bun:"automatic"`
	}

	err := db.NewSelect().
		Model((*agent.AgentProposal)(nil)).
		Column(prop.ToolName.Bare()).
		ColumnExpr(buncolgen.CountFilter("approved", prop.Status.Eq()), agent.ProposalStatusAccepted).
		ColumnExpr(buncolgen.CountFilter("modified", prop.Status.Eq()), agent.ProposalStatusModified).
		ColumnExpr(buncolgen.CountFilter("rejected", prop.Status.Eq()), agent.ProposalStatusRejected).
		ColumnExpr(buncolgen.CountFilter("pending", prop.Status.Eq()), agent.ProposalStatusPending).
		ColumnExpr(buncolgen.CountFilter("executed", prop.ExecutedAt.IsNotNull())).
		ColumnExpr(
			buncolgen.CountFilter("failed", prop.Status.Eq()),
			agent.ProposalStatusExecutionFailed,
		).
		// An automatic write is one the trust ladder let through: it ran at
		// the auto tier, so nobody was asked. Counting these separately is
		// what makes "how much is this agent doing on its own" answerable.
		ColumnExpr(
			buncolgen.CountFilter("automatic", prop.AutonomyTier.Eq(), prop.ExecutedAt.IsNotNull()),
			agent.TierAutoExecute,
		).
		Join(runJoin).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentProposalScopeTenant(q, req.TenantInfo)
		}).
		Where(run.AgentDefinitionID.Eq(), req.AgentDefinitionID).
		Where(prop.CreatedAt.Gte(), req.Since).
		GroupExpr(prop.ToolName.Qualified()).
		OrderExpr(prop.ToolName.OrderAsc()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to count agent proposals", zap.Error(err))

		return fmt.Errorf("count agent proposals: %w", err)
	}

	totals.ByTool = make([]agent.ToolOutcomeCount, 0, len(rows))
	for _, row := range rows {
		totals.ByTool = append(totals.ByTool, agent.ToolOutcomeCount{
			ToolName:  row.ToolName,
			Approved:  row.Approved,
			Modified:  row.Modified,
			Rejected:  row.Rejected,
			Executed:  row.Executed,
			Failed:    row.Failed,
			Pending:   row.Pending,
			Automatic: row.Automatic,
		})
	}

	return nil
}

func (r *repository) exceptions(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	db := r.db.DBForContext(ctx)

	exc := buncolgen.AgentExceptionColumns
	run := buncolgen.AgentRunColumns

	count, err := db.NewSelect().
		Model((*agent.AgentException)(nil)).
		Join("JOIN "+buncolgen.AgentRunTable.Name+" AS "+buncolgen.AgentRunTable.Alias+
			" ON "+run.ID.EqColumn(exc.RunID)).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentExceptionScopeTenant(q, req.TenantInfo)
		}).
		Where(run.AgentDefinitionID.Eq(), req.AgentDefinitionID).
		Where(exc.CreatedAt.Gte(), req.Since).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count agent exceptions", zap.Error(err))

		return fmt.Errorf("count agent exceptions: %w", err)
	}

	totals.Exceptions = count

	return nil
}

func (r *repository) usage(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	db := r.db.DBForContext(ctx)

	usage := buncolgen.AIUsageRecordColumns
	var counted struct {
		InputTokens  int              `bun:"input_tokens"`
		OutputTokens int              `bun:"output_tokens"`
		CostUSD      *decimal.Decimal `bun:"cost_usd"`
	}

	err := db.NewSelect().
		Model((*aiusage.AIUsageRecord)(nil)).
		ColumnExpr(buncolgen.Coalesce(usage.InputTokens, "0", "input_tokens")).
		ColumnExpr(buncolgen.Coalesce(usage.OutputTokens, "0", "output_tokens")).
		ColumnExpr(buncolgen.Coalesce(usage.CostUSD, "0", "cost_usd")).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AIUsageRecordScopeTenant(q, req.TenantInfo)
		}).
		Where(usage.AgentDefinitionID.Eq(), req.AgentDefinitionID).
		Where(usage.CreatedAt.Gte(), req.Since).
		Scan(ctx, &counted)
	if err != nil {
		r.l.Error("failed to sum agent usage", zap.Error(err))

		return fmt.Errorf("sum agent usage: %w", err)
	}

	totals.InputTokens = counted.InputTokens
	totals.OutputTokens = counted.OutputTokens
	if counted.CostUSD != nil {
		totals.CostUSD = *counted.CostUSD
	}

	return nil
}
