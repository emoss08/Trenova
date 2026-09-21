package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Chat Completions has one content field, so a model trained to think
// first marks its thinking inline. Left there it becomes the answer: shown
// to the person as the reply, stored as the reply, and replayed next turn
// as something the assistant said.
func TestSplitInlineThinking_LiftsAMarkedBlockOutOfTheReply(t *testing.T) {
	t.Parallel()

	reply, thinking := SplitInlineThinking(
		"<think>The user wants in-progress shipments. InTransit and Delayed " +
			"are the live ones.</think>Here are the shipments still moving.",
	)

	assert.Equal(t, "Here are the shipments still moving.", reply)
	assert.Equal(t,
		"The user wants in-progress shipments. InTransit and Delayed are the live ones.",
		thinking,
	)
}

func TestSplitInlineThinking_ReadsTheOtherTagsAndAnyCasing(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"<thinking>weighing it up</thinking>the answer",
		"<Think>weighing it up</Think>the answer",
		"<reasoning>weighing it up</reasoning>the answer",
	} {
		reply, thinking := SplitInlineThinking(text)
		assert.Equal(t, "the answer", reply, text)
		assert.Equal(t, "weighing it up", thinking, text)
	}
}

// A model cut off mid-thought never reached an answer. Everything after the
// tag is thinking, and the reply is empty — which the caller already treats
// as a failure worth reporting rather than as a reply worth showing.
func TestSplitInlineThinking_TreatsAnUnclosedBlockAsAllThinking(t *testing.T) {
	t.Parallel()

	reply, thinking := SplitInlineThinking("<think>still going and going")

	assert.Empty(t, reply)
	assert.Equal(t, "still going and going", thinking)
}

func TestSplitInlineThinking_LeavesUnmarkedTextAlone(t *testing.T) {
	t.Parallel()

	// Guessing where unmarked thinking ends would throw away answers, so
	// nothing without an opening tag is touched.
	reply, thinking := SplitInlineThinking("Here are the shipments still moving.")

	assert.Equal(t, "Here are the shipments still moving.", reply)
	assert.Empty(t, thinking)
}

// The case from the field: a server that both marks the thinking inline and
// repeats it in reasoning_content. Believing the content showed a person the
// model's deliberation where the answer should have been.
func TestMergeInlineThinking_DoesNotKeepTheTraceTwice(t *testing.T) {
	t.Parallel()

	thought := "Let me list the reports first."
	reply, trace := mergeInlineThinking("<think>"+thought+"</think>", textReasoning(thought))

	assert.Empty(t, reply, "thinking alone is not an answer")
	require.NotNil(t, trace)
	assert.Equal(t, thought, trace.Text)
}

func TestMergeInlineThinking_RefusesAReplyThatIsOnlyTheTraceRepeated(t *testing.T) {
	t.Parallel()

	thought := "I should look at the shipment dataset."
	reply, trace := mergeInlineThinking(thought, textReasoning(thought))

	assert.Empty(t, reply)
	require.NotNil(t, trace)
	assert.Equal(t, thought, trace.Text)
}

func TestMergeInlineThinking_KeepsBothWhenTheyDiffer(t *testing.T) {
	t.Parallel()

	reply, trace := mergeInlineThinking(
		"<think>inline thought</think>the answer",
		textReasoning("protocol thought"),
	)

	assert.Equal(t, "the answer", reply)
	require.NotNil(t, trace)
	assert.Contains(t, trace.Text, "protocol thought")
	assert.Contains(t, trace.Text, "inline thought")
}

func TestMergeInlineThinking_LeavesAnOrdinaryReplyUntouched(t *testing.T) {
	t.Parallel()

	reply, trace := mergeInlineThinking("the answer", nil)

	assert.Equal(t, "the answer", reply)
	assert.Nil(t, trace)
}
