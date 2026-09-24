package assistantfollowupservice

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// resumeWindow bounds how old a decision may be and still be reported late.
// A decision a conversation was never told about from before follow-ups
// existed is history, not news, and reporting it now would read as a reply
// to nothing.
const resumeWindow = 24 * time.Hour

type proposalLister interface {
	ListByThread(
		ctx context.Context,
		req repositories.ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
}

type planLister interface {
	ListByThread(
		ctx context.Context,
		req repositories.ListAgentPlansByThreadRequest,
	) ([]*agent.AgentPlan, error)
}

type decisionLister interface {
	ListByProposals(
		ctx context.Context,
		req repositories.ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
}

// decidedReader reads what a conversation raised that a person has decided.
type decidedReader struct {
	proposals proposalLister
	plans     planLister
	decisions decisionLister
}

func (r decidedReader) ready() bool {
	return r.proposals != nil && r.plans != nil && r.decisions != nil
}

// unreported is a decision the conversation has not been told about.
type unreported struct {
	runID      pulid.ID
	proposalID pulid.ID
	planID     pulid.ID
	decidedAt  int64
}

// ResumeFollowUps starts the follow-up for the oldest decision on something
// this conversation raised that it has not reported, if there is one.
//
// It is asked when a turn ends, because a turn in progress is the one thing
// that keeps a follow-up from starting. One follow-up is started at a time;
// when it ends it asks again, so decisions made in a burst are reported in
// the order they were made.
func (s *Service) ResumeFollowUps(ctx context.Context, req serviceports.ResumeFollowUpsRequest) {
	if !s.decided.ready() || req.ThreadID.IsNil() {
		return
	}

	ctx = context.WithoutCancel(ctx)
	tenant := pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID}

	next, err := s.oldestUnreported(ctx, req.ThreadID, tenant)
	if err != nil {
		s.l.Warn("could not read the conversation's decisions to report",
			zap.String("thread", req.ThreadID.String()),
			zap.Error(err),
		)
		return
	}
	if next == nil {
		return
	}

	s.FollowUp(ctx, serviceports.DecisionFollowUpRequest{
		TenantInfo: tenant,
		RunID:      next.runID,
		ProposalID: next.proposalID,
		PlanID:     next.planID,
	})
}

func (s *Service) oldestUnreported(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) (*unreported, error) {
	cutoff := timeutils.NowUnix() - int64(resumeWindow/time.Second)

	candidates, err := s.decidedProposals(ctx, threadID, tenant, cutoff)
	if err != nil {
		return nil, err
	}

	plans, err := s.decided.plans.ListByThread(ctx, repositories.ListAgentPlansByThreadRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		if plan == nil || !planAwaitsReport(plan.Status) ||
			plan.DecidedAt == nil || *plan.DecidedAt < cutoff {
			continue
		}
		candidates = append(candidates, unreported{
			runID:     plan.RunID,
			planID:    plan.ID,
			decidedAt: *plan.DecidedAt,
		})
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	notes, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
		Kinds:      []conversation.MessageKind{conversation.MessageKindDecisionNote},
	})
	if err != nil {
		return nil, err
	}

	candidates = slices.DeleteFunc(candidates, func(candidate unreported) bool {
		return assistantservice.DecisionReported(notes, candidate.proposalID, candidate.planID)
	})
	if len(candidates) == 0 {
		return nil, nil
	}

	oldest := slices.MinFunc(candidates, func(a, b unreported) int {
		return cmp.Compare(a.decidedAt, b.decidedAt)
	})

	return &oldest, nil
}

// decidedProposals are the proposals this conversation raised on their own
// that a person decided since the cut-off, and that have come to an end.
// A plan's steps are reported by the plan. A proposal that ran on its own was
// never decided, and one still being carried out is reported by the decision
// that is carrying it out, once it has.
func (s *Service) decidedProposals(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
	cutoff int64,
) ([]unreported, error) {
	proposals, err := s.decided.proposals.ListByThread(
		ctx,
		repositories.ListAgentProposalsByThreadRequest{ThreadID: threadID, TenantInfo: tenant},
	)
	if err != nil {
		return nil, err
	}

	ended := make(map[pulid.ID]*agent.AgentProposal, len(proposals))
	ids := make([]pulid.ID, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal == nil || proposal.PlanID != nil || !proposalAwaitsReport(proposal.Status) {
			continue
		}
		ended[proposal.ID] = proposal
		ids = append(ids, proposal.ID)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	decisions, err := s.decided.decisions.ListByProposals(
		ctx,
		repositories.ListAgentDecisionsByProposalsRequest{ProposalIDs: ids, TenantInfo: tenant},
	)
	if err != nil {
		return nil, err
	}

	out := make([]unreported, 0, len(ids))
	for _, decision := range decisions {
		if decision == nil || decision.ProposalID == nil || decision.CreatedAt < cutoff {
			continue
		}
		proposal, ok := ended[*decision.ProposalID]
		if !ok {
			continue
		}
		delete(ended, proposal.ID)
		out = append(out, unreported{
			runID:      proposal.RunID,
			proposalID: proposal.ID,
			decidedAt:  decision.CreatedAt,
		})
	}

	return out, nil
}

func proposalAwaitsReport(status agent.ProposalStatus) bool {
	switch status {
	case agent.ProposalStatusExecuted,
		agent.ProposalStatusExecutionFailed,
		agent.ProposalStatusRejected,
		agent.ProposalStatusSimulated:
		return true
	default:
		return false
	}
}

func planAwaitsReport(status agent.PlanStatus) bool {
	switch status {
	case agent.PlanStatusCompleted, agent.PlanStatusFailed, agent.PlanStatusRejected:
		return true
	default:
		return false
	}
}
