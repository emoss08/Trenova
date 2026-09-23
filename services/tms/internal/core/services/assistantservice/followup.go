package assistantservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// maxFollowUpErrorChars bounds how much of an execution error the note
// carries. The reason a write failed is its first line; the rest is a stack of
// wrapped causes the person will not read. A sentence split is no good here:
// "tiles[0].definitionId" would end at its first dot.
const maxFollowUpErrorChars = 400

// followUpInstruction is what the agent is asked to do with a decision.
const followUpInstruction = "Tell the person in one or two sentences what happened and the " +
	"most useful next step. Do not propose the same change again."

type decisionNoteParams struct {
	thread     *conversation.Thread
	proposalID pulid.ID
	planID     pulid.ID
	history    []conversation.Message
	tenant     pagination.TenantInfo
}

// decisionNote writes the input of the turn that follows a decision on one of
// the thread's proposals.
//
// An approval used to end the conversation: the card changed to "done" and the
// agent said nothing, so the person could not tell whether the dashboard they
// approved existed or where to find it. The note tells the agent what was
// decided and how it went, and the turn it starts is the agent saying so.
//
// Its first line is the person-readable record of the decision, which is what
// the thread shows in place of a message they did not type.
func (s *Service) decisionNote(ctx context.Context, p decisionNoteParams) (string, error) {
	if p.planID.IsNotNil() {
		return s.planDecisionNote(ctx, p)
	}
	if s.proposals == nil {
		return "", errortypes.NewBusinessError("Decisions cannot be followed up here")
	}

	stored, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   p.thread.ID,
		TenantInfo: p.tenant,
	})
	if err != nil {
		return "", err
	}

	var proposal *agent.AgentProposal
	for _, candidate := range stored {
		if candidate != nil && candidate.ID == p.proposalID {
			proposal = candidate
			break
		}
	}
	if proposal == nil {
		return "", errortypes.NewNotFoundError("That proposal is not part of this conversation")
	}

	multiErr := errortypes.NewMultiError()
	if proposal.Status == agent.ProposalStatusPending {
		multiErr.Add("followUpProposalId", errortypes.ErrInvalid,
			"That proposal has not been decided yet")
	}
	if alreadyFollowedUp(p.history, "proposal "+proposal.ID.String()) {
		multiErr.Add("followUpProposalId", errortypes.ErrDuplicate,
			"That decision has already been answered")
	}
	if multiErr.HasErrors() {
		return "", multiErr
	}

	return fmt.Sprintf("%s\nDecision on proposal %s (%s). %s%s",
		decisionLine(proposal), proposal.ID, proposal.ToolName, producedNote(proposal),
		followUpInstruction), nil
}

// producedNote names the ids an executed proposal's write produced, for the
// agent rather than the person, so it is on the note's second line. Without
// it the only id the agent held was the proposal's, and it passed that to
// describe_report in place of the report it had just saved.
func producedNote(proposal *agent.AgentProposal) string {
	if proposal.Status != agent.ProposalStatusExecuted {
		return ""
	}

	note := proposal.ExecutionResult.IDNote()
	if note == "" {
		return ""
	}

	return note + " "
}

// decisionLine says in one line what was decided and how it went.
func decisionLine(proposal *agent.AgentProposal) string {
	tool := proposal.ToolName
	switch proposal.Status {
	case agent.ProposalStatusExecuted:
		if made := proposal.ExecutionResult.Describe(); made != "" {
			return fmt.Sprintf("Approved %s, and it ran. %s", tool, made)
		}
		return fmt.Sprintf("Approved %s, and it ran.", tool)
	case agent.ProposalStatusExecutionFailed:
		reason, _, _ := strings.Cut(strings.TrimSpace(proposal.ExecutionError), "\n")
		reason = stringutils.TruncateRunes(reason, maxFollowUpErrorChars)
		if reason == "" {
			return fmt.Sprintf("Approved %s, but it failed when it ran.", tool)
		}
		return fmt.Sprintf("Approved %s, but it failed when it ran: %s", tool, reason)
	case agent.ProposalStatusAccepted:
		return fmt.Sprintf("Approved %s; it is being carried out.", tool)
	case agent.ProposalStatusModified:
		return fmt.Sprintf("Approved %s with changes; it is being carried out.", tool)
	case agent.ProposalStatusRejected:
		return fmt.Sprintf("Rejected %s.", tool)
	case agent.ProposalStatusExpired:
		return fmt.Sprintf("%s expired before anyone decided it.", tool)
	case agent.ProposalStatusSuperseded:
		return fmt.Sprintf("%s was replaced by a later proposal.", tool)
	default:
		return fmt.Sprintf("%s is now %s.", tool, proposal.Status)
	}
}

// planDecisionNote writes the input of the turn that follows a decision on
// one of the thread's plans: what was decided, and how far the steps got.
func (s *Service) planDecisionNote(ctx context.Context, p decisionNoteParams) (string, error) {
	if s.plans == nil {
		return "", errortypes.NewBusinessError("Decisions cannot be followed up here")
	}

	plans, err := s.plans.ListByThread(ctx, repositories.ListAgentPlansByThreadRequest{
		ThreadID:   p.thread.ID,
		TenantInfo: p.tenant,
	})
	if err != nil {
		return "", err
	}

	var plan *agent.AgentPlan
	for _, candidate := range plans {
		if candidate != nil && candidate.ID == p.planID {
			plan = candidate
			break
		}
	}
	if plan == nil {
		return "", errortypes.NewNotFoundError("That plan is not part of this conversation")
	}

	multiErr := errortypes.NewMultiError()
	if plan.Status == agent.PlanStatusPending {
		multiErr.Add("followUpPlanId", errortypes.ErrInvalid, "That plan has not been decided yet")
	}
	if alreadyFollowedUp(p.history, "plan "+plan.ID.String()) {
		multiErr.Add("followUpPlanId", errortypes.ErrDuplicate,
			"That decision has already been answered")
	}
	if multiErr.HasErrors() {
		return "", multiErr
	}

	return fmt.Sprintf("%s\nDecision on plan %s (%d steps). %s",
		planDecisionLine(plan), plan.ID, plan.StepCount, followUpInstruction), nil
}

// planDecisionLine says in one line what was decided on a plan and how far
// its steps got.
func planDecisionLine(plan *agent.AgentPlan) string {
	title := plan.Title
	switch plan.Status {
	case agent.PlanStatusCompleted:
		return fmt.Sprintf("Approved the plan %q, and all %d steps ran.", title, plan.StepCount)
	case agent.PlanStatusFailed:
		reason, _, _ := strings.Cut(strings.TrimSpace(plan.FailureError), "\n")
		reason = stringutils.TruncateRunes(reason, maxFollowUpErrorChars)
		step := plan.CompletedSteps + 1
		if plan.FailedStep != nil {
			step = *plan.FailedStep
		}
		if reason == "" {
			return fmt.Sprintf("Approved the plan %q, but step %d failed and the steps after it "+
				"were skipped.", title, step)
		}
		return fmt.Sprintf("Approved the plan %q, but step %d failed (%s) and the steps after "+
			"it were skipped.", title, step, reason)
	case agent.PlanStatusApproved:
		return fmt.Sprintf("Approved the plan %q; %d of %d steps have run so far.",
			title, plan.CompletedSteps, plan.StepCount)
	case agent.PlanStatusRejected:
		return fmt.Sprintf("Rejected the plan %q.", title)
	case agent.PlanStatusExpired:
		return fmt.Sprintf("The plan %q expired before anyone decided it.", title)
	default:
		return fmt.Sprintf("The plan %q is now %s.", title, plan.Status)
	}
}

// alreadyFollowedUp reports whether the thread already carries the note for
// this decision, so a double click or a retried request cannot start two
// turns about one decision.
func alreadyFollowedUp(history []conversation.Message, marker string) bool {
	for idx := range history {
		message := &history[idx]
		if message.Kind == conversation.MessageKindDecisionNote &&
			strings.Contains(message.Content, marker) {
			return true
		}
	}

	return false
}

// decisionHeadline is the person-readable record of a decision: the note's
// first line, which is all of it a reader is shown. The lines after it are
// instructions to the agent, and no rendering of the thread shows them.
func decisionHeadline(content string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(content), "\n")

	return strings.TrimSpace(line)
}

// markDecisionNote marks the turn's input as the application's note rather
// than words the person typed.
func markDecisionNote(messages []conversation.Message) {
	for idx := range messages {
		if messages[idx].Role == conversation.RoleUser {
			messages[idx].Kind = conversation.MessageKindDecisionNote
			return
		}
	}
}
