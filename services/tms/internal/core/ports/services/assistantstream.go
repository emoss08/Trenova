package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// TurnStreamFrame is one event as it was published, with the cursor a reader
// uses to ask for what came after it.
type TurnStreamFrame struct {
	// ID orders the frame and is handed to the reader verbatim as the SSE
	// event id, so a reconnecting browser's Last-Event-ID is a cursor this
	// side already understands.
	ID    string `json:"id"`
	Event string `json:"event"`
	// Data is the event's payload, already encoded. It is carried as bytes
	// rather than decoded and re-encoded on the way through: the relay is a
	// pipe, and nothing between the worker and the reader needs to understand
	// what a delta says.
	Data []byte `json:"data"`
}

// Terminal reports the frame that ends a turn.
func (f TurnStreamFrame) Terminal() bool {
	return TerminalAssistantEvent(f.Event)
}

// TurnStreamRef names one turn's stream.
type TurnStreamRef struct {
	TenantInfo pagination.TenantInfo
	TurnID     pulid.ID
}

// TurnStreamPublisher writes a turn's events where a reader can find them.
//
// A turn that runs on a worker has nobody holding its HTTP response, so its
// events cannot be written to one. They go here instead, and the API relays
// them. Which also buys the thing the old design could not do at any price: a
// reader who closed the tab can come back and rejoin a reply in progress,
// because the events are somewhere other than in the connection that was lost.
type TurnStreamPublisher interface {
	Publish(ctx context.Context, ref TurnStreamRef, event StreamEvent) error
	// Close publishes the terminal frame and shortens what is left to the
	// window a reconnecting reader needs. A stream that ends without one is
	// how a relay learns its worker died.
	Close(ctx context.Context, ref TurnStreamRef, event StreamEvent) error
}

// TurnStreamReader replays and then follows one turn's events.
type TurnStreamReader interface {
	// Read yields every frame after Cursor and keeps yielding until the turn
	// ends, ctx is done, or a callback returns an error. An empty cursor
	// starts at the beginning, which is what a fresh reader attaching to a
	// turn already in flight wants.
	Read(ctx context.Context, req ReadTurnStreamRequest) error
	// Exists reports whether the stream is still there. A turn whose stream
	// has expired is not a turn that failed; the answer is in the
	// conversation, and the reader is told to go and read it.
	Exists(ctx context.Context, ref TurnStreamRef) (bool, error)
}

// TurnFrameFunc receives frames in order.
type TurnFrameFunc func(frame TurnStreamFrame) error

// ReadTurnStreamRequest is one reader's position in one turn.
type ReadTurnStreamRequest struct {
	Ref TurnStreamRef
	// Cursor is the id of the last frame the reader actually applied, not the
	// last it received. Those differ when a connection dies mid-frame, and
	// resuming from the received one loses an event.
	Cursor string
	// OnFrame receives each event in order.
	OnFrame TurnFrameFunc
	// OnIdle is called when a read window passed with nothing in it, and is
	// how a silent turn stays distinguishable from a dead one. The reader
	// cannot make that distinction itself — only the caller knows whether the
	// workflow behind the stream is still running — so it gets asked, and
	// stops reading if the answer is an error.
	//
	// Nil means silence is never interesting and the read simply blocks again.
	OnIdle func() error
}
