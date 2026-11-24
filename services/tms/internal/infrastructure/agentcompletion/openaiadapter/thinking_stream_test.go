package openaiadapter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func streamInline(deltas ...string) (reply, thinking string) {
	var replyText, thinkText strings.Builder
	stream := newInlineThinkingStream(
		func(delta string) { replyText.WriteString(delta) },
		func(delta string) { thinkText.WriteString(delta) },
	)
	for _, delta := range deltas {
		stream.Write(delta)
	}
	stream.Flush()
	return replyText.String(), thinkText.String()
}

func TestInlineThinkingStream_NeverShowsTheThoughtAsTheReply(t *testing.T) {
	t.Parallel()

	reply, thinking := streamInline(
		"<thi", "nk>The user wants live ", "shipments.</th", "ink>Here are ", "the shipments.",
	)

	assert.Equal(t, "Here are the shipments.", reply)
	assert.Equal(t, "The user wants live shipments.", thinking)
}

func TestInlineThinkingStream_ReadsEveryTagInAnyCasing(t *testing.T) {
	t.Parallel()

	for _, deltas := range [][]string{
		{"<Thinking>", "weighing it", "</THINKING>", "the answer"},
		{"<reasoning>weighing it</reasoning>the answer"},
	} {
		reply, thinking := streamInline(deltas...)
		assert.Equal(t, "the answer", reply, deltas)
		assert.Equal(t, "weighing it", thinking, deltas)
	}
}

func TestInlineThinkingStream_AnUnterminatedBlockStaysThinking(t *testing.T) {
	t.Parallel()

	reply, thinking := streamInline("<think>still going and cut off </thi")

	assert.Empty(t, reply)
	assert.Equal(t, "still going and cut off </thi", thinking)
}

func TestInlineThinkingStream_LeavesUnmarkedTextAlone(t *testing.T) {
	t.Parallel()

	reply, thinking := streamInline("Rate is < 2.50 per mile", " and <b>firm</b> <t", "")

	assert.Equal(t, "Rate is < 2.50 per mile and <b>firm</b> <t", reply)
	assert.Empty(t, thinking)
}
