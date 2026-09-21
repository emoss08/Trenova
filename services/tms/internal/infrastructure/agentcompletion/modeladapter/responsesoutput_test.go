package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A reasoning item shares the message item's content shape and always comes
// first in the output. Reading every item's content therefore made the chain
// of thought the reply — and because the first non-empty part wins, the real
// answer that followed it was dropped entirely.
func TestSplitResponsesOutput_DoesNotMistakeReasoningForTheAnswer(t *testing.T) {
	t.Parallel()

	envelope := &responsesEnvelope{Output: []responsesItem{
		{
			Type:    "reasoning",
			Summary: []responsesSummaryPart{{Type: "summary_text", Text: "weighing the statuses"}},
			Content: []responsesMessagePart{
				{Type: "reasoning_text", Text: "weighing the statuses"},
			},
		},
		{
			Type: "message",
			Role: "assistant",
			Content: []responsesMessagePart{
				{Type: "output_text", Text: "Six shipments are still in transit."},
			},
		},
	}}

	text, toolCalls, refused := splitResponsesOutput(envelope)

	assert.Equal(t, "Six shipments are still in transit.", text)
	assert.Nil(t, toolCalls)
	assert.False(t, refused)
}

// A part of any other type on a message item is not the answer either.
func TestSplitResponsesOutput_ReadsOnlyOutputTextParts(t *testing.T) {
	t.Parallel()

	envelope := &responsesEnvelope{Output: []responsesItem{{
		Type: "message",
		Role: "assistant",
		Content: []responsesMessagePart{
			{Type: "reasoning_text", Text: "thinking out loud"},
			{Type: "output_text", Text: "the answer"},
		},
	}}}

	text, _, _ := splitResponsesOutput(envelope)

	assert.Equal(t, "the answer", text)
}

// A refusal anywhere still stops the read, whatever item carried it.
func TestSplitResponsesOutput_StillHonoursARefusal(t *testing.T) {
	t.Parallel()

	envelope := &responsesEnvelope{Output: []responsesItem{{
		Type:    "message",
		Role:    "assistant",
		Content: []responsesMessagePart{{Type: "refusal", Refusal: "I cannot help with that."}},
	}}}

	text, toolCalls, refused := splitResponsesOutput(envelope)

	require.True(t, refused)
	assert.Empty(t, text)
	assert.Nil(t, toolCalls)
}

// An item with no type is still read, since a replayed or minimal envelope
// need not name it.
func TestSplitResponsesOutput_ReadsAnUntypedItem(t *testing.T) {
	t.Parallel()

	envelope := &responsesEnvelope{Output: []responsesItem{{
		Content: []responsesMessagePart{{Text: "the answer"}},
	}}}

	text, _, _ := splitResponsesOutput(envelope)

	assert.Equal(t, "the answer", text)
}
