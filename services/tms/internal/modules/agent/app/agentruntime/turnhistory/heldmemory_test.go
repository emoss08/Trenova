package turnhistory

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A memory held for review because the turn had read outside text was approved
and ran, and the decision follow-up said it still needed an administrator's
approval. In the replay the call that filed it reads as what it came to, with
what the memory is now, and nothing of the "awaiting review" it said then.
*/
func TestReplay_AnApprovedHeldMemoryReadsAsKeptNotAwaitingReview(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Remember to plan Delayed shipments in 50s"},
		{
			ID:   sourceID,
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_memory",
				Name:      "agent_create_memory",
				Arguments: map[string]any{"content": "Plan Delayed shipments in batches of 50."},
			}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "call_memory",
			ToolName:   "agent_create_memory",
			Content: "Recorded a proposal to run \"agent_create_memory\". It is awaiting a " +
				"person's review at the Propose tier because this turn read text written " +
				"outside the organization, and has not run.",
		},
		{Role: conversation.RoleAssistant, Content: "I filed the memory for your review."},
		{
			Role:    conversation.RoleUser,
			Kind:    conversation.MessageKindDecisionNote,
			Content: "Approved agent_create_memory, and it ran.",
		},
	}
	outcomes := []agentruntimeports.ProposalOutcome{{
		SourceMessageID: sourceID,
		ToolName:        "agent_create_memory",
		Status:          agent.ProposalStatusExecuted,
		AutonomyTier:    agent.TierPropose,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "saved",
			Kind:   "memory",
			Name:   "Plan Delayed shipments in batches of 50.",
			State:  "kept and in effect, read in this person's later conversations",
		},
	}}

	messages := toAdapterMessages(history, outcomes)
	require.Len(t, messages, 5)

	replayed := messages[2].Content
	assert.NotContains(t, replayed, "awaiting")
	assert.Contains(t, replayed, `approved the proposal to run "agent_create_memory"`)
	assert.Contains(t, replayed, "Now: kept and in effect")
}
