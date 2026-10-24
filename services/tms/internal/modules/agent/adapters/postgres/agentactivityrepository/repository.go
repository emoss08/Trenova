package agentactivityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) agentports.AgentActivityRepository {
	return &repository{db: p.DB}
}

type runTotals struct {
	Runs     int64
	Failed   int64
	Working  int64
	Awaiting int64
}

type proposalTotals struct {
	Pending       int64
	OldestPending *int64
}

type decisionTotals struct {
	Accepted int64
	Modified int64
	Rejected int64
}

func workingStatuses() []agent.RunStatus {
	return []agent.RunStatus{
		agent.RunStatusPending,
		agent.RunStatusGatheringContext,
		agent.RunStatusDiagnosing,
	}
}

func openResolutionStates() []agent.ResolutionState {
	return []agent.ResolutionState{agent.ResolutionStateOpen, agent.ResolutionStateInReview}
}

func (r *repository) Totals(
	ctx context.Context,
	req agentports.AgentActivityTotalsRequest,
) (*agentports.AgentActivityTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentports.AgentActivityTotals, error) {
		runs := agentdb.AgentRuns
		proposals := agentdb.AgentProposals
		exceptions := agentdb.AgentExceptions
		decisions := agentdb.AgentDecisions

		var (
			runRow      runTotals
			proposalRow proposalTotals
			openCount   int64
			decisionRow decisionTotals
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(runs.Pick(
				keel.Count().Filter(runs.CreatedAt.Gte(req.RunsSince)),
				keel.Count().Filter(keel.And(
					runs.CreatedAt.Gte(req.RunsSince),
					runs.Status.Eq(agent.RunStatusFailed),
				)),
				keel.Count().Filter(runs.Status.In(workingStatuses()...)),
				keel.Count().Filter(runs.Status.Eq(agent.RunStatusAwaitingDecision)),
			), &runRow),
			keel.IntoValue(proposals.
				Pick(keel.Count(), proposals.CreatedAt.Min()).
				Where(proposals.Status.Eq(agent.ProposalStatusPending)), &proposalRow),
			keel.CountInto(
				exceptions.Where(exceptions.ResolutionState.In(openResolutionStates()...)),
				&openCount,
			),
			keel.IntoValue(decisions.Pick(
				keel.Count().Filter(decisions.Decision.Eq(agent.DecisionAccepted)),
				keel.Count().Filter(decisions.Decision.Eq(agent.DecisionModified)),
				keel.Count().Filter(decisions.Decision.Eq(agent.DecisionRejected)),
			).Where(
				decisions.ProposalID.IsNotNull(),
				decisions.UndoneAt.IsNull(),
				decisions.CreatedAt.Gte(req.DecisionsSince),
			), &decisionRow),
		)
		if err != nil {
			return nil, fmt.Errorf("count agent activity: %w", err)
		}

		return &agentports.AgentActivityTotals{
			Runs:              int(runRow.Runs),
			RunsFailed:        int(runRow.Failed),
			RunsWorking:       int(runRow.Working),
			RunsAwaiting:      int(runRow.Awaiting),
			PendingProposals:  int(proposalRow.Pending),
			OldestPendingAt:   proposalRow.OldestPending,
			OpenExceptions:    int(openCount),
			DecisionsAccepted: int(decisionRow.Accepted),
			DecisionsModified: int(decisionRow.Modified),
			DecisionsRejected: int(decisionRow.Rejected),
		}, nil
	})
}
