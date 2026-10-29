package modeladapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/stringutils"
)

// StreamSink receives model text as it is produced, in order, possibly one
// token at a time. It is never called after Stream returns.
type StreamSink func(delta string)

// Streamer is an adapter whose protocol can deliver text incrementally. The
// final Response is the same one Complete would have returned; the sink is a
// preview of its Text, not a substitute for it.
type Streamer interface {
	Stream(ctx context.Context, call *Call, sink StreamSink) (*Response, error)
}

// ErrStreamStalled marks a stream the provider stopped feeding.
//
// It is distinct from a cancelled turn and from a transport error: the
// connection was open and nothing was wrong with it, the model simply stopped
// producing. Naming it separately is what lets the reply say the provider went
// quiet instead of blaming the reader's own request.
var ErrStreamStalled = errors.New("provider stopped sending")

// idleGuard aborts a stream that goes quiet for too long.
//
// A streaming client carries no whole-request deadline, because one would cut
// off a long answer that is arriving perfectly well. Liveness is measured
// between bytes instead: the clock restarts on every read that returns data, so
// a model may take as long as it likes to produce an answer and only silence
// ends the stream.
type idleGuard struct {
	body   io.ReadCloser
	cancel context.CancelFunc
	timer  *time.Timer
	idle   time.Duration

	mu      sync.Mutex
	stalled bool
	closed  bool
}

func NewIdleGuard(body io.ReadCloser, cancel context.CancelFunc, idle time.Duration) io.ReadCloser {
	guard := &idleGuard{body: body, cancel: cancel}
	guard.timer = time.AfterFunc(idle, guard.trip)
	guard.idle = idle

	return guard
}

// trip cancels the request, which unblocks whatever read is waiting on it.
func (g *idleGuard) trip() {
	g.mu.Lock()
	g.stalled = true
	g.mu.Unlock()
	g.cancel()
}

func (g *idleGuard) Read(p []byte) (int, error) {
	n, err := g.body.Read(p)
	if n > 0 {
		// Reset only on data. A read that returns nothing has not proved the
		// provider is alive, so it must not buy another full window.
		g.timer.Reset(g.idle)
	}
	if err != nil {
		g.mu.Lock()
		stalled := g.stalled
		g.mu.Unlock()
		if stalled && !errors.Is(err, io.EOF) {
			return n, fmt.Errorf("%w after %s of silence", ErrStreamStalled, g.idle)
		}
	}

	return n, err
}

func (g *idleGuard) Close() error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()

		return nil
	}
	g.closed = true
	g.mu.Unlock()

	g.timer.Stop()
	err := g.body.Close()
	g.cancel()

	return err
}

// streamError is a failure the provider reported inside an otherwise healthy
// stream. It is retryable when the provider says the fault is on its side.
func StreamError(errType, message string) error {
	retryable := strings.Contains(errType, "overloaded") ||
		strings.Contains(errType, "rate_limit") ||
		strings.Contains(errType, "server_error") ||
		strings.Contains(errType, "api_error")

	return &TransportError{
		StatusCode: http.StatusBadGateway,
		Retryable:  retryable,
		Message:    stringutils.FirstNonEmpty(message, "provider stream failed"),
	}
}

// decodeArguments turns a JSON-encoded argument string into a map. Unparseable
// arguments become an empty map for the same reason the blocking path tolerates
// them: the tool's own validation reports a clearer error than a failed turn.
// decodeArguments reads a tool call's argument text. The second value is the
// parse failure, in words, or empty.
//
// It used to swallow the failure and return an empty map, on the theory that
// the tool would then report a clear validation error. Some tools do. A list
// tool does not: given no filters it lists, and the model reports the first
// page as the answer to the question it actually asked. The failure is
// returned so the runtime can refuse the call and say why.
func DecodeArguments(raw string) (args map[string]any, failure string) {
	args = map[string]any{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return args, ""
	}
	if err := sonic.Unmarshal([]byte(trimmed), &args); err != nil {
		return map[string]any{}, err.Error()
	}

	return args, ""
}
