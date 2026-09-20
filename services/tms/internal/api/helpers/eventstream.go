package helpers

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

// ErrStreamingUnsupported reports a response writer that cannot flush, which
// is the one thing an event stream cannot do without.
var ErrStreamingUnsupported = errors.New("streaming is not supported by this connection")

// defaultHeartbeat is how often a silent stream says it is alive. Most proxies
// drop an idle upstream at 60 seconds; a reasoning model routinely thinks for
// longer than that before its first token.
const defaultHeartbeat = 15 * time.Second

// EventStreamOptions configures one stream.
type EventStreamOptions struct {
	// Heartbeat is the comment-line interval. Zero means the default; a
	// negative value disables it.
	Heartbeat time.Duration
}

// EventStream is one text/event-stream response.
//
// It exists because two things about a long-lived response are wrong by
// default and were wrong here.
//
// The server's WriteTimeout is a deadline for the whole response. On an
// ordinary request that is the right shape; on a stream that is a wall-clock
// budget for the entire answer, and at 75 seconds it severed every assistant
// turn that ran longer than that — a reasoning model plus two tool calls —
// mid-token, with the request context cancelled behind it. The stream lifts
// the deadline for itself, since it is the one response that knows it will
// be open a while.
//
// And a stream that is quiet is indistinguishable from one that is dead to
// anything between the server and the reader. A comment line every few seconds
// keeps the connection recognisably alive through the silence before a model's
// first token, and a comment is invisible to a conforming parser.
//
// Writes are serialised. The heartbeat runs on its own goroutine and the
// handler emits from the request's, and two writers interleaving on one
// connection would put a comment in the middle of an event.
type EventStream struct {
	writer  gin.ResponseWriter
	flusher http.Flusher

	mu     sync.Mutex
	closed bool

	stop   context.CancelFunc
	ticker *time.Ticker
	done   chan struct{}
}

// OpenEventStream sends the response headers and returns the stream. It fails
// only when the connection cannot stream at all, before anything is written,
// so the caller can still answer with an ordinary error response.
func OpenEventStream(c *gin.Context, opts EventStreamOptions) (*EventStream, error) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, ErrStreamingUnsupported
	}

	// Best effort: a writer that cannot take a deadline (a test recorder, an
	// unusual middleware wrapper) is left as it is rather than refused, since
	// the stream still works on a server with no write timeout.
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher.Flush()

	stream := &EventStream{writer: c.Writer, flusher: flusher, done: make(chan struct{})}

	heartbeat := opts.Heartbeat
	if heartbeat == 0 {
		heartbeat = defaultHeartbeat
	}
	if heartbeat > 0 {
		ctx, cancel := context.WithCancel(c.Request.Context())
		stream.stop = cancel
		stream.ticker = time.NewTicker(heartbeat)
		go stream.beat(ctx)
	} else {
		close(stream.done)
	}

	return stream, nil
}

// Emit writes one event. A value that cannot be encoded is dropped rather than
// written half-formed: a broken frame would take every later event with it.
func (s *EventStream) Emit(event string, data any) {
	encoded, err := sonic.Marshal(data)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	_, _ = s.writer.WriteString("event: " + event + "\n")
	_, _ = s.writer.WriteString("data: " + string(encoded) + "\n\n")
	s.flusher.Flush()
}

// Close ends the heartbeat. Nothing is written after it returns.
func (s *EventStream) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return
	}
	s.closed = true
	s.mu.Unlock()

	if s.stop != nil {
		s.stop()
		<-s.done
	}
}

func (s *EventStream) beat(ctx context.Context) {
	defer close(s.done)
	defer s.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()

				return
			}
			_, _ = s.writer.WriteString(": keepalive\n\n")
			s.flusher.Flush()
			s.mu.Unlock()
		}
	}
}
