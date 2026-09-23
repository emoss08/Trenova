package turnstream

import (
	"testing"

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
