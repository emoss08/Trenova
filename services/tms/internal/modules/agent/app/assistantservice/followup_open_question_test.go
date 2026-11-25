package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A decision follow-up started while the agent's question to the person was
still open took the approval for the answer and started the work. From the
request a follow-up arrives with to the tool call its model makes, the turn now
only reports: the note says the question is waiting and the call is refused.
*/
func TestSendMessageStream_AFollowUpUnderAnOpenQuestionOnlyReports(t *testing.T) {
	t.Parallel()

	question := "Should I start planning drivers for the first batch of 50 Delayed shipments?"
	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: "agent_create_memory",
		Status:   agent.ProposalStatusExecuted,
	}
	svc, conversations, completion := followUpService(t, proposal,
		conversation.Message{
			ID:      pulid.MustNew("amsg_"),
			Role:    conversation.RoleUser,
			Content: "Plan drivers for the Delayed shipments",
		},
		conversation.Message{
			ID:   pulid.MustNew("amsg_"),
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_ask",
				Name:      "ask_user",
				Arguments: map[string]any{"question": question},
			}},
		},
		conversation.Message{
			ID:         pulid.MustNew("amsg_"),
			Role:       conversation.RoleTool,
			ToolCallID: "call_ask",
			ToolName:   "ask_user",
			Content:    "Asked the person.",
		},
	)
	completion.Turns = []*llmports.ChatCompletionResult{
		toolTurn("report_list_dashboards_view", map[string]any{}),
		assistanttest.TextTurn("The memory was saved. Your question is still waiting."),
	}
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:           conversations.Thread.ID,
		FollowUpProposalID: proposal.ID,
		TenantInfo:         actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	note := conversations.Appended[0]
	assert.Contains(t, note.Content, question)
	assert.Contains(t, note.Content, "their answer has not arrived")

	var refusal *conversation.Message
	for idx := range conversations.Appended {
		if conversations.Appended[idx].ToolName == "report_list_dashboards_view" {
			refusal = &conversations.Appended[idx]
		}
	}
	require.NotNil(t, refusal, "the call is answered, not dropped")
	assert.True(t, refusal.ToolFailed)
	assert.Contains(t, refusal.Content, "this turn only reports the decision")
}
