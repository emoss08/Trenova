package agentproposalrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/shared/money"
	"go.uber.org/zap"
)

// RecordExecution stores the outcome of running an approved proposal.
//
// Every outcome column is always written, including on success where the
// error is cleared and on failure where the result is: a proposal that failed,
// was re-approved, and then succeeded must not keep showing the old failure,
// nor a failed one what an earlier run made.
func (r *repository) RecordExecution(
	ctx context.Context,
	req *agentproposalports.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		executedAt := p.ExecutedAt.ToNull()
		if req.ExecutedAt != nil {
			executedAt = p.ExecutedAt.To(*req.ExecutedAt)
		}
		result := p.ExecutionResult.ToNull()
		if bounded := req.ExecutionResult.Bounded(); bounded != nil {
			result = p.ExecutionResult.To(bounded)
		}
		targetVersion := p.ExecutedTargetVersion.ToNull()
		if req.ExecutedTargetVersion != nil {
			targetVersion = p.ExecutedTargetVersion.To(*req.ExecutedTargetVersion)
		}

		sets := []keel.Assignment{
			p.Status.To(req.Status),
			executedAt,
			p.ExecutionError.To(req.ExecutionError),
			result,
			targetVersion,
		}
		if req.EgressClass.IsValid() {
			sets = append(sets, p.EgressClass.To(req.EgressClass))
		}
		if req.ExecutedByUserID.IsNotNil() {
			sets = append(sets, p.ExecutedByUserID.To(req.ExecutedByUserID))
		}

		entity, err := r.updateOne(ctx, req.ID, p.Where(p.ID.Eq(req.ID)), sets...)
		if err != nil {
			r.l.Error("failed to record proposal execution",
				zap.String("id", req.ID.String()), zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

// ListByThread returns every proposal raised during one assistant conversation.
//
// The link runs through the agent run rather than the proposal, because a
// proposal belongs to a run and a chat run's subject is the thread. Filtering on
// the subject type as well as the id keeps a thread id that happens to collide
// with some other subject's id from matching.
func (r *repository) ListByThread(
	ctx context.Context,
	req agentproposalports.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		run := p.Run()
		entities, err := p.Where(
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.Eq(req.ThreadID),
		).
			Order(p.CreatedAt.Asc(), p.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list proposals for thread",
				zap.String("threadId", req.ThreadID.String()), zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// RecordSimulation stores a preview in place of an execution. The proposal
// moves to Simulated whatever it was, because the decision that cleared it
// has been made and this is what came of it.
func (r *repository) RecordSimulation(
	ctx context.Context,
	req *agentproposalports.RecordAgentProposalSimulationRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		simulation := p.Simulation.ToNull()
		if req.Simulation != nil {
			simulation = p.Simulation.To(req.Simulation)
		}
		sets := []keel.Assignment{
			p.Status.To(agent.ProposalStatusSimulated),
			p.SimulatedAt.To(req.SimulatedAt),
			simulation,
		}
		if req.ExecutedByUserID.IsNotNil() {
			sets = append(sets, p.ExecutedByUserID.To(req.ExecutedByUserID))
		}

		entity, err := r.updateOne(ctx, req.ID, p.Where(p.ID.Eq(req.ID)), sets...)
		if err != nil {
			r.l.Error("failed to record proposal simulation",
				zap.String("id", req.ID.String()), zap.Error(err))

			return nil, fmt.Errorf("record proposal simulation: %w", err)
		}

		return entity, nil
	})
}

// CountExecutedTool counts one agent's executions of one tool since an
// instant, through the run that made each proposal.
func (r *repository) CountExecutedTool(
	ctx context.Context,
	req *agentproposalports.CountExecutedToolRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentProposals
		count, err := p.Where(
			p.Run().AgentDefinitionID.Eq(req.DefinitionID),
			p.ToolName.Eq(req.ToolName),
			p.ExecutedAt.Gte(req.Since),
			p.ExecutionError.IsNull(),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count executed tool proposals", zap.Error(err))

			return 0, fmt.Errorf("count executed tool proposals: %w", err)
		}

		return int(count), nil
	})
}

type automaticMoneyRow struct {
	Calls       int
	AmountMinor int64
}

var errMoneyCurrencyRequired = errors.New("sum automatic money writes: a currency is required")

func automaticMoneyQuery(
	req *agentproposalports.SumAutomaticMoneyRequest,
	currency string,
) *keel.Query[*agent.AgentProposal] {
	p := agentdb.AgentProposals
	status, at := agent.ProposalStatusExecuted, p.ExecutedAt
	if req.Simulated {
		status, at = agent.ProposalStatusSimulated, p.SimulatedAt
	}

	return p.Pick(keel.Count(), keel.NumericOf(p.MoneyMinor.Abs().Sum()).Coalesce(0)).
		Where(
			p.AutonomyTier.Eq(agent.TierAutoExecute),
			p.Status.Eq(status),
			p.MoneyMinor.IsNotNull(),
			p.MoneyCurrency.Eq(currency),
			at.IsNotNull(),
			keel.When(req.Since > 0, at.Gte(req.Since)),
			keel.When(req.RunID.IsNotNil(), p.RunID.Eq(req.RunID)),
		)
}

func (r *repository) SumAutomaticMoney(
	ctx context.Context,
	req *agentproposalports.SumAutomaticMoneyRequest,
) (agenttooltrust.MoneyUse, error) {
	currency := money.ISOCode(req.Currency)
	if currency == "" {
		return agenttooltrust.MoneyUse{}, errMoneyCurrencyRequired
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (agenttooltrust.MoneyUse, error) {
		row, err := keel.IntoOne[automaticMoneyRow](
			ctx,
			r.db.Keel(ctx),
			automaticMoneyQuery(req, currency),
		)
		if err != nil {
			r.l.Error("failed to sum automatic money writes", zap.Error(err))

			return agenttooltrust.MoneyUse{}, fmt.Errorf("sum automatic money writes: %w", err)
		}

		return agenttooltrust.MoneyUse{Calls: row.Calls, AmountMinor: row.AmountMinor}, nil
	})
}
