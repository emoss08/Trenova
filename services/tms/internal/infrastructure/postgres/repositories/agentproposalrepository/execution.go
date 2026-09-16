package agentproposalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// RecordExecution stores the outcome of running an approved proposal.
//
// Both columns are always written, including on success where the error is
// cleared: a proposal that failed, was re-approved, and then succeeded must not
// keep showing the old failure.
func (r *repository) RecordExecution(
	ctx context.Context,
	req repositories.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	log := r.l.With(
		zap.String("operation", "RecordExecution"),
		zap.String("id", req.ID.String()),
	)

	entity := new(agent.AgentProposal)
	cols := buncolgen.AgentProposalColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.Status.Set(), req.Status).
		Set(cols.ExecutedAt.Set(), req.ExecutedAt).
		Set(cols.ExecutionError.Set(), req.ExecutionError).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to record proposal execution", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// ListByThread returns every proposal raised during one assistant conversation.
//
// The link runs through the agent run rather than the proposal, because a
// proposal belongs to a run and a chat run's subject is the thread. Filtering on
// the subject type as well as the id keeps a thread id that happens to collide
// with some other subject's id from matching.
func (r *repository) ListByThread(
	ctx context.Context,
	req repositories.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	log := r.l.With(
		zap.String("operation", "ListByThread"),
		zap.String("threadId", req.ThreadID.String()),
	)

	entities := make([]*agent.AgentProposal, 0)
	cols := buncolgen.AgentProposalColumns
	runCols := buncolgen.AgentRunColumns

	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Join("JOIN agent_runs AS ar ON "+
			runCols.ID.EqColumn(cols.RunID)+" AND "+
			runCols.OrganizationID.EqColumn(cols.OrganizationID)+" AND "+
			runCols.BusinessUnitID.EqColumn(cols.BusinessUnitID),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo).
				Where(runCols.SubjectType.Eq(), agent.SubjectAssistantThread).
				Where(runCols.SubjectID.Eq(), req.ThreadID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to list proposals for thread", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
