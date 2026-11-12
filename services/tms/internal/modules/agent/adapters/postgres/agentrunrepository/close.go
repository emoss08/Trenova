package agentrunrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const (
	openSubjectConstraint = "uq_agent_runs_open_subject"
	closeSettledReason    = "close agent runs whose proposals were all decided or expired, across every organization"
)

// CloseDecided completes a run that waits on decisions once none of its
// proposals is pending. The condition is the statement's own, so the last
// decision and the expiry sweep racing for the same run close it once.
func (r *repository) CloseDecided(
	ctx context.Context,
	req *agentruntimeports.CloseDecidedRunRequest,
) (*agent.AgentRun, error) {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		rows, err := run.Where(
			run.ID.Eq(req.RunID),
			awaitingWithNothingPending(run, false),
		).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			run.Status.To(agent.RunStatusCompleted),
			run.CompletedAt.To(req.Now),
		)
		if err != nil {
			return nil, runErrors.Map(err, req.RunID.String())
		}
		if len(rows) == 0 {
			return nil, nil
		}

		return rows[0], nil
	})
}

// CloseSettled completes, across every organization, the runs that wait on
// decisions with nothing left pending: their last proposals expired.
func (r *repository) CloseSettled(
	ctx context.Context,
	req *agentruntimeports.CloseSettledRunsRequest,
) ([]*agent.AgentRun, error) {
	ctx = dbscope.WithSystem(ctx, closeSettledReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		settled := run.Pick(run.ID).
			Where(awaitingWithNothingPending(run, true)).
			Order(run.UpdatedAt.Asc(), run.ID.Asc()).
			Limit(req.Limit).
			AllTenants()

		rows, err := run.AllTenants().
			Where(run.ID.InQuery(settled), awaitingWithNothingPending(run, true)).
			UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				run.Status.To(agent.RunStatusCompleted),
				run.CompletedAt.To(req.Now),
			)
		if err != nil {
			return nil, err
		}

		return rows, nil
	})
}

// awaitingWithNothingPending is a run that closes when decided
// (agent.AgentRun.ClosesWhenDecided) and waits on decisions none of which is
// still pending.
func awaitingWithNothingPending(run *agentdb.AgentRunsTable, allTenants bool) keel.Pred {
	p := agentdb.AgentProposals
	pending := p.Pick(p.ID).Where(
		p.RunID.EqCol(run.ID),
		p.OrganizationID.EqCol(run.OrganizationID),
		p.BusinessUnitID.EqCol(run.BusinessUnitID),
		p.Status.Eq(agent.ProposalStatusPending),
	)
	if allTenants {
		pending = pending.AllTenants()
	}

	return keel.And(
		run.Status.Eq(agent.RunStatusAwaitingDecision),
		keel.Or(run.WorkflowID.IsNotNull(), run.Trigger.Eq(agent.RunTriggerEvent)),
		keel.Not(keel.Exists(pending)),
	)
}

// FailWaiting fails a run that waits on decisions, with the reason it was
// stopped. It returns nil when the run was not waiting.
func (r *repository) FailWaiting(
	ctx context.Context,
	req *agentruntimeports.FailWaitingRunRequest,
) (*agent.AgentRun, error) {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		rows, err := run.Where(
			run.ID.Eq(req.RunID),
			run.Status.Eq(agent.RunStatusAwaitingDecision),
		).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			run.Status.To(agent.RunStatusFailed),
			run.ErrorMessage.To(req.Reason),
			run.CompletedAt.To(req.Now),
		)
		if err != nil {
			return nil, runErrors.Map(err, req.RunID.String())
		}
		if len(rows) == 0 {
			return nil, nil
		}

		return rows[0], nil
	})
}
