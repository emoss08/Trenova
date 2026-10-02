package turnstream

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
)

// A reader resumes after the last event it applied. A cursor that is not an
// offset, such as one a browser kept from the stream this replaced, starts
// over, which the client handles by refetching the conversation.
func TestAfter(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(0), after(""), "no cursor reads from the start")
	assert.Equal(t, int64(8), after("7"))
	assert.Equal(t, int64(0), after("1758000000000-0"), "a redis entry id starts over")
	assert.Equal(t, int64(0), after("-3"))
}

/*
After every batch it delivers, the subscription sleeps for its poll cooldown
before asking for the next, and the library's default is 100ms — on top of the
publisher's own 100ms batching, so a reply's words arrived in clumps. The poll
waits on the workflow until there is something to return, so a short cooldown
costs no extra polls; it only stops the reader idling after each delivery.
*/
func TestSubscription_DoesNotIdleBetweenBatches(t *testing.T) {
	t.Parallel()

	opts := subscription("41")

	assert.Equal(t, int64(42), opts.FromOffset)
	assert.Equal(t, []string{temporaltype.StreamEventsTopic}, opts.Topics)
	assert.Positive(t, opts.PollCooldown, "zero falls back to the library's 100ms")
	assert.LessOrEqual(t, opts.PollCooldown, 10*time.Millisecond)
}
