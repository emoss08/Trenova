package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
type repeatGuard struct {
	failures map[string]string
}

func newRepeatGuard() *repeatGuard {
	return &repeatGuard{failures: make(map[string]string, 4)}
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
			"including anything you did manage to get.",
		name, previous,
	)
}

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
