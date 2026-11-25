package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const batchQuestion = "Should I start planning drivers for the first batch of 50 Delayed shipments?"

func askedInTheSameTurn(proposalID pulid.ID) []conversation.Message {
	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "Plan drivers for the Delayed shipments"},
		{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_memory",
				Name:      "agent_create_memory",
				Arguments: map[string]any{"content": "Plan Delayed shipments in batches of 50"},
			}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "call_memory",
			ToolName:   "agent_create_memory",
			Content: "Recorded a proposal to run \"agent_create_memory\" (proposal " +
				proposalID.String() + "). It is awaiting a person's review at the Propose tier " +
				"and has not run.",
		},
		{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:   "call_ask",
				Name: "ask_user",
				Arguments: map[string]any{
					"question": batchQuestion,
					"options": []any{
						map[string]any{"label": "Yes, start with batch 1 of 50"},
						map[string]any{"label": "No, hold off"},
					},
				},
			}},
		},
		{Role: conversation.RoleTool, ToolCallID: "call_ask", ToolName: "ask_user"},
	}
}

/*
The agent filed agent_create_memory as a proposal and, in the same turn, asked
whether to start planning drivers for the first batch of 50. The person
approved the memory before answering. The follow-up wrote "You chose 'Yes,
start with batch 1 of 50'" and started planning. A follow-up that arrives under
an open question is now told the question is still waiting, that the
approval is not its answer, and to report only; and the turn is handed the
question so the runtime refuses its tool calls.
*/
func TestDecisionNote_HoldsAFollowUpUnderAnOpenQuestionToTheReport(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	thread := &conversation.Thread{ID: pulid.MustNew("ath_")}
	notes := &Notes{
		Proposals: &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{{
			ID:       proposalID,
			ToolName: "agent_create_memory",
			Status:   agent.ProposalStatusExecuted,
		}}},
	}
	actor := assistanttest.Actor()

	followed, err := notes.DecisionNote(t.Context(), &DecisionNoteParams{
		Thread:     thread,
		ProposalID: proposalID,
		History:    askedInTheSameTurn(proposalID),
		Tenant:     actor.TenantInfo(),
	})
	require.NoError(t, err)

	assert.Equal(t, batchQuestion, followed.OpenQuestion)
	assert.Contains(t, followed.Note, batchQuestion)
	assert.Contains(t, followed.Note, "still waiting for them and their answer has not arrived")
	assert.Contains(t, followed.Note, "do not say or assume which option they chose")
	assert.NotContains(t, followed.Note, continueTheWork,
		"a follow-up under an open question does not take the next step")
	assert.Equal(t, "Approved agent_create_memory, and it ran.",
		DecisionHeadline(followed.Note), "the person reads the decision, not the instruction")

	answered := append(askedInTheSameTurn(proposalID), conversation.Message{
		Role: conversation.RoleUser, Content: "Yes, start with batch 1 of 50",
	})
	followed, err = notes.DecisionNote(t.Context(), &DecisionNoteParams{
		Thread:     thread,
		ProposalID: proposalID,
		History:    answered,
		Tenant:     actor.TenantInfo(),
	})
	require.NoError(t, err)
	assert.Empty(t, followed.OpenQuestion, "a question the person answered is not open")
	assert.Contains(t, followed.Note, continueTheWork)
}
