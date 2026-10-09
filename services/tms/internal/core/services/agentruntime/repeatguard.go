package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// repeatGuard stops a turn re-sending a call that has already failed unchanged.
//
// A tool that rejects its input rejects the same input the same way, but a model
// reading its own failure does not always conclude that. One sent a report
// parameter in a shape the compiler refuses five times in a row — narrating a
// different fix before each attempt and then sending the identical payload —
// which spent most of the turn's tool budget, filled the thread with the same
// red row five times, and left the person watching an assistant apparently
// trying things.
//
// The guard is on the arguments, not the tool: a second call with different
// arguments is the retry that was wanted, and only a byte-identical repeat is
// refused. What comes back names the original failure, because a model that
// repeats itself is one that did not absorb the error the first time.
//
// A read that succeeded is guarded the same way until something writes. One
// turn fetched the same billing queue item five times running, each time
// getting the record it already had; nothing in between could have changed
// it. The identical read is answered from the earlier one until a call that is
// not a read runs, after which the record may have moved and reading it again
// is the right thing to do.
type repeatGuard struct {
	failures map[string]string
	reads    map[string]bool
	// refusals counts the repeats refused this turn. A model that sends a
	// refused call a second time has stopped reading its results, and the
	// loop stops offering it tools (maxRepeatedFailures).
	refusals int
}

// maxRepeatedFailures is how many repeats of failed calls a turn refuses
// before it stops calling tools and answers. One billing turn sent the same
// refused transfer five more times after the first refusal, alternating
// between two variants, and spent its whole budget; the person waited for
// a minute to be told what the second refusal already showed.
const maxRepeatedFailures = 2

// refused counts one refused repeat and reports whether the turn has now
// repeated failed calls often enough to stop.
func (g *repeatGuard) refused() bool {
	g.refusals++

	return g.refusals >= maxRepeatedFailures
}

func newRepeatGuard() *repeatGuard {
	return &repeatGuard{failures: make(map[string]string, 4), reads: make(map[string]bool, 4)}
}

// isRead is a tool that only looks something up, by the names read tools take.
func isRead(name string) bool {
	for _, prefix := range []string{"get_", "list_", "search_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// readBefore reports whether this exact read already succeeded since the last
// call that could have changed anything.
func (g *repeatGuard) readBefore(call serviceports.ToolCall) bool {
	if !isRead(call.Name) {
		return false
	}
	key, ok := callKey(call)

	return ok && g.reads[key]
}

// ran notes a call that succeeded: a read is remembered, and anything else
// forgets every read, since it may have changed what they returned.
func (g *repeatGuard) ran(call serviceports.ToolCall) {
	if !isRead(call.Name) {
		clear(g.reads)
		return
	}
	if key, ok := callKey(call); ok {
		g.reads[key] = true
	}
}

// repeatedRead is what the model gets instead of the same record again.
func repeatedRead(name string) string {
	return fmt.Sprintf(
		"This exact call to %q already ran earlier in this turn, and nothing has changed "+
			"since, so it was not run again: its result is above. Work from that result. "+
			"Call it again only with different arguments.",
		name,
	)
}

// seen reports the earlier failure for this exact call, if there was one.
func (g *repeatGuard) seen(call serviceports.ToolCall) (string, bool) {
	key, ok := callKey(call)
	if !ok {
		return "", false
	}

	previous, found := g.failures[key]

	return previous, found
}

// record remembers a failure so the identical call is not run again.
func (g *repeatGuard) record(call serviceports.ToolCall, message string) {
	key, ok := callKey(call)
	if !ok {
		return
	}

	if _, exists := g.failures[key]; !exists {
		g.failures[key] = message
	}
}

// refusal is what the model gets instead of the same error a second time.
func repeatRefusal(name, previous string) string {
	return fmt.Sprintf(
		"This exact call to %q was already made in this turn and failed: %s\n\n"+
			"It was not run again, because the same arguments produce the same result. "+
			"Either change the arguments — read the message above for what the tool "+
			"actually accepts — or stop and tell the person plainly what is blocking, "+
			"including anything you did manage to get. Sending a failed call again "+
			"unchanged once more ends this turn's tool use.",
		name, previous,
	)
}

// changeStopOnRepeats is the loop ending its tool use once a turn has sent
// failed calls again unchanged maxRepeatedFailures times.
const changeStopOnRepeats = "agent-loop-stop-on-repeats"

// stoppedCallText answers a call in the same batch as the repeat that ended
// the turn's tool use: every call a model makes needs a result, and this one
// did not run.
func stoppedCallText(name string) string {
	return fmt.Sprintf(
		"Tool %q was not run: this turn stopped calling tools after a call that had "+
			"already failed was sent again unchanged.",
		name,
	)
}

// stuckNote is what the model is told when the loop stops for repeats.
const stuckNote = "You sent calls that had already failed, unchanged, more than once, so " +
	"this turn's tools are closed. Answer the person now from what the tools returned: " +
	"say what you were trying to do, what the tool refused, in plain words, and what " +
	"they can do instead, such as where in Trenova they can do it themselves. Do not " +
	"say the change was made, and do not ask for another tool."

// stuckReply ends a turn stopped for repeats when even the answer without
// tools could not be had.
const stuckReply = "I could not finish this: the tool kept refusing the request as I sent " +
	"it, so nothing was changed. Try asking again in different words, or make the " +
	"change in Trenova directly."

// callKey identifies a call by its name and its arguments.
//
// ConfigStd, not the default: sonic's fast path emits map keys in whatever order
// it finds them, so two calls differing only in how a provider happened to
// serialise the same arguments would hash differently and the guard would never
// fire. ConfigStd sorts them, which is the whole reason this identifies a call
// at all. A call whose arguments cannot be marshalled is never matched, which
// fails open — running a tool twice is a smaller fault than refusing a call
// that was never made.
func callKey(call serviceports.ToolCall) (string, bool) {
	encoded, err := sonic.ConfigStd.Marshal(call.Arguments)
	if err != nil {
		return "", false
	}

	sum := sha256.Sum256(append([]byte(call.Name+"\x00"), encoded...))

	return hex.EncodeToString(sum[:]), true
}

// fanOutAfter is how many single-record fetches of one kind a turn makes
// before it is reminded that a list usually already has what it is after.
const fanOutAfter = 3

// fanOutNote is added to a single-record fetch once a turn has made several
// of the same kind. A model that listed the billing queue and then fetched
// each of its nine items to read one field spent nine steps, and the person
// read nine citations, on what the list either had or could not show.
func fanOutNote(name string, count int) string {
	if count < fanOutAfter || !strings.HasPrefix(name, "get_") {
		return ""
	}

	return fmt.Sprintf("\n\n[This is the %d%s %s call in this turn. If a list or search "+
		"already returned these records, answer from it; fetch a record on its own only for "+
		"a field the list does not carry, and a field withheld by access stays withheld.]",
		count, ordinalSuffix(count), name)
}

func ordinalSuffix(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
