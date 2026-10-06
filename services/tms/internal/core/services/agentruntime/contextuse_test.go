package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// numbered gives a history its sequence numbers from zero, as the thread
// stores them.
func numbered(messages []conversation.Message) []conversation.Message {
	for idx := range messages {
		messages[idx].Sequence = idx
		if messages[idx].Kind == "" {
			messages[idx].Kind = conversation.MessageKindMessage
		}
	}

	return messages
}

func compactionAfter(history []conversation.Message, through int, summary string) conversation.Message {
	return conversation.Message{
		Role:       conversation.RoleUser,
		Kind:       conversation.MessageKindCompaction,
		Sequence:   history[len(history)-1].Sequence + 1,
		Content:    summary,
		Compaction: &conversation.Compaction{Through: through, Summarized: through + 1},
	}
}

/*
A compacted conversation is replayed from its summary: the model reads the
summary first, as a message from the person's side, told what it is, then the
turns the compaction kept whole, then what followed. Nothing the summary
replaced reaches the model.
*/
func TestReplayHistory_StartsFromTheLatestSummary(t *testing.T) {
	t.Parallel()

	history := numbered(append(append(
		readTurn("c1", "list_shipments", "S1, S2, S3"),
		quietTurns(2)...),
		conversation.Message{Role: conversation.RoleUser, Content: "And the late ones?"},
		conversation.Message{Role: conversation.RoleAssistant, Content: "Two are late."},
	))
	// The first read and the first quiet turn are summarized; the two turns
	// after them are kept whole.
	summary := compactionAfter(history, 5, "- The person listed shipments S1 to S3.")
	history = append(history, summary)

	messages, _ := replayHistory(history, nil)

	require.Len(t, messages, 5)
	assert.Equal(t, serviceports.RoleUser, messages[0].Role)
	assert.True(t, strings.HasPrefix(messages[0].Content, compactionPreamble),
		"the model is told the message is a summary standing in for what was compacted")
	assert.True(t, strings.HasSuffix(messages[0].Content, summary.Content))
	assert.Equal(t, "Turn 1", messages[1].Content)
	assert.Equal(t, "Noted.", messages[2].Content)
	assert.Equal(t, "And the late ones?", messages[3].Content)
	assert.Equal(t, "Two are late.", messages[4].Content)
	for _, message := range messages {
		assert.NotEqual(t, serviceports.RoleTool, message.Role,
			"the compacted tool result is not replayed")
	}
}

func TestSplitForCompaction_KeepsTheLatestTurnsWhole(t *testing.T) {
	t.Parallel()

	history := numbered(append(readTurn("c1", "list_shipments", "S1"), quietTurns(3)...))

	older, through := SplitForCompaction(history)

	// Four turns: the read and the first quiet one are summarized.
	require.Len(t, older, 6)
	assert.Equal(t, 5, through)
	assert.Equal(t, "Turn 1", history[through+1].Content, "the stretch ends where the kept turns begin")
}

func TestSplitForCompaction_HasNothingToDoWithOnlyTheKeptTurns(t *testing.T) {
	t.Parallel()

	older, _ := SplitForCompaction(numbered(quietTurns(CompactionKeepTurns)))
	assert.Empty(t, older)

	// Right after a compaction only the summary sits in front of the turns
	// it kept: there is nothing new to summarize.
	history := numbered(quietTurns(4))
	history = append(history, compactionAfter(history, 3, "- Earlier."))
	older, _ = SplitForCompaction(history)
	assert.Empty(t, older)
}

/*
A conversation compacted once is compacted again from its summary forward: the
summary goes into the new one with what followed it, and the stretch the new
one replaces ends past the summary's own.
*/
func TestSplitForCompaction_FoldsTheLastSummaryIn(t *testing.T) {
	t.Parallel()

	history := numbered(quietTurns(4))
	summary := compactionAfter(history, 3, "- Earlier.")
	history = append(history, summary)
	for _, message := range quietTurns(2) {
		message.Sequence = history[len(history)-1].Sequence + 1
		message.Kind = conversation.MessageKindMessage
		history = append(history, message)
	}

	older, through := SplitForCompaction(history)

	require.NotEmpty(t, older)
	assert.True(t, older[0].Compacted(), "the last summary is summarized again")
	assert.Greater(t, through, summary.Compaction.Through)
}

/*
The meter splits the context into the agent's instructions, what was said,
what the tools returned, and the text of the files the person attached, which
reaches the model through the document tools.
*/
func TestMeasureContext_CountsEachPartAgainstTheModelsWindow(t *testing.T) {
	t.Parallel()

	history := numbered(append(append(
		readTurn("c1", "list_shipments", strings.Repeat("x", 3000)),
		readTurn("c2", "get_document_summary", strings.Repeat("y", 900))...),
		quietTurns(2)...,
	))

	usage := MeasureContext(ContextRequest{
		System:  strings.Repeat("s", 4000),
		History: history,
		Model:   "claude-sonnet-4-5",
		Now:     42,
	})

	assert.Equal(t, 1000, usage.Instructions)
	assert.Equal(t, 300, usage.Files, "the attached file's text counts as files")
	assert.Positive(t, usage.ToolResults)
	assert.Positive(t, usage.Messages)
	assert.Equal(t, 200_000, usage.Window)
	assert.Equal(t, "claude-sonnet-4-5", usage.Model)
	assert.Equal(t, int64(42), usage.MeasuredAt)
	assert.Positive(t, usage.Compactable, "the two reads are behind the latest two turns")
	assert.Less(t, usage.Compactable, usage.Total())
}

func TestMeasureContext_KeepsTheLastInstructionsWithoutAPrompt(t *testing.T) {
	t.Parallel()

	usage := MeasureContext(ContextRequest{
		Instructions: 6200,
		History:      numbered(quietTurns(1)),
	})

	assert.Equal(t, 6200, usage.Instructions)
	assert.Zero(t, usage.Compactable, "one turn is all kept whole")
}

// A provider configured with its window is measured against that window, not
// the one its model id names: a Qwen served with a 128k context is not near
// full at 30k tokens.
func TestMeasureContext_AConfiguredWindowWinsOverTheModelID(t *testing.T) {
	t.Parallel()

	usage := MeasureContext(ContextRequest{
		History: numbered(quietTurns(1)),
		Model:   "qwen2.5-coder:32b",
		Window:  131_072,
	})
	assert.Equal(t, 131_072, usage.Window)

	inferred := MeasureContext(ContextRequest{History: numbered(quietTurns(1)), Model: "qwen2.5-coder:32b"})
	assert.Equal(t, 32_768, inferred.Window)
}
