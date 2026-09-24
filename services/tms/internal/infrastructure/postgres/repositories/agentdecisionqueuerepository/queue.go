package agentdecisionqueuerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLimit = 25
	maxLimit     = 100
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

func New(p Params) repositories.AgentDecisionQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdecisionqueue-repository"),
	}
}

// unionQuery is the queue as one relation: every pending proposal that
// stands on its own, and every pending plan as one row, each with the run's
// agent beside it. Both halves scope by tenant and skip what has expired.
type unionQuery struct {
	sql  strings.Builder
	args []any
}

func (u *unionQuery) write(fragment string, args ...any) {
	u.sql.WriteString(fragment)
	u.args = append(u.args, args...)
}

// pendingUnion writes the subquery; the caller selects from it as "q" with
// the columns kind, id, created_at, agent_definition_id and agent_name.
func pendingUnion(req repositories.ListPendingDecisionsRequest) *unionQuery {
	proposals := buncolgen.AgentProposalColumns
	plans := buncolgen.AgentPlanColumns
	runs := buncolgen.AgentRunColumns
	definitions := buncolgen.DefinitionColumns

	u := &unionQuery{}
	joinRunAndDefinition := func(ownerOrg, ownerBU, ownerRun string) {
		u.write(" JOIN " + buncolgen.AgentRunTable.Name + " AS " + buncolgen.AgentRunTable.Alias +
			" ON " + runs.ID.Qualified() + " = " + ownerRun +
			" AND " + runs.OrganizationID.Qualified() + " = " + ownerOrg +
			" AND " + runs.BusinessUnitID.Qualified() + " = " + ownerBU)
		u.write(
			" LEFT JOIN " + buncolgen.DefinitionTable.Name + " AS " + buncolgen.DefinitionTable.Alias +
				" ON " + definitions.ID.Qualified() + " = " + runs.AgentDefinitionID.Qualified() +
				" AND " + definitions.OrganizationID.Qualified() + " = " + runs.OrganizationID.Qualified() +
				" AND " + definitions.BusinessUnitID.Qualified() + " = " + runs.BusinessUnitID.Qualified(),
		)
	}
	commonWhere := func(org, bu, status, expiresAt string) {
		u.write(
			" WHERE "+org+" = ? AND "+bu+" = ? AND "+status+" = ?",
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			agent.ProposalStatusPending,
		)
		u.write(" AND ("+expiresAt+" IS NULL OR "+expiresAt+" = 0 OR "+expiresAt+" > ?)", req.Now)
		if !req.AgentDefinitionID.IsNil() {
			u.write(" AND "+runs.AgentDefinitionID.Qualified()+" = ?", req.AgentDefinitionID)
		}
		if req.ExcludeShadowDefinitions {
			u.write(" AND COALESCE(" + definitions.ShadowMode.Qualified() + ", FALSE) = FALSE")
		}
		writeAudience(u, req.Audience)
	}

	u.write("SELECT ? AS kind, "+proposals.ID.Qualified()+" AS id, "+
		proposals.CreatedAt.Qualified()+" AS created_at, "+
		runs.AgentDefinitionID.Qualified()+" AS agent_definition_id, "+
		"COALESCE("+definitions.Name.Qualified()+", '') AS agent_name, "+
		proposals.ToolName.Qualified()+" AS tool_name"+
		" FROM "+buncolgen.AgentProposalTable.Name+" AS "+buncolgen.AgentProposalTable.Alias,
		string(repositories.PendingDecisionProposal))
	joinRunAndDefinition(
		proposals.OrganizationID.Qualified(),
		proposals.BusinessUnitID.Qualified(),
		proposals.RunID.Qualified(),
	)
	commonWhere(
		proposals.OrganizationID.Qualified(),
		proposals.BusinessUnitID.Qualified(),
		proposals.Status.Qualified(),
		proposals.ExpiresAt.Qualified(),
	)
	u.write(" AND " + proposals.PlanID.Qualified() + " IS NULL")
	if req.ToolName != "" {
		u.write(" AND "+proposals.ToolName.Qualified()+" = ?", req.ToolName)
	}

	u.write(" UNION ALL ")

	u.write("SELECT ? AS kind, "+plans.ID.Qualified()+" AS id, "+
		plans.CreatedAt.Qualified()+" AS created_at, "+
		runs.AgentDefinitionID.Qualified()+" AS agent_definition_id, "+
		"COALESCE("+definitions.Name.Qualified()+", '') AS agent_name, "+
		"? AS tool_name"+
		" FROM "+buncolgen.AgentPlanTable.Name+" AS "+buncolgen.AgentPlanTable.Alias,
		string(repositories.PendingDecisionPlan), planToolName)
	joinRunAndDefinition(
		plans.OrganizationID.Qualified(),
		plans.BusinessUnitID.Qualified(),
		plans.RunID.Qualified(),
	)
	commonWhere(
		plans.OrganizationID.Qualified(),
		plans.BusinessUnitID.Qualified(),
		plans.Status.Qualified(),
		plans.ExpiresAt.Qualified(),
	)
	if req.ToolName != "" {
		u.write(" AND EXISTS (SELECT 1 FROM "+buncolgen.AgentProposalTable.Name+" AS step"+
			" WHERE step."+proposals.PlanID.Name+" = "+plans.ID.Qualified()+
			" AND step."+proposals.OrganizationID.Name+" = "+plans.OrganizationID.Qualified()+
			" AND step."+proposals.BusinessUnitID.Name+" = "+plans.BusinessUnitID.Qualified()+
			" AND step."+proposals.ToolName.Name+" = ?)", req.ToolName)
	}

	return u
}

// writeAudience keeps what agents the reader may use raised: an agent open to
// everyone, a system agent, one granted to the reader's roles, and a run of a
// retired built-in agent, which has no definition to restrict it.
func writeAudience(u *unionQuery, audience *repositories.AgentAudience) {
	if audience == nil {
		return
	}

	definitions := buncolgen.DefinitionColumns
	u.write(" AND ("+definitions.ID.Qualified()+" IS NULL OR "+
		definitions.AccessMode.Qualified()+" = ? OR "+
		definitions.SystemKey.Qualified()+" IS NOT NULL",
		agentdefinition.AccessEveryone)
	if len(audience.GrantedAgentIDs) > 0 {
		u.write(" OR "+definitions.ID.Qualified()+" IN (?)", bun.List(audience.GrantedAgentIDs))
	}
	u.write(")")
}

// planToolName is how a plan reads in a by-tool count: several writes
// decided as one, not any single tool's.
const planToolName = "plan"

func (r *repository) ListPending(
	ctx context.Context,
	req repositories.ListPendingDecisionsRequest,
) (*repositories.PendingDecisionsPage, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	inner := pendingUnion(req)
	q := &unionQuery{}
	q.write("SELECT kind, id, created_at FROM (")
	q.write(inner.sql.String(), inner.args...)
	q.write(") AS q")
	if req.After != nil {
		q.write(" WHERE (created_at < ? OR (created_at = ? AND id < ?))",
			req.After.CreatedAt, req.After.CreatedAt, req.After.ID)
	}
	q.write(" ORDER BY created_at DESC, id DESC LIMIT ?", limit+1)

	entries := make([]repositories.PendingDecisionEntry, 0, limit+1)
	if err := r.db.DBForContext(ctx).NewRaw(q.sql.String(), q.args...).Scan(ctx, &entries); err != nil {
		r.l.Error("failed to list pending decisions", zap.Error(err))

		return nil, fmt.Errorf("list pending decisions: %w", err)
	}

	page := &repositories.PendingDecisionsPage{Entries: entries}
	if len(entries) > limit {
		page.Entries = entries[:limit]
		page.HasNextPage = true
	}

	return page, nil
}

func (r *repository) CountPending(
	ctx context.Context,
	req repositories.ListPendingDecisionsRequest,
) (int, error) {
	inner := pendingUnion(req)
	q := &unionQuery{}
	q.write("SELECT COUNT(*) FROM (")
	q.write(inner.sql.String(), inner.args...)
	q.write(") AS q")

	var count int
	if err := r.db.DBForContext(ctx).NewRaw(q.sql.String(), q.args...).Scan(ctx, &count); err != nil {
		r.l.Error("failed to count pending decisions", zap.Error(err))

		return 0, fmt.Errorf("count pending decisions: %w", err)
	}

	return count, nil
}

func (r *repository) Summary(
	ctx context.Context,
	req repositories.PendingDecisionSummaryRequest,
) (*repositories.PendingDecisionSummary, error) {
	listReq := repositories.ListPendingDecisionsRequest{
		TenantInfo:               req.TenantInfo,
		ExcludeShadowDefinitions: req.ExcludeShadowDefinitions,
		Audience:                 req.Audience,
		Now:                      req.Now,
	}
	db := r.db.DBForContext(ctx)

	var totals struct {
		Total    int    `bun:"total"`
		OldestAt *int64 `bun:"oldest_at"`
	}
	inner := pendingUnion(listReq)
	q := &unionQuery{}
	q.write("SELECT COUNT(*) AS total, MIN(created_at) AS oldest_at FROM (")
	q.write(inner.sql.String(), inner.args...)
	q.write(") AS q")
	if err := db.NewRaw(q.sql.String(), q.args...).Scan(ctx, &totals); err != nil {
		r.l.Error("failed to summarize pending decisions", zap.Error(err))

		return nil, fmt.Errorf("summarize pending decisions: %w", err)
	}

	summary := &repositories.PendingDecisionSummary{
		Total:   totals.Total,
		ByAgent: []repositories.PendingDecisionAgentCount{},
		ByTool:  []repositories.PendingDecisionToolCount{},
	}
	if totals.Total == 0 {
		return summary, nil
	}
	summary.OldestAt = totals.OldestAt

	byAgent := make([]repositories.PendingDecisionAgentCount, 0, 8)
	inner = pendingUnion(listReq)
	q = &unionQuery{}
	q.write("SELECT agent_definition_id, agent_name, COUNT(*) AS count FROM (")
	q.write(inner.sql.String(), inner.args...)
	q.write(") AS q GROUP BY agent_definition_id, agent_name ORDER BY count DESC, agent_name ASC")
	if err := db.NewRaw(q.sql.String(), q.args...).Scan(ctx, &byAgent); err != nil {
		r.l.Error("failed to count pending decisions by agent", zap.Error(err))

		return nil, fmt.Errorf("count pending decisions by agent: %w", err)
	}
	summary.ByAgent = byAgent

	byTool := make([]repositories.PendingDecisionToolCount, 0, 8)
	inner = pendingUnion(listReq)
	q = &unionQuery{}
	q.write("SELECT tool_name, COUNT(*) AS count FROM (")
	q.write(inner.sql.String(), inner.args...)
	q.write(") AS q GROUP BY tool_name ORDER BY count DESC, tool_name ASC")
	if err := db.NewRaw(q.sql.String(), q.args...).Scan(ctx, &byTool); err != nil {
		r.l.Error("failed to count pending decisions by tool", zap.Error(err))

		return nil, fmt.Errorf("count pending decisions by tool: %w", err)
	}
	summary.ByTool = byTool

	return summary, nil
}
