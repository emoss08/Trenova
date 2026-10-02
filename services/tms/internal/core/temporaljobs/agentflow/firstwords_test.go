package agentflow

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

/*
The stream batches what a model call publishes on a ticker that starts with
the first publish, so the first words of every call in a turn's loop waited a
whole interval before they were even sent. The first piece of the reply —
text or thinking — goes at once; what follows is batched as before.
*/
func TestFirstWords_FlushesOnlyTheFirstPieceOfTheReply(t *testing.T) {
	t.Parallel()

	var first firstWords
	tool := serviceports.StreamEvent{Event: serviceports.AssistantEventToolStarted}
	thinking := serviceports.StreamEvent{Event: serviceports.AssistantEventReasoning}
	text := serviceports.StreamEvent{Event: serviceports.AssistantEventDelta}

	assert.False(t, first.flush(tool), "only the reply's own words are hurried")
	assert.True(t, first.flush(thinking))
	assert.False(t, first.flush(text))
	assert.False(t, first.flush(thinking))

	var answerFirst firstWords
	assert.True(t, answerFirst.flush(text))
	assert.False(t, answerFirst.flush(text))
}
