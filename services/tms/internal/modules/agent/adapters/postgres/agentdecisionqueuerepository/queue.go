package agentdecisionqueuerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision/agentdecisiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan/agentplandb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLimit = 25
	maxLimit     = 100
)

var pendingColumns = []string{
	"kind",
	"id",
	"created_at",
	"agent_definition_id",
	"agent_name",
	"shadow",
	"tool_name",
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentdecisionqueueports.AgentDecisionQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdecisionqueue-repository"),
	}
}

type pendingTotals struct {
	Total    int64
	OldestAt *int64
}

// pendingQueue is the queue as one relation: every pending proposal that
// stands on its own, and every pending plan as one row, each with the run's
// agent beside it. Both halves are scoped to the tenant and skip what has
// expired.
func pendingQueue(req *agentdecisionqueueports.ListPendingDecisionsQuery) *keel.CTETable {
	p := agentdb.AgentProposals
	pRun := agentdb.AgentRuns.As("prun")
	pDef := agentdefinitiondb.Definitions.As("pdef")
	proposals := p.
		Pick(
			keel.Text(string(agentdecisionqueueports.PendingDecisionProposal)),
			p.ID,
			p.CreatedAt,
			pRun.AgentDefinitionID,
			pDef.Name.Coalesce(""),
			pDef.ShadowMode.Coalesce(false),
			keel.CastText(p.ToolName),
		).
		Join(pRun, pRun.ID.EqCol(p.RunID)).
		LeftJoin(pDef, pDef.ID.EqCol(pRun.AgentDefinitionID)).
		Where(
			p.Status.Eq(agent.ProposalStatusPending),
			keel.Or(p.ExpiresAt.IsNull(), p.ExpiresAt.Eq(0), p.ExpiresAt.Gt(req.Now)),
			keel.When(!req.AgentDefinitionID.IsNil(), pRun.AgentDefinitionID.Eq(req.AgentDefinitionID)),
			audience(pDef, req.Audience),
			p.PlanID.IsNull(),
			keel.When(req.ToolName != "", p.ToolName.Eq(req.ToolName)),
		)

	pl := agentplandb.AgentPlans
	plRun := agentdb.AgentRuns.As("plrun")
	plDef := agentdefinitiondb.Definitions.As("pldef")
	step := agentdb.AgentProposals.As("step")
	plans := pl.
		Pick(
			keel.Text(string(agentdecisionqueueports.PendingDecisionPlan)),
			pl.ID,
			pl.CreatedAt,
			plRun.AgentDefinitionID,
			plDef.Name.Coalesce(""),
			plDef.ShadowMode.Coalesce(false),
			keel.Text(agentplan.PlanToolName),
		).
		Join(plRun, plRun.ID.EqCol(pl.RunID)).
		LeftJoin(plDef, plDef.ID.EqCol(plRun.AgentDefinitionID)).
		Where(
			pl.Status.Eq(agentplan.PlanStatusPending),
			keel.Or(pl.ExpiresAt.IsNull(), pl.ExpiresAt.Eq(0), pl.ExpiresAt.Gt(req.Now)),
			keel.When(!req.AgentDefinitionID.IsNil(), plRun.AgentDefinitionID.Eq(req.AgentDefinitionID)),
			audience(plDef, req.Audience),
			keel.When(req.ToolName != "", keel.Exists(step.Where(
				step.PlanID.EqCol(pl.ID),
				step.ToolName.Eq(req.ToolName),
			))),
		)

	return keel.CTE("q", pendingColumns, keel.UnionAll(proposals, plans))
}

// audience keeps what agents the reader may use raised: an agent open to
// everyone, a system agent, one granted to the reader's roles, and a run of a
// retired built-in agent, which has no definition to restrict it.
func audience(
	def *agentdefinitiondb.DefinitionsTable,
	audience *kernelports.AgentAudience,
) keel.Pred {
	if audience == nil {
		return nil
	}

	return keel.Or(
		def.ID.IsNull(),
		def.AccessMode.Eq(agentdefinition.AccessEveryone),
		def.SystemKey.IsNotNull(),
		keel.When(len(audience.GrantedAgentIDs) > 0, def.ID.In(audience.GrantedAgentIDs...)),
	)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}

	return min(limit, maxLimit)
}

func (r *repository) ListPending(
	ctx context.Context,
	req *agentdecisionqueueports.ListPendingDecisionsQuery,
) (*agentdecisionqueueports.PendingDecisionsPageRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentdecisionqueueports.PendingDecisionsPageRow, error) {
			limit := clampLimit(req.Limit)
			q := pendingQueue(req)
			createdAt, id := q.Col("created_at"), q.Col("id")

			var after keel.Pred
			if req.After != nil {
				after = keel.Or(
					createdAt.Lt(req.After.CreatedAt),
					keel.And(createdAt.Eq(req.After.CreatedAt), id.Lt(req.After.ID.String())),
				)
			}

			entries, err := keel.Into[agentdecisionqueueports.PendingDecisionEntry](
				ctx,
				r.db.Keel(ctx),
				q.Query().With(q).
					Pick(q.Col("kind"), id, createdAt, q.Col("shadow")).
					Where(after).
					Order(createdAt.Desc(), id.Desc()).
					Limit(limit+1),
			)
			if err != nil {
				r.l.Error("failed to list pending decisions", zap.Error(err))

				return nil, fmt.Errorf("list pending decisions: %w", err)
			}

			page := &agentdecisionqueueports.PendingDecisionsPageRow{Entries: entries}
			if page.Entries == nil {
				page.Entries = []agentdecisionqueueports.PendingDecisionEntry{}
			}
			if len(entries) > limit {
				page.Entries = entries[:limit]
				page.HasNextPage = true
			}

			return page, nil
		},
	)
}

func (r *repository) CountPending(
	ctx context.Context,
	req *agentdecisionqueueports.ListPendingDecisionsQuery,
) (keel.RowCount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (keel.RowCount, error) {
		q := pendingQueue(req)
		count, err := q.Query().With(q).
			Where(keel.When(req.ActionableOnly, keel.Not(keel.BoolOf(q.Col("shadow"))))).
			CountEstimate(ctx, r.db.Keel(ctx), pagination.ExactTotalUpTo)
		if err != nil {
			r.l.Error("failed to count pending decisions", zap.Error(err))

			return keel.RowCount{}, fmt.Errorf("count pending decisions: %w", err)
		}

		return count, nil
	})
}

func (r *repository) Summary(
	ctx context.Context,
	req agentdecisionqueueports.PendingDecisionSummaryRequest,
) (*agentdecisionqueueports.PendingDecisionSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentdecisionqueueports.PendingDecisionSummary, error) {
			q := pendingQueue(&agentdecisionqueueports.ListPendingDecisionsQuery{
				TenantInfo: req.TenantInfo,
				Audience:   req.Audience,
				Now:        req.Now,
			})
			definitionID, agentName, toolName := q.Col("agent_definition_id"),
				q.Col("agent_name"),
				q.Col("tool_name")

			var (
				totals  pendingTotals
				byAgent []agentdecisionqueueports.PendingDecisionAgentCount
				byTool  []agentdecisionqueueports.PendingDecisionToolCount
			)
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoValue(
					q.Query().With(q).Pick(keel.Count(), q.Col("created_at").Min()),
					&totals,
				),
				keel.IntoSlice(
					q.Query().With(q).
						Group(definitionID, agentName).
						Pick(definitionID, agentName, keel.Count()).
						Order(keel.Count().Desc(), agentName.Asc()),
					&byAgent,
				),
				keel.IntoSlice(
					q.Query().With(q).
						Group(toolName).
						Pick(toolName, keel.Count()).
						Order(keel.Count().Desc(), toolName.Asc()),
					&byTool,
				),
			)
			if err != nil {
				r.l.Error("failed to summarize pending decisions", zap.Error(err))

				return nil, fmt.Errorf("summarize pending decisions: %w", err)
			}

			summary := &agentdecisionqueueports.PendingDecisionSummary{
				Total:   int(totals.Total),
				ByAgent: []agentdecisionqueueports.PendingDecisionAgentCount{},
				ByTool:  []agentdecisionqueueports.PendingDecisionToolCount{},
			}
			if totals.Total == 0 {
				return summary, nil
			}
			summary.OldestAt = totals.OldestAt
			if byAgent != nil {
				summary.ByAgent = byAgent
			}
			if byTool != nil {
				summary.ByTool = byTool
			}

			return summary, nil
		},
	)
}

func (r *repository) ListRecent(
	ctx context.Context,
	req agentdecisionqueueports.ListRecentDecisionsQuery,
) ([]agentdecisionqueueports.RecentDecisionEntry, error) {
	limit := clampLimit(req.Limit)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentdecisionqueueports.RecentDecisionEntry, error) {
			d := agentdecisiondb.AgentDecisions
			p := agentdb.AgentProposals
			run := agentdb.AgentRuns
			def := agentdefinitiondb.Definitions
			u := tenantdb.Users

			entries, err := keel.Into[agentdecisionqueueports.RecentDecisionEntry](
				ctx,
				r.db.Keel(ctx),
				d.
					Pick(
						d.ID,
						p.ID,
						keel.CastText(d.Decision),
						d.ReasonCode,
						d.Note.Coalesce(""),
						d.DecidedByUserID,
						u.Name.Coalesce(""),
						d.CreatedAt,
					).
					Join(p, p.ID.EqCol(d.ProposalID)).
					Join(run, run.ID.EqCol(p.RunID)).
					LeftJoin(def, def.ID.EqCol(run.AgentDefinitionID)).
					LeftJoin(u, u.ID.EqCol(d.DecidedByUserID)).
					Where(
						d.CreatedAt.Gte(req.Since),
						p.PlanID.IsNull(),
						audience(def, req.Audience),
					).
					Order(d.CreatedAt.Desc(), d.ID.Desc()).
					Limit(limit),
			)
			if err != nil {
				r.l.Error("failed to list recent decisions", zap.Error(err))

				return nil, fmt.Errorf("list recent decisions: %w", err)
			}
			if entries == nil {
				entries = []agentdecisionqueueports.RecentDecisionEntry{}
			}

			return entries, nil
		},
	)
}
