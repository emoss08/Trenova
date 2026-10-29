package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/decisionnotes"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func followUpService(
	t *testing.T,
	proposal *agent.AgentProposal,
	history ...conversation.Message,
) (*Service, *assistanttest.StubConversations, *assistanttest.ScriptedCompletion) {
	t.Helper()

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("The Operations dashboard is ready under Reports."),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	svc.proposals = &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{proposal}}
	conversations.Messages = history

	return svc, conversations, completion
}

/*
An approval is answered.

The Report Builder thread ended at "Approved": the card flipped to done and the
agent said nothing, so the person could not tell whether the report existed or
where it was. The client now asks for the turn that follows a decision, and the
agent is told what was decided and how it went.
*/
func TestSendMessageStream_AnswersADecision(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "report_create_dashboard",
		Status:   agent.ProposalStatusExecuted,
	}
	svc, conversations, completion := followUpService(t, proposal)
	actor := assistanttest.Actor()

	result, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(t, "The Operations dashboard is ready under Reports.", result.Reply)
	note := conversations.Appended[0]
	assert.Equal(t, conversation.MessageKindDecisionNote, note.Kind,
		"the thread shows a note, not words the person typed")
	assert.Contains(t, note.Content, "Approved report_create_dashboard, and it ran.")
	assert.Contains(t, note.Content, proposal.ID.String())

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1]
	assert.Contains(t, last.Content, "what happened")
	assert.Equal(t, "Dispatch", conversations.Thread.Title, "a note does not retitle the thread")
}

func TestSendMessageStream_SaysWhyAnApprovedChangeFailed(t *testing.T) {
	t.Parallel()

	refused := errortypes.NewMultiError()
	refused.Add("bol", errortypes.ErrDuplicate,
		"BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009")
	refused.Add("formulaTemplateId", errortypes.ErrRequired,
		"Choose a rating method: no rate agreement covers this lane")
	proposal := &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		ToolName:       "shipment_create",
		Status:         agent.ProposalStatusExecutionFailed,
		ExecutionError: refused.Error(),
	}
	svc, conversations, _ := followUpService(t, proposal)
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	headline := decisionnotes.DecisionHeadline(conversations.Appended[0].Content)
	assert.Equal(t,
		"Approved shipment_create, but it failed when it ran: validation failed: "+
			"bol: BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009; "+
			"formulaTemplateId: Choose a rating method: no rate agreement covers this lane",
		headline,
		"the person reads why, not a bare \"validation failed:\"")
}

func TestSendMessageStream_RefusesAFollowUpThatCannotBeAnswered(t *testing.T) {
	t.Parallel()

	decided := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "report_create_dashboard",
		Status:   agent.ProposalStatusRejected,
	}
	pending := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "report_create_dashboard",
		Status:   agent.ProposalStatusPending,
	}
	actor := assistanttest.Actor()
	send := func(svc *Service, threadID, proposalID pulid.ID, content string) error {
		_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
			ThreadID:           threadID,
			Content:            content,
			FollowUpProposalID: proposalID,
			TenantInfo:         actor.TenantInfo(),
		}, actor, nil)
		return err
	}

	svc, conversations, _ := followUpService(t, pending)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, send(svc, conversations.Thread.ID, pending.ID, ""), &multiErr,
		"a proposal nobody has decided has no outcome to report")

	svc, conversations, _ = followUpService(t, decided)
	require.Error(t, send(svc, conversations.Thread.ID, pulid.MustNew("ap_"), ""),
		"a proposal from another thread is not this thread's to answer")
	require.Error(t, send(svc, conversations.Thread.ID, decided.ID, "and also this"),
		"a follow-up carries no words of its own")

	svc, conversations, completion := followUpService(t, decided, conversation.Message{
		Role:    conversation.RoleUser,
		Kind:    conversation.MessageKindDecisionNote,
		Content: "Rejected report_create_dashboard.\nDecision on proposal " + decided.ID.String(),
	})
	require.ErrorAs(t, send(svc, conversations.Thread.ID, decided.ID, ""), &multiErr,
		"a decision is answered once, however many times the client asks")
	assert.Nil(t, completion.LastReq)
}

type stubPlanStore struct {
	plans []*agentplan.AgentPlan
}

func (s *stubPlanStore) ListByThread(
	_ context.Context,
	_ agentports.ListAgentPlansByThreadRequest,
) ([]*agentplan.AgentPlan, error) {
	return s.plans, nil
}

/*
A plan is answered once, as a plan.

Approving a plan decides every step, and each step used to be a proposal the
client could ask about on its own. The server now asks once, for the plan,
after its last step, and the agent is told how far the steps got.
*/
func TestSendMessageStream_AnswersAPlanDecision(t *testing.T) {
	t.Parallel()

	failedStep := 2
	plan := &agentplan.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		Title:          "Recover load 4471",
		Status:         agentplan.PlanStatusFailed,
		StepCount:      3,
		CompletedSteps: 1,
		FailedStep:     &failedStep,
		FailureError:   "the driver is out of hours\nwrapped: cause",
	}
	svc, conversations, _ := followUpService(t, &agent.AgentProposal{ID: pulid.MustNew("ap_")})
	svc.plans = &stubPlanStore{plans: []*agentplan.AgentPlan{plan}}
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.Thread.ID,
		FollowUpPlanID: plan.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	note := conversations.Appended[0]
	assert.Equal(t, conversation.MessageKindDecisionNote, note.Kind)
	assert.Contains(t, note.Content,
		`Approved the plan "Recover load 4471", but step 2 failed (the driver is out of hours)`)
	assert.Contains(t, note.Content, "plan "+plan.ID.String())
}

// A follow-up names one decision. A plan still waiting has nothing to report,
// and a request naming a proposal and a plan at once is refused.
func TestSendMessageStream_RefusesAPlanFollowUpThatCannotBeAnswered(t *testing.T) {
	t.Parallel()

	pending := &agentplan.AgentPlan{ID: pulid.MustNew("apl_"), Status: agentplan.PlanStatusPending}
	svc, conversations, completion := followUpService(t, &agent.AgentProposal{
		ID:     pulid.MustNew("ap_"),
		Status: agent.ProposalStatusExecuted,
	})
	svc.plans = &stubPlanStore{plans: []*agentplan.AgentPlan{pending}}
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.Thread.ID,
		FollowUpPlanID: pending.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	_, err = svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpPlanID:     pending.ID,
		FollowUpProposalID: pulid.MustNew("ap_"),
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.ErrorAs(t, err, &multiErr)
	assert.Nil(t, completion.LastReq)
}

func (s *stubPlanStore) ExpirePendingByThread(
	context.Context,
	agentports.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return 0, nil
}
