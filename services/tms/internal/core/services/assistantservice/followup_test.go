package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func followUpService(
	t *testing.T,
	proposal *agent.AgentProposal,
	history ...conversation.Message,
) (*Service, *stubConversations, *scriptedCompletion) {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("The Operations dashboard is ready under Reports."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	svc.proposals = &stubProposalRepo{byThread: []*agent.AgentProposal{proposal}}
	conversations.messages = history

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
		ToolName: "create_dashboard",
		Status:   agent.ProposalStatusExecuted,
	}
	svc, conversations, completion := followUpService(t, proposal)
	actor := testActor()

	result, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(t, "The Operations dashboard is ready under Reports.", result.Reply)
	note := conversations.appended[0]
	assert.Equal(t, conversation.MessageKindDecisionNote, note.Kind,
		"the thread shows a note, not words the person typed")
	assert.Contains(t, note.Content, "Approved create_dashboard, and it ran.")
	assert.Contains(t, note.Content, proposal.ID.String())

	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1]
	assert.Contains(t, last.Content, "what happened")
	assert.Equal(t, "Dispatch", conversations.thread.Title, "a note does not retitle the thread")
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
		ToolName:       "create_shipment",
		Status:         agent.ProposalStatusExecutionFailed,
		ExecutionError: refused.Error(),
	}
	svc, conversations, _ := followUpService(t, proposal)
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	headline := decisionHeadline(conversations.appended[0].Content)
	assert.Equal(t,
		"Approved create_shipment, but it failed when it ran: validation failed: "+
			"bol: BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009; "+
			"formulaTemplateId: Choose a rating method: no rate agreement covers this lane",
		headline,
		"the person reads why, not a bare \"validation failed:\"")
}

func TestExecutionFailureReason_KeepsTheListedProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "one line with wrapped causes after it",
			text: "tiles[0].definitionId: This report is not available.\nwrapped: cause",
			want: "tiles[0].definitionId: This report is not available.",
		},
		{
			name: "a header and its problems",
			text: "validation failed:\n- bol: taken\n- weight: too heavy",
			want: "validation failed: bol: taken; weight: too heavy",
		},
		{
			name: "no more than five problems",
			text: "validation failed:\n- a: 1\n- b: 2\n- c: 3\n- d: 4\n- e: 5\n- f: 6\n- g: 7",
			want: "validation failed: a: 1; b: 2; c: 3; d: 4; e: 5; and 2 more",
		},
		{
			name: "problems end at the first line that is not one",
			text: "validation failed:\n- a: 1\ncaused by: x\n- b: 2",
			want: "validation failed: a: 1",
		},
		{
			name: "nothing",
			text: "  ",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, executionFailureReason(tt.text))
		})
	}
}

func TestSendMessageStream_RefusesAFollowUpThatCannotBeAnswered(t *testing.T) {
	t.Parallel()

	decided := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "create_dashboard",
		Status:   agent.ProposalStatusRejected,
	}
	pending := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "create_dashboard",
		Status:   agent.ProposalStatusPending,
	}
	actor := testActor()
	send := func(svc *Service, threadID, proposalID pulid.ID, content string) error {
		_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
			ThreadID:           threadID,
			Content:            content,
			FollowUpProposalID: proposalID,
			TenantInfo:         actor.TenantInfo(),
		}, actor, nil)
		return err
	}

	svc, conversations, _ := followUpService(t, pending)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, send(svc, conversations.thread.ID, pending.ID, ""), &multiErr,
		"a proposal nobody has decided has no outcome to report")

	svc, conversations, _ = followUpService(t, decided)
	require.Error(t, send(svc, conversations.thread.ID, pulid.MustNew("ap_"), ""),
		"a proposal from another thread is not this thread's to answer")
	require.Error(t, send(svc, conversations.thread.ID, decided.ID, "and also this"),
		"a follow-up carries no words of its own")

	svc, conversations, completion := followUpService(t, decided, conversation.Message{
		Role:    conversation.RoleUser,
		Kind:    conversation.MessageKindDecisionNote,
		Content: "Rejected create_dashboard.\nDecision on proposal " + decided.ID.String(),
	})
	require.ErrorAs(t, send(svc, conversations.thread.ID, decided.ID, ""), &multiErr,
		"a decision is answered once, however many times the client asks")
	assert.Nil(t, completion.LastReq)
}

type stubPlanStore struct {
	plans []*agent.AgentPlan
}

func (s *stubPlanStore) ListByThread(
	_ context.Context,
	_ repositories.ListAgentPlansByThreadRequest,
) ([]*agent.AgentPlan, error) {
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
	plan := &agent.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		Title:          "Recover load 4471",
		Status:         agent.PlanStatusFailed,
		StepCount:      3,
		CompletedSteps: 1,
		FailedStep:     &failedStep,
		FailureError:   "the driver is out of hours\nwrapped: cause",
	}
	svc, conversations, _ := followUpService(t, &agent.AgentProposal{ID: pulid.MustNew("ap_")})
	svc.plans = &stubPlanStore{plans: []*agent.AgentPlan{plan}}
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:       conversations.thread.ID,
		FollowUpPlanID: plan.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	note := conversations.appended[0]
	assert.Equal(t, conversation.MessageKindDecisionNote, note.Kind)
	assert.Contains(t, note.Content,
		`Approved the plan "Recover load 4471", but step 2 failed (the driver is out of hours)`)
	assert.Contains(t, note.Content, "plan "+plan.ID.String())
}

// A follow-up names one decision. A plan still waiting has nothing to report,
// and a request naming a proposal and a plan at once is refused.
func TestSendMessageStream_RefusesAPlanFollowUpThatCannotBeAnswered(t *testing.T) {
	t.Parallel()

	pending := &agent.AgentPlan{ID: pulid.MustNew("apl_"), Status: agent.PlanStatusPending}
	svc, conversations, completion := followUpService(t, &agent.AgentProposal{
		ID:     pulid.MustNew("ap_"),
		Status: agent.ProposalStatusExecuted,
	})
	svc.plans = &stubPlanStore{plans: []*agent.AgentPlan{pending}}
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:       conversations.thread.ID,
		FollowUpPlanID: pending.ID,
		TenantInfo:     actor.TenantInfo(),
	}, actor, nil)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	_, err = svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpPlanID:     pending.ID,
		FollowUpProposalID: pulid.MustNew("ap_"),
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.ErrorAs(t, err, &multiErr)
	assert.Nil(t, completion.LastReq)
}

func TestFollowUpInstruction_KeepsTheReplyToWhatTheNoteAndTheCardHold(t *testing.T) {
	t.Parallel()

	assert.Contains(t, followUpInstruction,
		"Report only what this note and the proposal's card hold")
	assert.Contains(t, followUpInstruction, "Do not propose the same change again.")
}

func (s *stubPlanStore) ExpirePendingByThread(
	context.Context,
	repositories.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return 0, nil
}

// "Approved update_shipment, and it ran" left the agent nothing to say, and
// gpt-6-luna reported the weight change without the weight. A write that
// reports no result says what it ran with, the approver's changes winning.
func TestRanWith_NamesTheApprovedValuesOfAWriteWithNoResult(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		Status:   agent.ProposalStatusExecuted,
		ToolName: "update_shipment",
		ToolParams: map[string]any{
			"shipmentId": "shp_1",
			"weight":     float64(38500),
			"pieces":     float64(10),
			"_why":       map[string]any{"because": "the person said so"},
		},
	}
	changed := []*agent.AgentDecision{{
		ProposalID:    &proposal.ID,
		Modifications: map[string]any{"pieces": float64(12)},
	}}

	got := ranWith(proposal, changed)
	assert.Equal(t, " It ran with pieces: 12; weight: 38500.", got)

	proposal.ExecutionResult = &agent.ToolExecutionResult{Action: "created", Kind: "report", Name: "On-time"}
	assert.Empty(t, ranWith(proposal, nil), "a write that describes its own result says that instead")

	proposal.ExecutionResult = nil
	proposal.Status = agent.ProposalStatusRejected
	assert.Empty(t, ranWith(proposal, nil))
}
