package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
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

	result, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
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

	proposal := &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		ToolName:       "create_dashboard",
		Status:         agent.ProposalStatusExecutionFailed,
		ExecutionError: "tiles[0].definitionId: This report is not available.\nwrapped: cause",
	}
	svc, conversations, _ := followUpService(t, proposal)
	actor := testActor()

	_, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:           conversations.thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Contains(t, conversations.appended[0].Content,
		"but it failed when it ran: tiles[0].definitionId: This report is not available.")
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
		_, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
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
