package conversationtranscript_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationtranscript"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTranscriptOf_NothingSaidIsNoTranscript(t *testing.T) {
	t.Parallel()

	assert.Nil(t, conversationtranscript.RunTranscriptOf(nil))
	assert.Nil(t, conversationtranscript.RunTranscriptOf([]conversation.Message{}))
}

func TestRunTranscriptOf_KeepsWhatAReaderNeedsAndNothingTheProviderOwns(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew("agdef_")
	transcript := conversationtranscript.RunTranscriptOf([]conversation.Message{
		{
			Role:    conversation.RoleAssistant,
			Kind:    conversation.MessageKindMessage,
			Content: "Looking it up.",
			Reasoning: &conversation.ReasoningTrace{
				Text:      "Find the shipment first.",
				Signature: "sig",
				Encrypted: "opaque",
			},
			ToolCalls: []conversation.ToolCallRecord{{
				ID:           "c1",
				Name:         "shipment_get",
				Arguments:    map[string]any{"id": "shp_1"},
				ProviderData: map[string]any{"thought": "secret"},
			}},
			CreatedAt: 5,
		},
		{
			Role:              conversation.RoleTool,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: delegate,
			DelegateCallID:    "d1",
			ToolCallID:        "c1",
			ToolName:          "shipment_get",
			ToolSummary:       "S-1001",
			ToolVerdict:       "ran",
			Content:           `{"id":"shp_1"}`,
			CreatedAt:         6,
		},
	})

	require.NotNil(t, transcript)
	assert.Zero(t, transcript.OmittedMessages)
	assert.Equal(t, []agent.TranscriptMessage{
		{
			Role:      "Assistant",
			Kind:      "Message",
			Content:   "Looking it up.",
			Reasoning: "Find the shipment first.",
			ToolCalls: []agent.TranscriptToolCall{{
				ID:        "c1",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			}},
			CreatedAt: 5,
		},
		{
			Role:              "Tool",
			Kind:              "Delegated",
			Content:           `{"id":"shp_1"}`,
			ToolCallID:        "c1",
			ToolName:          "shipment_get",
			ToolVerdict:       "ran",
			ToolSummary:       "S-1001",
			AgentDefinitionID: delegate,
			DelegateCallID:    "d1",
			CreatedAt:         6,
		},
	}, transcript.Messages)
}

func TestRunTranscriptOf_AMessageOverTheBoundKeepsOnlyItsShape(t *testing.T) {
	t.Parallel()

	transcript := conversationtranscript.RunTranscriptOf([]conversation.Message{{
		Role:    conversation.RoleAssistant,
		Content: strings.Repeat("a", agent.TranscriptMessageBytes),
		ToolCalls: []conversation.ToolCallRecord{
			{ID: "c1", Name: "report_run", Arguments: map[string]any{"q": "x"}},
		},
		CreatedAt: 9,
	}})

	require.NotNil(t, transcript)
	require.Len(t, transcript.Messages, 1)
	kept := transcript.Messages[0]
	assert.True(t, kept.Omitted)
	assert.Empty(t, kept.Content)
	assert.Equal(t, []agent.TranscriptToolCall{{ID: "c1", Name: "report_run"}}, kept.ToolCalls)
	assert.Equal(t, int64(9), kept.CreatedAt)
}

func TestRunTranscriptOf_LeavesOutTheMiddleOfALongRun(t *testing.T) {
	t.Parallel()

	messages := make([]conversation.Message, 0, 30)
	for idx := range 30 {
		messages = append(messages, conversation.Message{
			Role:      conversation.RoleTool,
			Content:   strings.Repeat("z", 30*1024),
			CreatedAt: int64(idx),
		})
	}

	transcript := conversationtranscript.RunTranscriptOf(messages)

	require.NotNil(t, transcript)
	kept := transcript.Messages
	assert.Len(t, kept, 8, "eight thirty-kilobyte messages fit in the bound")
	assert.Equal(t, 22, transcript.OmittedMessages)
	assert.Equal(t, 4, transcript.OmittedAt)
	assert.Equal(t, int64(3), kept[3].CreatedAt)
	assert.Equal(t, int64(26), kept[4].CreatedAt)
	assert.Equal(t, int64(29), kept[7].CreatedAt)
}

func TestMessageOfTranscript_ReadsBackWhatTheRunKept(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew("agdef_")
	original := []conversation.Message{
		{
			Role:      conversation.RoleAssistant,
			Kind:      conversation.MessageKindMessage,
			Content:   "Looking it up.",
			Reasoning: &conversation.ReasoningTrace{Text: "Find the shipment first."},
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "c1",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			}},
			CreatedAt: 10,
		},
		{
			Role:              conversation.RoleTool,
			Kind:              conversation.MessageKindDelegated,
			ToolCallID:        "c1",
			ToolName:          "shipment_get",
			ToolFailed:        true,
			ToolVerdict:       "denied",
			ToolSummary:       "Not permitted",
			AgentDefinitionID: delegate,
			DelegateCallID:    "d1",
			Content:           "refused",
			CreatedAt:         11,
		},
	}

	transcript := conversationtranscript.RunTranscriptOf(original)
	require.NotNil(t, transcript)
	require.Len(t, transcript.Messages, len(original))

	for idx := range original {
		assert.Equal(
			t,
			original[idx],
			conversationtranscript.MessageOfTranscript(&transcript.Messages[idx]),
		)
	}
}

func TestMessageOfTranscript_AnUnmarkedEntryIsAPlainMessage(t *testing.T) {
	t.Parallel()

	message := conversationtranscript.MessageOfTranscript(&agent.TranscriptMessage{Role: "User"})

	assert.Equal(t, conversation.MessageKindMessage, message.Kind)
	assert.Nil(t, message.Reasoning)
	assert.Nil(t, message.ToolCalls)
}
