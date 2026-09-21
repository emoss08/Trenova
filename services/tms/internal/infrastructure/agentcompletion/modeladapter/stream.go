package modeladapter

import (
	"bufio"
	"bytes"
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

// maxStreamLineBytes bounds one line of a streamed reply. A single tool-call
// argument payload can be large, but anything past this is not a line a model
// produced; it is a broken framing that would otherwise consume memory forever.
const maxStreamLineBytes = 4 << 20

// postStream sends body and hands back the open reply body. A non-2xx status
// is read to completion and reported the same way postJSON reports it, so a
// stream that fails before its first byte is indistinguishable from a failed
// blocking call to the retry logic.
//
// It takes the whole call rather than a client because a stream needs the one
// without a whole-request deadline, and because the returned body is wrapped in
// an idle guard built from the call's own timeout.
func postStream(
	ctx context.Context,
	call *Call,
	url string,
	headers map[string]string,
	body any,
) (io.ReadCloser, error) {
	encoded, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}

	// Cancelled by the idle guard when the provider stops sending, and by Close
	// when the caller is done, so neither leaves the connection open.
	streamCtx, cancel := context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(streamCtx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		cancel()

		return nil, fmt.Errorf("build provider request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/x-ndjson, application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := call.streamHTTPClient().Do(req)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("execute provider request: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()

		return nil, transportError(resp, payload)
	}

	return newIdleGuard(resp.Body, cancel, call.streamIdleTimeout()), nil
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

func newIdleGuard(body io.ReadCloser, cancel context.CancelFunc, idle time.Duration) io.ReadCloser {
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

// readSSE parses a text/event-stream body and hands each event to handle. It
// follows the specification's framing: an event ends at a blank line, several
// data lines join with newlines, and a line starting with a colon is a comment
// (providers send those as keep-alives). Returning an error from handle stops
// the read and surfaces that error.
func readSSE(r io.Reader, handle func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), maxStreamLineBytes)

	var (
		event string
		data  []string
	)

	flush := func() error {
		if len(data) == 0 {
			event = ""
			return nil
		}
		err := handle(event, strings.Join(data, "\n"))
		event = ""
		data = data[:0]

		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
			continue
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read provider stream: %w", err)
	}

	return flush()
}

// readNDJSON hands each non-empty line to handle. Ollama streams this way: one
// JSON object per line, no framing beyond the newline.
func readNDJSON(r io.Reader, handle func(line []byte) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), maxStreamLineBytes)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := handle(line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read provider stream: %w", err)
	}

	return nil
}

// streamError is a failure the provider reported inside an otherwise healthy
// stream. It is retryable when the provider says the fault is on its side.
func streamError(errType, message string) error {
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
func decodeArguments(raw string) (map[string]any, string) {
	args := map[string]any{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return args, ""
	}
	if err := sonic.Unmarshal([]byte(trimmed), &args); err != nil {
		return map[string]any{}, err.Error()
	}

	return args, ""
}

// textReasoning wraps readable thinking with nothing to replay. Nil for
// nothing, so a reply without thinking carries no trace at all.
func textReasoning(text string) *ReasoningTrace {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	return &ReasoningTrace{Text: text}
}
