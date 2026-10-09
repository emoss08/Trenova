package agentscorecardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
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

// Aggregate answers the whole scorecard in five counted queries: the runs,
// the proposals grouped by tool, how each tool's calls ended, the exceptions,
// and what the model cost.
// None of them returns a row per record, so the cost of the page does not
// grow with how busy the agent has been.
func (r *repository) Aggregate(
	ctx context.Context,
	req repositories.ScorecardRequest,
) (*agent.ScorecardTotals, error) {
	totals := &agent.ScorecardTotals{
		CostUSD:      decimal.Zero,
		ByTool:       []agent.ToolOutcomeCount{},
		ToolVerdicts: []agent.ToolVerdictCount{},
		Trend:        []agent.ScorecardPoint{},
	}

	for _, step := range []func(context.Context, repositories.ScorecardRequest, *agent.ScorecardTotals) error{
		r.runs,
		r.proposals,
		r.toolVerdicts,
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
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
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
	})
}

// proposals groups by tool, because "which of this agent's tools is being
// rejected" is the question a person actually has, and one total hides a bad
// tool inside nine good ones.
func (r *repository) proposals(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
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
	})
}

const (
	verdictLabelled = "labelled"
	verdictRanked   = "ranked"
	verdictToolName = "tool_name"
	verdictVerdict  = "verdict"
	verdictReason   = "reason"
	verdictCalls    = "calls"
	verdictTotal    = "total"
	verdictRank     = "reason_rank"
)

// verdictRow is one reason a tool's calls ended one way, ranked among the
// reasons for that tool and verdict, with the verdict's own total beside it.
type verdictRow struct {
	ToolName string `bun:"tool_name"`
	Verdict  string `bun:"verdict"`
	Reason   string `bun:"reason"`
	Calls    int    `bun:"calls"`
	Total    int    `bun:"total"`
}

// toolVerdicts counts the agent's tool calls by tool and by how they ended,
// with the reasons given most often, from the step ledger: every call that
// reached dispatch is a step, its verdict and its refusal reason included,
// whether a run or a conversation turn made it. The step names the agent
// that made the call, so a delegate's calls count toward the delegate.
//
// A step from before verdicts were kept reads as its status implies, and a
// step still Started, whose outcome nobody recorded, as unknown. The ranking
// is done in SQL, so only the few reasons kept per verdict leave the database.
func (r *repository) toolVerdicts(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)

		step := buncolgen.AgentRunStepColumns

		labelled := db.NewSelect().
			Model((*agent.AgentRunStep)(nil)).
			ColumnExpr(step.ToolName.As(verdictToolName)).
			ColumnExpr(
				buncolgen.Expr(
					"COALESCE(NULLIF({0}->>'verdict', ''), "+
						"CASE {1} WHEN ? THEN ? WHEN ? THEN ? ELSE ? END) AS ?",
					step.Outcome,
					step.Status,
				),
				services.RunStepStarted, aitrace.OutcomeUnknown,
				services.RunStepFailed, aitrace.OutcomeFailed,
				aitrace.OutcomeRan,
				bun.Ident(verdictVerdict),
			).
			ColumnExpr(
				step.Outcome.Expr("COALESCE({}->>'reason', '') AS ?"),
				bun.Ident(verdictReason),
			).
			Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentRunStepScopeTenant(q, req.TenantInfo)
			}).
			Where(step.Kind.Eq(), services.RunStepTool).
			Where(step.AgentDefinitionID.Eq(), req.AgentDefinitionID).
			Where(step.CreatedAt.Gte(), req.Since)

		ranked := db.NewSelect().
			TableExpr("(?) AS ?", labelled, bun.Ident(verdictLabelled)).
			ColumnExpr("?, ?, ?", bun.Ident(verdictToolName), bun.Ident(verdictVerdict),
				bun.Ident(verdictReason)).
			ColumnExpr("count(*) AS ?", bun.Ident(verdictCalls)).
			ColumnExpr("(sum(count(*)) OVER (PARTITION BY ?, ?))::bigint AS ?",
				bun.Ident(verdictToolName), bun.Ident(verdictVerdict), bun.Ident(verdictTotal)).
			ColumnExpr(
				"row_number() OVER (PARTITION BY ?, ? "+
					"ORDER BY (? = '') ASC, count(*) DESC, ? ASC) AS ?",
				bun.Ident(verdictToolName), bun.Ident(verdictVerdict),
				bun.Ident(verdictReason), bun.Ident(verdictReason), bun.Ident(verdictRank),
			).
			GroupExpr("?, ?, ?", bun.Ident(verdictToolName), bun.Ident(verdictVerdict),
				bun.Ident(verdictReason))

		var rows []verdictRow
		err := db.NewSelect().
			TableExpr("(?) AS ?", ranked, bun.Ident(verdictRanked)).
			ColumnExpr("?, ?, ?, ?, ?",
				bun.Ident(verdictToolName), bun.Ident(verdictVerdict), bun.Ident(verdictReason),
				bun.Ident(verdictCalls), bun.Ident(verdictTotal)).
			Where("? <= ?", bun.Ident(verdictRank), agent.MaxVerdictReasons).
			OrderExpr("? ASC, ? ASC, ? ASC",
				bun.Ident(verdictToolName), bun.Ident(verdictVerdict), bun.Ident(verdictRank)).
			Scan(ctx, &rows)
		if err != nil {
			r.l.Error("failed to count agent tool verdicts", zap.Error(err))

			return fmt.Errorf("count agent tool verdicts: %w", err)
		}

		totals.ToolVerdicts = foldVerdicts(rows)

		return nil
	})
}

// foldVerdicts turns the ranked reason rows, ordered by tool, verdict and
// rank, into one count per tool and verdict. A verdict's total is on every
// one of its rows; an empty reason is a call that gave none.
func foldVerdicts(rows []verdictRow) []agent.ToolVerdictCount {
	out := make([]agent.ToolVerdictCount, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		last := len(out) - 1
		if last < 0 || out[last].ToolName != row.ToolName || out[last].Verdict != row.Verdict {
			out = append(out, agent.ToolVerdictCount{
				ToolName:   row.ToolName,
				Verdict:    row.Verdict,
				Calls:      row.Total,
				TopReasons: make([]agent.ToolVerdictReason, 0, agent.MaxVerdictReasons),
			})
			last++
		}
		if row.Reason == "" {
			continue
		}
		out[last].TopReasons = append(out[last].TopReasons, agent.ToolVerdictReason{
			Reason: row.Reason,
			Calls:  row.Calls,
		})
	}

	return out
}

func (r *repository) exceptions(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
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
	})
}

func (r *repository) usage(
	ctx context.Context,
	req repositories.ScorecardRequest,
	totals *agent.ScorecardTotals,
) error {
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
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
	})
}
