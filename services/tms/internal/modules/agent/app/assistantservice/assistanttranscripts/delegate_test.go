package assistanttranscripts

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

// The transcript shows another agent's steps nested under the call that
// handed it the task, headed by that agent's name, and the conversation's
// own steps as they always were.
func TestRenderTranscript_NestsADelegatesStepsUnderItsCall(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew("agdef_")
	messages := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Build a dashboard of on-time delivery.",
			CreatedAt: 1_790_000_000},
		{
			Role:      conversation.RoleAssistant,
			CreatedAt: 1_790_000_001,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_1",
				Name:      "delegate_task",
				Arguments: map[string]any{"agentId": delegate.String(), "task": "Make the report."},
			}},
		},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			Content: "Make the report.", CreatedAt: 1_790_000_002},
		{Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			ToolCalls: []conversation.ToolCallRecord{{ID: "call_d1", Name: "report_create"}},
			CreatedAt: 1_790_000_003},
		{Role: conversation.RoleTool, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			ToolCallID: "call_d1", ToolName: "report_create",
			Content: "Tool \"report_create\" ran successfully.", CreatedAt: 1_790_000_004},
		{Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			Content: "Saved the report rd_1.", CreatedAt: 1_790_000_005},
		{Role: conversation.RoleTool, ToolCallID: "call_1", ToolName: "delegate_task",
			Content: "the account", CreatedAt: 1_790_000_006},
		{Role: conversation.RoleAssistant, Content: "The tile is ready.", CreatedAt: 1_790_000_007},
	}

	body := renderTranscript(&transcriptInput{
		Thread:     transcriptThread(),
		AgentName:  "Homepage Widget Builder",
		Delegates:  map[pulid.ID]string{delegate: "Report Builder"},
		Messages:   messages,
		ExportedAt: 1_790_000_100,
	})

	assert.Equal(t, 1, strings.Count(body, "## Handed to Report Builder"),
		"the delegate's steps are one section")
	assert.Contains(t, body, "> ### Task · ")
	assert.Contains(t, body, "> Make the report.")
	assert.Contains(t, body, "> ## Report Builder · ")
	assert.Contains(t, body, "> **Called `report_create`**")
	assert.Contains(t, body, "> Saved the report rd_1.")
	assert.Equal(t, 1, strings.Count(body, "## You · "),
		"the task is not shown as something the person typed")
	assert.Less(t, strings.Index(body, "**Called `delegate_task`**"),
		strings.Index(body, "## Handed to Report Builder"))
	assert.Less(t, strings.Index(body, "## Handed to Report Builder"),
		strings.Index(body, "### Result from `delegate_task`"))
	assert.Contains(t, body, "## Homepage Widget Builder · ")
}

// A delegate deleted since is named plainly rather than left blank.
func TestRenderTranscript_NamesADeletedDelegatePlainly(t *testing.T) {
	t.Parallel()

	body := renderTranscript(&transcriptInput{
		Thread:    transcriptThread(),
		AgentName: "Homepage Widget Builder",
		Messages: []conversation.Message{{
			Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: pulid.MustNew("agdef_"), DelegateCallID: "call_1",
			Content: "Done.",
		}},
	})

	assert.Contains(t, body, "## Handed to Another agent")
}
