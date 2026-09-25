package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// The run event kinds the trail reads: a tool call that ended without a
// step (a refusal before anything was claimed) and the edges of a delegated
// task.
var projectedEventKinds = []string{
	serviceports.AssistantEventToolFinished,
	serviceports.AssistantEventDelegateStarted,
	serviceports.AssistantEventDelegateFinished,
}

type sourceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSource(p Params) repositories.AIAuditSourceRepository {
	return &sourceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-source-repository"),
	}
}

func keyset(
	q *bun.SelectQuery,
	ts, id *buncolgen.Column,
	page repositories.AIAuditSourcePage,
) *bun.SelectQuery {
	return q.
		Where(buncolgen.Expr("({0}, {1}) > (?, ?)", *ts, *id), page.AfterTS, page.AfterID).
		Where(ts.Lte(), page.UntilTS).
		Order(ts.OrderAsc(), id.OrderAsc()).
		Limit(page.Limit)
}

func (r *sourceRepository) ListRuns(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*agent.AgentRun, error) {
	cols := buncolgen.AgentRunColumns
	rows := make([]*agent.AgentRun, 0, page.Limit)

	if err := keyset(
		r.db.DBForContext(ctx).NewSelect().Model(&rows),
		&cols.UpdatedAt, &cols.ID, page,
	).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent runs for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListTurns(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*conversation.AssistantTurn, error) {
	cols := buncolgen.AssistantTurnColumns
	rows := make([]*conversation.AssistantTurn, 0, page.Limit)

	if err := keyset(
		r.db.DBForContext(ctx).NewSelect().Model(&rows),
		&cols.UpdatedAt, &cols.ID, page,
	).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read assistant turns for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListUsage(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*aiusage.AIUsageRecord, error) {
	cols := buncolgen.AIUsageRecordColumns
	rows := make([]*aiusage.AIUsageRecord, 0, page.Limit)

	if err := keyset(
		r.db.DBForContext(ctx).NewSelect().Model(&rows),
		&cols.CreatedAt, &cols.ID, page,
	).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read AI usage for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListSteps(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*agent.AgentRunStep, error) {
	cols := buncolgen.AgentRunStepColumns
	rows := make([]*agent.AgentRunStep, 0, page.Limit)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Where(cols.Kind.Eq(), string(serviceports.RunStepTool))
	if err := keyset(q, &cols.UpdatedAt, &cols.ID, page).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent run steps for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListEvents(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*agent.AgentRunEvent, error) {
	cols := buncolgen.AgentRunEventColumns
	rows := make([]*agent.AgentRunEvent, 0, page.Limit)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Where(cols.Kind.In(), bun.List(projectedEventKinds))
	if err := keyset(q, &cols.CreatedAt, &cols.ID, page).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent run events for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListProposals(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*agent.AgentProposal, error) {
	cols := buncolgen.AgentProposalColumns
	rows := make([]*agent.AgentProposal, 0, page.Limit)

	if err := keyset(
		r.db.DBForContext(ctx).NewSelect().Model(&rows),
		&cols.UpdatedAt, &cols.ID, page,
	).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent proposals for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ListDecisions(
	ctx context.Context,
	page repositories.AIAuditSourcePage,
) ([]*agent.AgentDecision, error) {
	cols := buncolgen.AgentDecisionColumns
	rows := make([]*agent.AgentDecision, 0, page.Limit)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		ExcludeColumn(cols.Preview.String()).
		Where(cols.ProposalID.IsNotNull())
	if err := keyset(q, &cols.CreatedAt, &cols.ID, page).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent decisions for the AI audit trail: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) RunsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentRun, error) {
	rows := make([]*agent.AgentRun, 0, len(ids))
	if len(ids) == 0 {
		return rows, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AgentRunApplyTenant(tenantInfo)).
		Where(buncolgen.AgentRunColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent runs by id: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) TurnsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	rows := make([]*conversation.AssistantTurn, 0, len(ids))
	if len(ids) == 0 {
		return rows, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AssistantTurnApplyTenant(tenantInfo)).
		Where(buncolgen.AssistantTurnColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read assistant turns by id: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) TurnsByRunIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runIDs []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	rows := make([]*conversation.AssistantTurn, 0, len(runIDs))
	if len(runIDs) == 0 {
		return rows, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AssistantTurnApplyTenant(tenantInfo)).
		Where(buncolgen.AssistantTurnColumns.RunID.In(), bun.List(runIDs)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read assistant turns by run: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) TurnsInWindow(
	ctx context.Context,
	req *repositories.AIAuditTurnWindow,
) ([]*conversation.AssistantTurn, error) {
	if req == nil || len(req.ThreadIDs) == 0 {
		return make([]*conversation.AssistantTurn, 0), nil
	}

	rows := make([]*conversation.AssistantTurn, 0, len(req.ThreadIDs))
	cols := buncolgen.AssistantTurnColumns
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AssistantTurnApplyTenant(req.TenantInfo)).
		Where(cols.ThreadID.In(), bun.List(req.ThreadIDs)).
		Where(cols.StartedAt.Lte(), req.To).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.CompletedAt.IsNull()).
				WhereOr(cols.CompletedAt.Gte(), req.From)
		}).
		Order(cols.StartedAt.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read assistant turns in a window: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ThreadsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.Thread, error) {
	rows := make([]*conversation.Thread, 0, len(ids))
	if len(ids) == 0 {
		return rows, nil
	}

	cols := buncolgen.ThreadColumns
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Column(
			cols.ID.String(),
			cols.OrganizationID.String(),
			cols.BusinessUnitID.String(),
			cols.UserID.String(),
			cols.AgentDefinitionID.String(),
		).
		Apply(buncolgen.ThreadApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read assistant threads by id: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) EvaluationsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.Evaluation, error) {
	rows := make([]*agent.Evaluation, 0, len(ids))
	if len(ids) == 0 {
		return rows, nil
	}

	cols := buncolgen.EvaluationColumns
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Column(
			cols.ID.String(),
			cols.OrganizationID.String(),
			cols.BusinessUnitID.String(),
			cols.AgentDefinitionID.String(),
			cols.DefinitionVersion.String(),
		).
		Apply(buncolgen.EvaluationApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent evaluations by id: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ProposalsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentProposal, error) {
	rows := make([]*agent.AgentProposal, 0, len(ids))
	if len(ids) == 0 {
		return rows, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AgentProposalApplyTenant(tenantInfo)).
		Where(buncolgen.AgentProposalColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read agent proposals by id: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) StartedToolSteps(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ownerIDs []pulid.ID,
) ([]*agent.AgentRunStep, error) {
	rows := make([]*agent.AgentRunStep, 0, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return rows, nil
	}

	cols := buncolgen.AgentRunStepColumns
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Apply(buncolgen.AgentRunStepApplyTenant(tenantInfo)).
		Where(cols.OwnerID.In(), bun.List(ownerIDs)).
		Where(cols.Kind.Eq(), string(serviceports.RunStepTool)).
		Where(cols.Status.Eq(), string(serviceports.RunStepStarted)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read unsettled agent run steps: %w", err)
	}

	return rows, nil
}

func (r *sourceRepository) ToolStepCalls(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	calls []repositories.OwnerCall,
) (map[repositories.OwnerCall]struct{}, error) {
	found := make(map[repositories.OwnerCall]struct{}, len(calls))
	if len(calls) == 0 {
		return found, nil
	}

	owners := make([]pulid.ID, 0, len(calls))
	callIDs := make([]string, 0, len(calls))
	for _, call := range calls {
		owners = append(owners, call.OwnerID)
		callIDs = append(callIDs, call.CallID)
	}

	type ownedCall struct {
		OwnerID pulid.ID `bun:"owner_id"`
		CallID  string   `bun:"call_id"`
	}
	rows := make([]ownedCall, 0, len(calls))

	cols := buncolgen.AgentRunStepColumns
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*agent.AgentRunStep)(nil)).
		Column(cols.OwnerID.String(), cols.CallID.String()).
		Apply(buncolgen.AgentRunStepApplyTenant(tenantInfo)).
		Where(cols.OwnerID.In(), bun.List(owners)).
		Where(cols.CallID.In(), bun.List(callIDs)).
		Where(cols.Kind.Eq(), string(serviceports.RunStepTool)).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read the tool calls agent run steps claimed: %w", err)
	}

	for _, row := range rows {
		found[repositories.OwnerCall{OwnerID: row.OwnerID, CallID: row.CallID}] = struct{}{}
	}

	return found, nil
}

type namedRow struct {
	ID   pulid.ID `bun:"id"`
	Name string   `bun:"name"`
}

func namesOf(rows []namedRow) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}

	return names
}

// UserNames reads the names of people a tenant's rows name. The ids come
// from the tenant's own rows, and the read is kept to its business unit.
func (r *sourceRepository) UserNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	if len(ids) == 0 {
		return map[pulid.ID]string{}, nil
	}

	cols := buncolgen.UserColumns
	rows := make([]namedRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*tenant.User)(nil)).
		Column(cols.ID.String(), cols.Name.String()).
		Where(cols.ID.In(), bun.List(ids)).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read user names for the AI audit trail: %w", err)
	}

	return namesOf(rows), nil
}

func (r *sourceRepository) AgentNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	if len(ids) == 0 {
		return map[pulid.ID]string{}, nil
	}

	cols := buncolgen.DefinitionColumns
	rows := make([]namedRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*agentdefinition.Definition)(nil)).
		Column(cols.ID.String(), cols.Name.String()).
		Apply(buncolgen.DefinitionApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read agent names for the AI audit trail: %w", err)
	}

	return namesOf(rows), nil
}
