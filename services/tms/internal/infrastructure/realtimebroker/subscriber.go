package realtimebroker

import (
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

// entry is one bus message as the fan-out read it, decoded once and shared by
// every local subscriber of its tenant.
type entry struct {
	id         streamID
	cursor     string
	tenant     string
	audience   string
	event      string
	scope      string
	payload    []byte
	portal     []byte
	ephemeral  bool
	connection string
	action     string
}

// subscriber is one open stream on this instance.
//
// While the stream is being set up (its snapshot read and its replay run) the
// fan-out already delivers to it, into a backlog. Setup then releases the
// backlog behind the prelude, and from there on anything at or below the
// replay floor is dropped, since the replay already carried it.
type subscriber struct {
	conn    services.RealtimeConnection
	userID  string
	tenant  string
	shard   int
	frames  chan services.RealtimeFrame
	onClose func(*subscriber)

	mu      sync.Mutex
	scopes  map[string]struct{}
	prelude []services.RealtimeFrame
	backlog []services.RealtimeFrame
	paused  bool
	floor   streamID
	closed  bool
	reason  string
}

func newSubscriber(
	conn *services.RealtimeConnection,
	tenant string,
	shard int,
	buffer int,
	onClose func(*subscriber),
) *subscriber {
	return &subscriber{
		conn:    *conn,
		userID:  conn.UserID.String(),
		tenant:  tenant,
		shard:   shard,
		frames:  make(chan services.RealtimeFrame, buffer),
		onClose: onClose,
		scopes:  make(map[string]struct{}, 2),
		paused:  true,
	}
}

func (s *subscriber) ConnectionID() string {
	return s.conn.ConnectionID
}

func (s *subscriber) Prelude() []services.RealtimeFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	prelude := s.prelude
	s.prelude = nil
	return prelude
}

func (s *subscriber) Frames() <-chan services.RealtimeFrame {
	return s.frames
}

func (s *subscriber) Reason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reason
}

func (s *subscriber) Close() {
	s.shutdown(closeDisconnect)
}

// closeDisconnect is the reason recorded when the reader's request ends. It
// is never sent, since there is nobody left to send it to.
const closeDisconnect = "disconnect"

// shutdown closes the stream once, with the first reason given. The
// broker's cleanup runs outside the lock, since it reaches back into the hub.
func (s *subscriber) shutdown(reason string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.reason = reason
	s.backlog = nil
	close(s.frames)
	s.mu.Unlock()

	if s.onClose != nil {
		s.onClose(s)
	}
}

func (s *subscriber) join(scope string) {
	s.mu.Lock()
	s.scopes[scope] = struct{}{}
	s.mu.Unlock()
}

func (s *subscriber) joinedScopes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	scopes := make([]string, 0, len(s.scopes))
	for scope := range s.scopes {
		scopes = append(scopes, scope)
	}
	return scopes
}

// accepts decides whether this reader may see e at all, and with which
// payload. It also moves the reader's own presence membership when e is
// about this connection, which is how an instance learns a join that another
// instance served.
func (s *subscriber) accepts(e *entry) ([]byte, bool) {
	if e.audience != "" && e.audience != s.userID {
		return nil, false
	}

	if e.scope != "" {
		own := e.connection == s.conn.ConnectionID
		s.mu.Lock()
		if own && e.action == services.RealtimePresenceEnter {
			s.scopes[e.scope] = struct{}{}
		}
		_, member := s.scopes[e.scope]
		if own && e.action == services.RealtimePresenceLeave {
			delete(s.scopes, e.scope)
			member = true
		}
		s.mu.Unlock()
		if !member {
			return nil, false
		}
	}

	if !s.conn.Portal || (e.audience != "" && e.audience == s.userID) {
		return e.payload, true
	}
	if e.portal == nil {
		return nil, false
	}
	return e.portal, true
}

// deliver hands a live entry to the reader. A reader that cannot keep up is
// closed rather than waited for: one slow browser must not hold up the fan-out
// every other reader on this instance depends on. It reconnects and resumes
// from its last cursor.
func (s *subscriber) deliver(e *entry) bool {
	payload, ok := s.accepts(e)
	if !ok {
		return false
	}

	frame := services.RealtimeFrame{ID: e.cursor, Event: e.event, Data: payload}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	if !s.floor.IsZero() && !s.floor.Less(e.id) {
		s.mu.Unlock()
		return false
	}
	if s.paused {
		if len(s.backlog) >= cap(s.frames) {
			s.mu.Unlock()
			s.shutdown(services.RealtimeCloseOverflow)
			return false
		}
		s.backlog = append(s.backlog, frame)
		s.mu.Unlock()
		return true
	}

	select {
	case s.frames <- frame:
		s.mu.Unlock()
		return true
	default:
		s.mu.Unlock()
		s.shutdown(services.RealtimeCloseOverflow)
		return false
	}
}

// send queues a frame that is not from the bus, such as a heartbeat. It is
// dropped rather than forced when the reader is behind: a reader with a full
// buffer is plainly still receiving.
func (s *subscriber) send(frame services.RealtimeFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.paused {
		return
	}
	select {
	case s.frames <- frame:
	default:
	}
}

// release ends setup: the prelude is fixed, the floor is set, and the backlog
// that arrived meanwhile follows the prelude minus anything the replay covered.
func (s *subscriber) release(prelude []services.RealtimeFrame, floor streamID) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}

	s.prelude = prelude
	s.floor = floor
	backlog := s.backlog
	s.backlog = nil
	s.paused = false

	for _, frame := range backlog {
		if !floor.IsZero() && frame.ID != "" {
			if c, ok := parseCursor(frame.ID); ok && !floor.Less(c.id) {
				continue
			}
		}
		select {
		case s.frames <- frame:
		default:
			s.mu.Unlock()
			s.shutdown(services.RealtimeCloseOverflow)
			return
		}
	}
	s.mu.Unlock()
}
