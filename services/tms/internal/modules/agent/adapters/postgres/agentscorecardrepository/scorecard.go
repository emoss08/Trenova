package agentscorecardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/shopspring/decimal"
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

func New(p Params) agentports.AgentScorecardRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-scorecard-repository"),
	}
}

type runCounts struct {
	Runs   int
	Failed int
}

type toolRow struct {
	ToolName  string
	Approved  int
	Modified  int
	Rejected  int
	Pending   int
	Executed  int
	Failed    int
	Automatic int
}

// verdictRow is one reason a tool's calls ended one way, ranked among the
// reasons for that tool and verdict, with the verdict's own total beside it.
type verdictRow struct {
	ToolName string
	Verdict  string
	Reason   string
	Calls    int
	Total    int
}

type usageTotals struct {
	InputTokens  int
	OutputTokens int
	CostUSD      decimal.Decimal
}

type scorecardReads struct {
	runs       runCounts
	tools      []toolRow
	verdicts   []verdictRow
	exceptions int64
	usage      usageTotals
}

// Aggregate answers the whole scorecard in one round trip of five counted
// reads: the runs, the proposals grouped by tool, how each tool's calls
// ended, the exceptions, and what the model cost. None of them returns a row
// per record, so the cost of the page does not grow with how busy the agent
// has been.
func (r *repository) Aggregate(
	ctx context.Context,
	req agentports.ScorecardRequest,
) (*agent.ScorecardTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.ScorecardTotals, error) {
		var reads scorecardReads
		if err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(runsQuery(&req), &reads.runs),
			keel.IntoSlice(proposalsQuery(&req), &reads.tools),
			keel.IntoSlice(verdictsQuery(&req), &reads.verdicts),
			keel.CountInto(exceptionsQuery(&req), &reads.exceptions),
			keel.IntoValue(usageQuery(&req), &reads.usage),
		); err != nil {
			r.l.Error("failed to read the agent scorecard", zap.Error(err))

			return nil, fmt.Errorf("read agent scorecard: %w", err)
		}

		totals := &agent.ScorecardTotals{
			Runs:         reads.runs.Runs,
			RunsFailed:   reads.runs.Failed,
			Exceptions:   int(reads.exceptions),
			InputTokens:  reads.usage.InputTokens,
			OutputTokens: reads.usage.OutputTokens,
			CostUSD:      reads.usage.CostUSD,
			ByTool:       make([]agent.ToolOutcomeCount, 0, len(reads.tools)),
			ToolVerdicts: foldVerdicts(reads.verdicts),
			Trend:        []agent.ScorecardPoint{},
		}
		for _, row := range reads.tools {
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

		return totals, nil
	})
}

func runsQuery(req *agentports.ScorecardRequest) *keel.Query[agent.AgentRun] {
	run := agentdb.AgentRuns

	return run.Pick(keel.Count(), keel.Count().Filter(run.Status.Eq(agent.RunStatusFailed))).
		Where(run.AgentDefinitionID.Eq(req.AgentDefinitionID), run.CreatedAt.Gte(req.Since))
}

// proposalsQuery groups by tool, because "which of this agent's tools is
// being rejected" is the question a person actually has, and one total hides
// a bad tool inside nine good ones. An automatic write is one the trust
// ladder let through: it ran at the auto tier, so nobody was asked.
func proposalsQuery(req *agentports.ScorecardRequest) *keel.Query[agent.AgentProposal] {
	p := agentdb.AgentProposals
	status := func(s agent.ProposalStatus) keel.Agg {
		return keel.Count().Filter(p.Status.Eq(s))
	}

	return p.Group(p.ToolName).
		Pick(
			p.ToolName,
			status(agent.ProposalStatusAccepted),
			status(agent.ProposalStatusModified),
			status(agent.ProposalStatusRejected),
			status(agent.ProposalStatusPending),
			keel.Count().Filter(p.ExecutedAt.IsNotNull()),
			status(agent.ProposalStatusExecutionFailed),
			keel.Count().Filter(keel.And(
				p.AutonomyTier.Eq(agent.TierAutoExecute),
				p.ExecutedAt.IsNotNull(),
			)),
		).
		Where(p.Run().AgentDefinitionID.Eq(req.AgentDefinitionID), p.CreatedAt.Gte(req.Since)).
		Order(p.ToolName.Asc())
}

// verdictsQuery counts the agent's tool calls by tool and by how they ended,
// with the reasons given most often, from the step ledger: every call that
// reached dispatch is a step, its verdict and its refusal reason included,
// whether a run or a conversation turn made it. The step names the agent
// that made the call, so a delegate's calls count toward the delegate.
//
// A step from before verdicts were kept reads as its status implies, and a
// step still Started, whose outcome nobody recorded, as unknown. The ranking
// is done in SQL, so only the few reasons kept per verdict leave the database.
func verdictsQuery(req *agentports.ScorecardRequest) *keel.Query[keel.Row] {
	step := agentdb.AgentRunSteps
	labelled := keel.CTE("labelled", []string{"tool_name", "verdict", "reason"}, step.
		Pick(
			step.ToolName,
			keel.DynOf(step.Outcome.Text("verdict")).NullIf("").Coalesce(keel.DynCase().
				When(
					keel.DynOf(step.Status).Eq(string(agentports.RunStepStarted)),
					keel.DynLit(string(aitrace.OutcomeUnknown)).Cast(keel.TypeString),
				).
				When(
					keel.DynOf(step.Status).Eq(string(agentports.RunStepFailed)),
					keel.DynLit(string(aitrace.OutcomeFailed)).Cast(keel.TypeString),
				).
				Else(keel.DynLit(string(aitrace.OutcomeRan)).Cast(keel.TypeString))),
			keel.DynOf(step.Outcome.Text("reason")).Coalesce(""),
		).
		Where(
			step.Kind.Eq(string(agentports.RunStepTool)),
			step.AgentDefinitionID.Eq(req.AgentDefinitionID),
			step.CreatedAt.Gte(req.Since),
		))

	tool, verdict, reason := labelled.Col(
		"tool_name",
	), labelled.Col(
		"verdict",
	), labelled.Col(
		"reason",
	)
	partition := []keel.Expr{tool, verdict}
	ranked := keel.CTE(
		"ranked",
		[]string{"tool_name", "verdict", "reason", "calls", "total", "reason_rank"},
		labelled.Query().
			Group(tool, verdict, reason).
			Pick(
				tool,
				verdict,
				reason,
				keel.Count(),
				keel.DynOf(keel.Sum(keel.Count()).Over(partition)).Cast(keel.TypeInt64),
				keel.RowNumber().Over(
					partition,
					keel.Asc(keel.DynOf(reason).Eq("")),
					keel.Count().Desc(),
					reason.Asc(),
				),
			),
	)

	rank := ranked.Col("reason_rank")

	return ranked.Query().With(labelled, ranked).
		Pick(
			ranked.Col("tool_name"),
			ranked.Col("verdict"),
			ranked.Col("reason"),
			ranked.Col("calls"),
			ranked.Col("total"),
		).
		Where(rank.Lte(agent.MaxVerdictReasons)).
		Order(ranked.Col("tool_name").Asc(), ranked.Col("verdict").Asc(), rank.Asc())
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

func exceptionsQuery(req *agentports.ScorecardRequest) *keel.Query[agent.AgentException] {
	e := agentdb.AgentExceptions
	run := agentdb.AgentRuns

	return e.Join(run, run.ID.EqCol(e.RunID)).
		Where(run.AgentDefinitionID.Eq(req.AgentDefinitionID), e.CreatedAt.Gte(req.Since))
}

func usageQuery(req *agentports.ScorecardRequest) *keel.Query[aiusage.AIUsageRecord] {
	u := aiusagedb.AIUsageRecords

	return u.Pick(
		keel.DynOf(u.InputTokens).Sum().Coalesce(0),
		keel.DynOf(u.OutputTokens).Sum().Coalesce(0),
		keel.DynOf(u.CostUSD).Sum().Coalesce(0),
	).Where(u.AgentDefinitionID.Eq(req.AgentDefinitionID), u.CreatedAt.Gte(req.Since))
}
