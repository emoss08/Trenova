package decisionnotes

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationworkingset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordanchorports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Notes struct {
	Anchors   recordanchorports.RecordAnchorReader
	Proposals proposalstores.ProposalStore
	Plans     proposalstores.PlanStore
	Decisions agentdecisionports.AgentDecisionRepository
}

// DecisionNote writes the input of the turn that follows a decision on one of
// the thread's proposals.
//
// An approval used to end the conversation: the card changed to "done" and the
// agent said nothing, so the person could not tell whether the dashboard they
// approved existed or where to find it. The note tells the agent what was
// decided and how it went, and the turn it starts is the agent saying so.
//
// Its first line is the person-readable record of the decision, which is what
// the thread shows in place of a message they did not type.
func (s *Notes) DecisionNote(
	ctx context.Context,
	p *DecisionNoteParams,
) (decisionFollowUp, error) {
	if p.PlanID.IsNotNil() {
		return s.planDecisionNote(ctx, p)
	}
	if s.Proposals == nil {
		return decisionFollowUp{}, errortypes.NewBusinessError(
			"Decisions cannot be followed up here",
		)
	}

	stored, err := s.Proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   p.Thread.ID,
			TenantInfo: p.Tenant,
		},
	)
	if err != nil {
		return decisionFollowUp{}, err
	}

	var proposal *agent.AgentProposal
	for _, candidate := range stored {
		if candidate != nil && candidate.ID == p.ProposalID {
			proposal = candidate
			break
		}
	}
	if proposal == nil {
		return decisionFollowUp{}, errortypes.NewNotFoundError(
			"That proposal is not part of this conversation",
		)
	}

	multiErr := errortypes.NewMultiError()
	if proposal.Status == agent.ProposalStatusPending {
		multiErr.Add("followUpProposalId", errortypes.ErrInvalid,
			"That proposal has not been decided yet")
	}
	if alreadyFollowedUp(p.History, "proposal "+proposal.ID.String()) {
		multiErr.Add("followUpProposalId", errortypes.ErrDuplicate,
			"That decision has already been answered")
	}
	if multiErr.HasErrors() {
		return decisionFollowUp{}, multiErr
	}

	decisions, err := s.decisionsOn(ctx, p.Tenant, []pulid.ID{proposal.ID})
	if err != nil {
		return decisionFollowUp{}, err
	}
	note := noteFrom(decisions, []pulid.ID{proposal.ID})
	ask := followUpAsk(proposal.Status == agent.ProposalStatusRejected, note)
	refused := refusedOverChange([]*agent.AgentProposal{proposal})
	if len(refused) > 0 {
		ask = refusedAsk(note)
	}

	return decisionFollowUp{
		Note: fmt.Sprintf("%s%s\nDecision on proposal %s (%s). %s%s%s",
			decisionLine(proposal), ranWith(proposal, decisions), proposal.ID, proposal.ToolName,
			liftedHold(p.History, proposal), producedNote(proposal), ask),
		Decided: []*agent.AgentProposal{proposal},
		Changed: changedTargets(refused),
	}, nil
}

func (s *Notes) AnchorChanged(
	ctx context.Context,
	anchors []agentturn.RuntimeAnchor,
	changed []agentpagecontext.EntityRef,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) []agentturn.RuntimeAnchor {
	if s.Anchors == nil || len(changed) == 0 {
		return anchors
	}

	unread := make([]agentpagecontext.EntityRef, 0, len(changed))
	anchored := func(id string) bool {
		return slices.ContainsFunc(anchors, func(a agentturn.RuntimeAnchor) bool {
			return a.ID == id
		}) || slices.ContainsFunc(unread, func(r agentpagecontext.EntityRef) bool { return r.ID == id })
	}
	for _, ref := range changed {
		if anchored(ref.ID) {
			continue
		}
		unread = append(unread, ref)
		if len(unread) == conversationworkingset.MaxWorkingSet {
			break
		}
	}
	if len(unread) == 0 {
		return anchors
	}

	return append(anchors, s.Anchors.Anchor(ctx, &recordanchorports.RecordAnchorRequest{
		TenantInfo: tenant,
		Actor:      actor,
		Records:    unread,
	})...)
}

func (s *Notes) personNote(
	ctx context.Context,
	tenant pagination.TenantInfo,
	proposalIDs []pulid.ID,
) (string, error) {
	decisions, err := s.decisionsOn(ctx, tenant, proposalIDs)
	if err != nil {
		return "", err
	}

	return noteFrom(decisions, proposalIDs), nil
}

func (s *Notes) decisionsOn(
	ctx context.Context,
	tenant pagination.TenantInfo,
	proposalIDs []pulid.ID,
) ([]*agentdecision.AgentDecision, error) {
	if s.Decisions == nil || len(proposalIDs) == 0 {
		return nil, nil
	}

	return s.Decisions.ListByProposals(
		ctx,
		agentdecisionports.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: proposalIDs,
			TenantInfo:  tenant,
		},
	)
}

// planDecisionNote writes the input of the turn that follows a decision on
// one of the thread's plans: what was decided, and how far the steps got.
func (s *Notes) planDecisionNote(
	ctx context.Context,
	p *DecisionNoteParams,
) (decisionFollowUp, error) {
	if s.Plans == nil {
		return decisionFollowUp{}, errortypes.NewBusinessError(
			"Decisions cannot be followed up here",
		)
	}

	plans, err := s.Plans.ListByThread(ctx, agentplanports.ListAgentPlansByThreadRequest{
		ThreadID:   p.Thread.ID,
		TenantInfo: p.Tenant,
	})
	if err != nil {
		return decisionFollowUp{}, err
	}

	var plan *agentplan.AgentPlan
	for _, candidate := range plans {
		if candidate != nil && candidate.ID == p.PlanID {
			plan = candidate
			break
		}
	}
	if plan == nil {
		return decisionFollowUp{}, errortypes.NewNotFoundError(
			"That plan is not part of this conversation",
		)
	}

	multiErr := errortypes.NewMultiError()
	if plan.Status == agentplan.PlanStatusPending {
		multiErr.Add("followUpPlanId", errortypes.ErrInvalid, "That plan has not been decided yet")
	}
	if alreadyFollowedUp(p.History, "plan "+plan.ID.String()) {
		multiErr.Add("followUpPlanId", errortypes.ErrDuplicate,
			"That decision has already been answered")
	}
	if multiErr.HasErrors() {
		return decisionFollowUp{}, multiErr
	}

	steps, err := s.planSteps(ctx, p, plan.ID)
	if err != nil {
		return decisionFollowUp{}, err
	}

	stepIDs := make([]pulid.ID, 0, len(steps))
	for _, step := range steps {
		stepIDs = append(stepIDs, step.ID)
	}
	note, err := s.personNote(ctx, p.Tenant, stepIDs)
	if err != nil {
		return decisionFollowUp{}, err
	}
	line := planDecisionLine(plan, steps)
	ask := followUpAsk(plan.Status == agentplan.PlanStatusRejected, note)
	refused := refusedOverChange(steps)
	if len(refused) > 0 {
		ask = refusedAsk(note)
		if plan.Status == agentplan.PlanStatusFailed && noneRan(steps) {
			line = planRefusedLine(plan, refused)
		}
	}

	return decisionFollowUp{
		Note: fmt.Sprintf("%s\nDecision on plan %s (%d steps). %s%s",
			line, plan.ID, plan.StepCount, stepsProducedNote(steps), ask),
		Decided: steps,
		Changed: changedTargets(refused),
	}, nil
}

func (s *Notes) planSteps(
	ctx context.Context,
	p *DecisionNoteParams,
	planID pulid.ID,
) ([]*agent.AgentProposal, error) {
	if s.Proposals == nil {
		return nil, nil
	}

	stored, err := s.Proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   p.Thread.ID,
			TenantInfo: p.Tenant,
		},
	)
	if err != nil {
		return nil, err
	}

	steps := make([]*agent.AgentProposal, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil && proposal.PlanID != nil && *proposal.PlanID == planID {
			steps = append(steps, proposal)
		}
	}
	slices.SortStableFunc(steps, func(a, b *agent.AgentProposal) int {
		return cmp.Compare(a.PlanStep, b.PlanStep)
	})

	return steps, nil
}
