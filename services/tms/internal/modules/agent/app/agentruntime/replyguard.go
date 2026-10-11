package agentruntime

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	// replyCheckEvery is how many new bytes arrive between loop checks. The
	// check reads the reply's tail; running it on every token would cost more
	// than the loop it catches.
	replyCheckEvery = 256

	// maxReplyRetries bounds how often one turn asks again for a reply that
	// looped or said nothing. A model that does it twice running will do it a
	// third time, and the person is better served by a plain line.
	maxReplyRetries = 1
)

// loopedReply ends a turn whose reply looped twice. The loop itself is never
// kept: a person scrolling past three hundred copies of one word learns nothing
// from it but that the assistant is broken.
const loopedReply = "I lost the thread while writing that answer. Ask again and I'll start fresh."

// emptyReply ends a turn whose model twice produced nothing at all. Silence
// reads as a hung screen; this reads as an answer somebody can act on.
const emptyReply = "I couldn't put an answer together that time. Ask again, or tell me a " +
	"little more about what you need."

// loopRestartReason is what the reader is told while a looping reply is
// discarded and asked for again.
const loopRestartReason = "The reply began repeating itself, so it is being written again."

// replyGuard watches a reply as it streams and stops it the moment it falls
// into a loop. Small models that start repeating a fragment do not stop until
// they run out of tokens; the Homepage Widget Builder's reply ended in
// hundreds of "time-" before the provider cut it off.
type replyGuard struct {
	text    strings.Builder
	checked int
	tripped bool
	cancel  context.CancelFunc
}

func newReplyGuard(cancel context.CancelFunc) *replyGuard {
	return &replyGuard{cancel: cancel}
}

// feed takes one delta and reports whether it may be shown. Once the reply has
// looped nothing more is shown, and the stream is cancelled so the provider
// stops writing.
func (g *replyGuard) feed(delta string) bool {
	if g.tripped {
		return false
	}

	g.text.WriteString(delta)
	if g.text.Len()-g.checked < replyCheckEvery {
		return true
	}
	g.checked = g.text.Len()

	if _, looped := stringutils.DegenerateTail(g.text.String()); looped {
		g.tripped = true
		g.cancel()
		return false
	}

	return true
}

// looped reports whether the reply looped, checking the whole of it once more
// for a loop that arrived after the last check or without streaming at all.
func (g *replyGuard) looped(final string) bool {
	if g.tripped {
		return true
	}
	_, looped := stringutils.DegenerateTail(final)

	return looped
}
