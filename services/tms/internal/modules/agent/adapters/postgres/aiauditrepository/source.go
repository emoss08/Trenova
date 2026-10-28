package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// The run event kinds the trail reads: a tool call that ended without a
// step (a refusal before anything was claimed) and the edges of a delegated
// task.
var projectedEventKinds = []string{
	assistantports.AssistantEventToolFinished,
	assistantports.AssistantEventDelegateStarted,
	assistantports.AssistantEventDelegateFinished,
}

type sourceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSource(p Params) aiauditports.AIAuditSourceRepository {
	return &sourceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-source-repository"),
	}
}

func keyset[T any](
	q *keel.Query[*T],
	ts keel.Int64Col,
	id keelkinds.PulidIDCol,
	page aiauditports.AIAuditSourcePage,
) *keel.Query[*T] {
	return q.AllTenants().
		Where(
			keel.Tuple(ts, id).Gt(keel.Tuple(page.AfterTS, page.AfterID)),
			ts.Lte(page.UntilTS),
		).
		Order(ts.Asc(), id.Asc()).
		Limit(page.Limit)
}

func (r *sourceRepository) ListRuns(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*agent.AgentRun, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project agent runs into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		rows, err := keyset(run.Omit(run.Transcript), run.UpdatedAt, run.ID, page).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent runs for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListTurns(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*conversation.AssistantTurn, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project assistant turns into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.AssistantTurn, error) {
		turn := conversationdb.AssistantTurns
		rows, err := keyset(turn.Query(), turn.UpdatedAt, turn.ID, page).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read assistant turns for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListUsage(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*aiusage.AIUsageRecord, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project AI usage into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiusage.AIUsageRecord, error) {
		u := aiusagedb.AIUsageRecords
		rows, err := keyset(u.Query(), u.CreatedAt, u.ID, page).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read AI usage for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListSteps(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*agent.AgentRunStep, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project agent steps into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRunStep, error) {
		step := agentdb.AgentRunSteps
		rows, err := keyset(
			step.Where(step.Kind.Eq(string(agentruntimeports.RunStepTool))),
			step.UpdatedAt, step.ID, page,
		).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent run steps for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListEvents(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*agent.AgentRunEvent, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project agent events into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRunEvent, error) {
		event := agentdb.AgentRunEvents
		rows, err := keyset(
			event.Where(event.Kind.In(projectedEventKinds...)),
			event.CreatedAt, event.ID, page,
		).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent run events for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListProposals(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*agent.AgentProposal, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project proposals into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		rows, err := keyset(p.Query(), p.UpdatedAt, p.ID, page).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent proposals for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ListDecisions(
	ctx context.Context,
	page aiauditports.AIAuditSourcePage,
) ([]*agent.AgentDecision, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project proposal decisions into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentDecision, error) {
		d := agentdb.AgentDecisions
		rows, err := keyset(
			d.Omit(d.Preview).Where(d.ProposalID.IsNotNull()),
			d.CreatedAt, d.ID, page,
		).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent decisions for the AI audit trail: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) RunsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		if len(ids) == 0 {
			return []*agent.AgentRun{}, nil
		}

		run := agentdb.AgentRuns
		rows, err := run.Where(run.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent runs by id: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) TurnsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.AssistantTurn, error) {
		if len(ids) == 0 {
			return []*conversation.AssistantTurn{}, nil
		}

		turn := conversationdb.AssistantTurns
		rows, err := turn.Where(turn.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read assistant turns by id: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) TurnsByRunIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	runIDs []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.AssistantTurn, error) {
		if len(runIDs) == 0 {
			return []*conversation.AssistantTurn{}, nil
		}

		turn := conversationdb.AssistantTurns
		rows, err := turn.Where(turn.RunID.In(runIDs...)).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read assistant turns by run: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) TurnsInWindow(
	ctx context.Context,
	req *aiauditports.AIAuditTurnWindow,
) ([]*conversation.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.AssistantTurn, error) {
		if req == nil || len(req.ThreadIDs) == 0 {
			return make([]*conversation.AssistantTurn, 0), nil
		}

		turn := conversationdb.AssistantTurns
		rows, err := turn.
			Where(
				turn.ThreadID.In(req.ThreadIDs...),
				turn.StartedAt.Lte(req.To),
				keel.Or(turn.CompletedAt.IsNull(), turn.CompletedAt.Gte(req.From)),
			).
			Order(turn.StartedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read assistant turns in a window: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ThreadsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.Thread, error) {
		if len(ids) == 0 {
			return []*conversation.Thread{}, nil
		}

		th := conversationdb.Threads
		rows, err := th.
			Pick(th.ID, th.OrganizationID, th.BusinessUnitID, th.UserID, th.AgentDefinitionID).
			Where(th.ID.In(ids...)).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read assistant threads by id: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) EvaluationsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.Evaluation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Evaluation, error) {
		if len(ids) == 0 {
			return []*agent.Evaluation{}, nil
		}

		ev := agentdb.Evaluations
		rows, err := ev.
			Pick(
				ev.ID,
				ev.OrganizationID,
				ev.BusinessUnitID,
				ev.AgentDefinitionID,
				ev.DefinitionVersion,
			).
			Where(ev.ID.In(ids...)).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent evaluations by id: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) ProposalsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(ids) == 0 {
			return []*agent.AgentProposal{}, nil
		}

		p := agentdb.AgentProposals
		rows, err := p.Where(p.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read agent proposals by id: %w", err)
		}

		return rows, nil
	})
}

func (r *sourceRepository) StartedToolSteps(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ownerIDs []pulid.ID,
) ([]*agent.AgentRunStep, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"project started tool steps into the AI audit chain across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRunStep, error) {
		if len(ownerIDs) == 0 {
			return []*agent.AgentRunStep{}, nil
		}

		step := agentdb.AgentRunSteps
		rows, err := step.AllTenants().
			Where(
				step.OrganizationID.Eq(tenantInfo.OrgID),
				step.BusinessUnitID.Eq(tenantInfo.BuID),
				step.OwnerID.In(ownerIDs...),
				step.Kind.Eq(string(agentruntimeports.RunStepTool)),
				step.Status.Eq(string(agentruntimeports.RunStepStarted)),
			).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read unsettled agent run steps: %w", err)
		}

		return rows, nil
	})
}

type ownedCall struct {
	OwnerID pulid.ID
	CallID  string
}

func (r *sourceRepository) ToolStepCalls(
	ctx context.Context,
	_ pagination.TenantInfo,
	calls []aiauditports.OwnerCall,
) (map[aiauditports.OwnerCall]struct{}, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[aiauditports.OwnerCall]struct{}, error) {
			found := make(map[aiauditports.OwnerCall]struct{}, len(calls))
			if len(calls) == 0 {
				return found, nil
			}

			owners := make([]pulid.ID, 0, len(calls))
			callIDs := make([]string, 0, len(calls))
			for _, call := range calls {
				owners = append(owners, call.OwnerID)
				callIDs = append(callIDs, call.CallID)
			}

			step := agentdb.AgentRunSteps
			rows, err := keel.Into[ownedCall](ctx, r.db.Keel(ctx), step.
				Where(
					step.OwnerID.In(owners...),
					step.CallID.In(callIDs...),
					step.Kind.Eq(string(agentruntimeports.RunStepTool)),
				).
				Pick(step.OwnerID, step.CallID))
			if err != nil {
				return nil, fmt.Errorf("read the tool calls agent run steps claimed: %w", err)
			}

			for _, row := range rows {
				found[aiauditports.OwnerCall{OwnerID: row.OwnerID, CallID: row.CallID}] = struct{}{}
			}

			return found, nil
		},
	)
}

type namedRow struct {
	ID   pulid.ID
	Name string
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
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		if len(ids) == 0 {
			return map[pulid.ID]string{}, nil
		}

		u := tenantdb.Users
		rows, err := keel.Into[namedRow](ctx, r.db.Keel(ctx),
			u.Where(u.ID.In(ids...)).Pick(u.ID, u.Name))
		if err != nil {
			return nil, fmt.Errorf("read user names for the AI audit trail: %w", err)
		}

		return namesOf(rows), nil
	})
}

func (r *sourceRepository) AgentNames(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		if len(ids) == 0 {
			return map[pulid.ID]string{}, nil
		}

		d := agentdefinitiondb.Definitions
		rows, err := keel.Into[namedRow](ctx, r.db.Keel(ctx),
			d.Where(d.ID.In(ids...)).Pick(d.ID, d.Name))
		if err != nil {
			return nil, fmt.Errorf("read agent names for the AI audit trail: %w", err)
		}

		return namesOf(rows), nil
	})
}
