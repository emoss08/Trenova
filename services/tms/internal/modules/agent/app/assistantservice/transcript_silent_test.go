package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
)

/*
A turn that ends on a question to the person saves an empty reply: ask_user
tells the model to write nothing more. The export drew it as a header with
nothing under it and counted it as a message.
*/
func TestRenderTranscript_LeavesOutTheSilentReplyAfterAQuestion(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	body := renderTranscript(&transcriptInput{
		Thread:    thread,
		AgentName: "Shipment Assistant",
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "Create a shipment from this", CreatedAt: 1},
			{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{
				ID: "call_ask", Name: "ask_user",
			}}, CreatedAt: 2},
			{
				Role:       conversation.RoleTool,
				ToolCallID: "call_ask",
				ToolName:   "ask_user",
				Content:    "{}",
				CreatedAt:  3,
			},
			{Role: conversation.RoleAssistant, Content: "  ", CreatedAt: 4},
			{Role: conversation.RoleUser, Content: "42000", CreatedAt: 5},
		},
	})

	assert.Equal(t, 1, strings.Count(body, "## Shipment Assistant"),
		"the call to ask_user is drawn and the empty reply is not")
	assert.Contains(t, body, "- **Messages:** 4\n")
}
