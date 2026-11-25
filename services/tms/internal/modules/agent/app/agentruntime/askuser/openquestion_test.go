package askuser

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
)

func asked(id, question string) conversation.Message {
	return conversation.Message{
		Role: conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{
			ID:        id,
			Name:      "ask_user",
			Arguments: map[string]any{"question": question},
		}},
	}
}

// Only a message the person typed answers a question, as the thread shows
// it: a decision note or a wait picked up leaves it open.
func TestOpenQuestion_IsTheNewestQuestionNothingThePersonSaidHasAnswered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		history []conversation.Message
		want    string
	}{
		{
			name: "asked and not answered",
			history: []conversation.Message{
				{Role: conversation.RoleUser, Content: "Plan the delayed loads"},
				asked("call_1", "Start with batch 1?"),
				{Role: conversation.RoleTool, ToolCallID: "call_1", ToolName: "ask_user"},
			},
			want: "Start with batch 1?",
		},
		{
			name: "a decision note and its report leave it open",
			history: []conversation.Message{
				asked("call_1", "Start with batch 1?"),
				{
					Role:    conversation.RoleUser,
					Kind:    conversation.MessageKindDecisionNote,
					Content: "Approved agent_create_memory, and it ran.",
				},
				{Role: conversation.RoleAssistant, Content: "The memory was saved."},
			},
			want: "Start with batch 1?",
		},
		{
			name: "the person answered",
			history: []conversation.Message{
				asked("call_1", "Start with batch 1?"),
				{
					Role:    conversation.RoleUser,
					Kind:    conversation.MessageKindMessage,
					Content: "Yes",
				},
			},
		},
		{
			name: "a question the runtime refused was never shown",
			history: []conversation.Message{
				{Role: conversation.RoleUser, Content: "Plan the delayed loads"},
				asked("call_1", "Which shipments, pickups and drivers?"),
				{
					Role:       conversation.RoleTool,
					ToolCallID: "call_1",
					ToolName:   "ask_user",
					ToolFailed: true,
				},
			},
		},
		{
			name:    "nothing asked",
			history: []conversation.Message{{Role: conversation.RoleUser, Content: "Hi"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, OpenQuestion(tt.history))
		})
	}
}
