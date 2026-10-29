package loopguard

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

	// MaxReplyRetries bounds how often one turn asks again for a reply that
	// looped or said nothing. A model that does it twice running will do it a
	// third time, and the person is better served by a plain line.
	MaxReplyRetries = 1
)

// LoopedReply ends a turn whose reply looped twice. The loop itself is never
// kept: a person scrolling past three hundred copies of one word learns nothing
// from it but that the assistant is broken.
const LoopedReply = "I lost the thread while writing that answer. Ask again and I'll start fresh."

// EmptyReply ends a turn whose model twice produced nothing at all. Silence
// reads as a hung screen; this reads as an answer somebody can act on.
const EmptyReply = "I couldn't put an answer together that time. Ask again, or tell me a " +
	"little more about what you need."

// LoopRestartReason is what the reader is told while a looping reply is
// discarded and asked for again.
const LoopRestartReason = "The reply began repeating itself, so it is being written again."

// ReplyGuard watches a reply as it streams and stops it the moment it falls
// into a loop. Small models that start repeating a fragment do not stop until
// they run out of tokens; the Homepage Widget Builder's reply ended in
// hundreds of "time-" before the provider cut it off.
type ReplyGuard struct {
	text    strings.Builder
	checked int
	Tripped bool
	cancel  context.CancelFunc
}

func NewReplyGuard(cancel context.CancelFunc) *ReplyGuard {
	return &ReplyGuard{cancel: cancel}
}

// Feed takes one delta and reports whether it may be shown. Once the reply has
// looped nothing more is shown, and the stream is cancelled so the provider
// stops writing.
func (g *ReplyGuard) Feed(delta string) bool {
	if g.Tripped {
		return false
	}

	g.text.WriteString(delta)
	if g.text.Len()-g.checked < replyCheckEvery {
		return true
	}
	g.checked = g.text.Len()

	if _, looped := stringutils.DegenerateTail(g.text.String()); looped {
		g.Tripped = true
		g.cancel()
		return false
	}

	return true
}

// Looped reports whether the reply looped, checking the whole of it once more
// for a loop that arrived after the last check or without streaming at all.
func (g *ReplyGuard) Looped(final string) bool {
	if g.Tripped {
		return true
	}
	_, looped := stringutils.DegenerateTail(final)

	return looped
}
