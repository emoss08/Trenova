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
	// WorkflowID is the execution hosting the stream.
	WorkflowID string
}

// TurnStreamReader replays and then follows one turn's events.
//
// The events live in the turn's own workflow, which is running before the
// request that started it returns. So a reader can never attach ahead of the
// stream it wants, which is the race the stream exists to rule out.
type TurnStreamReader interface {
	// Read yields every frame after Cursor, in order, and keeps yielding
	// until the turn's last frame, until the turn's workflow has closed, or
	// until ctx is done or OnFrame returns an error. An empty cursor starts at
	// the beginning, which is what a fresh reader attaching to a turn already
	// in flight wants.
	//
	// It returns nil having yielded no terminal frame when the workflow closed
	// before the reader caught up. The caller then has to say how the turn
	// ended from the turn's record.
	Read(ctx context.Context, req ReadTurnStreamRequest) error
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
}
