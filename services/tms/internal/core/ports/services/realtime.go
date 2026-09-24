package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RealtimeEventReady         = "ready"
	RealtimeEventReset         = "reset"
	RealtimeEventHeartbeat     = "heartbeat"
	RealtimeEventClose         = "close"
	RealtimeEventInvalidation  = "resource.invalidation"
	RealtimeEventPresence      = "presence"
	RealtimeEventPresenceState = "presence.snapshot"
	RealtimeEventTyping        = "typing"

	RealtimePresenceEnter = "enter"
	RealtimePresenceLeave = "leave"

	RealtimeScopeUsers = "users"

	RealtimeCloseRotate   = "rotate"
	RealtimeCloseShutdown = "shutdown"
	RealtimeCloseOverflow = "overflow"
)

type ResourceInvalidationEvent struct {
	EventID        string    `json:"eventId"`
	OrganizationID string    `json:"organizationId"`
	BusinessUnitID string    `json:"businessUnitId"`
	Type           string    `json:"type"`
	Resource       string    `json:"resource"`
	Action         string    `json:"action"`
	EntityID       string    `json:"entityId,omitempty"`
	Fields         []string  `json:"fields,omitempty"`
	EntityVersion  int64     `json:"entityVersion,omitempty"`
	Entity         any       `json:"entity,omitempty"`
	RecordID       string    `json:"recordId,omitempty"`
	ActorUserID    string    `json:"actorUserId,omitempty"`
	ActorType      string    `json:"actorType,omitempty"`
	ActorID        string    `json:"actorId,omitempty"`
	ActorAPIKeyID  string    `json:"actorApiKeyId,omitempty"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type PublishResourceInvalidationRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	// AudienceUserID limits delivery to one person's connections. Zero means
	// every connection in the tenant.
	AudienceUserID pulid.ID
	Resource       string
	Action         string
	EventType      string
	Fields         []string
	EntityVersion  int64
	Entity         any
	RecordID       pulid.ID
	ActorUserID    pulid.ID
	ActorType      PrincipalType
	ActorID        pulid.ID
	ActorAPIKeyID  pulid.ID
}

type RealtimeService interface {
	PublishResourceInvalidation(
		ctx context.Context,
		req *PublishResourceInvalidationRequest,
	) error
}

// RealtimeEnvelope is one event on its way to every connection that may see
// it. Payload is what an internal user receives; PortalPayload is what a
// portal user receives when the event is not addressed to them, and nil keeps
// the event from portal users altogether.
type RealtimeEnvelope struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	AudienceUserID pulid.ID
	Event          string
	Scope          string
	Payload        []byte
	PortalPayload  []byte
	// PresenceConnectionID and PresenceAction are set on a presence event, so
	// the instance holding that connection learns which scopes it has joined
	// without a second round trip.
	PresenceConnectionID string
	PresenceAction       string
	// Ephemeral events describe the moment they are sent and are never replayed
	// to a reader resuming from an earlier position.
	Ephemeral bool
}

type RealtimeFrame struct {
	ID    string
	Event string
	Data  []byte
}

type RealtimePresenceMember struct {
	UserID       string `json:"userId"`
	ConnectionID string `json:"connectionId"`
	Name         string `json:"name"`
}

type RealtimePresenceEvent struct {
	Scope        string `json:"scope"`
	Action       string `json:"action"`
	UserID       string `json:"userId"`
	ConnectionID string `json:"connectionId"`
	Name         string `json:"name,omitempty"`
}

type RealtimePresenceSnapshot struct {
	Scope   string                   `json:"scope"`
	Members []RealtimePresenceMember `json:"members"`
}

type RealtimeTypingEvent struct {
	Scope  string `json:"scope"`
	UserID string `json:"userId"`
	Name   string `json:"name,omitempty"`
	Stop   bool   `json:"stop,omitempty"`
}

type RealtimeReadyEvent struct {
	ConnectionID        string `json:"connectionId"`
	HeartbeatIntervalMs int64  `json:"heartbeatIntervalMs"`
	Resumed             bool   `json:"resumed"`
	// Cursor is where a stream that did not resume starts, so a reader that
	// sees no event before its next reconnect still has a position to resume
	// from rather than refetching everything.
	Cursor string `json:"cursor,omitempty"`
}

type RealtimeCloseEvent struct {
	Reason string `json:"reason"`
}

type RealtimeConnection struct {
	ConnectionID   string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
	Name           string
	Portal         bool
}

type OpenRealtimeStreamRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
	Portal         bool
	JoinUsers      bool
	LastEventID    string
}

// RealtimeStream is one reader's live connection. Prelude is written before
// anything from Frames. Frames closes when the stream ends for any reason;
// Reason then says why.
type RealtimeStream interface {
	ConnectionID() string
	Prelude() []RealtimeFrame
	Frames() <-chan RealtimeFrame
	Reason() string
	Close()
}

type RealtimeScopeRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
	ConnectionID   string
	Scope          string
}

type RealtimeTypingRequest struct {
	RealtimeScopeRequest
	Stop bool
}

type RealtimeGateway interface {
	Open(ctx context.Context, req *OpenRealtimeStreamRequest) (RealtimeStream, error)
	JoinPresence(ctx context.Context, req *RealtimeScopeRequest) (*RealtimePresenceSnapshot, error)
	LeavePresence(ctx context.Context, req *RealtimeScopeRequest) error
	SignalTyping(ctx context.Context, req *RealtimeTypingRequest) error
	// Drain ends every open stream with a shutdown notice, so readers reconnect
	// to another instance instead of holding this one's shutdown open.
	Drain()
}

// RealtimePublisher writes events to the bus every instance reads from.
//
// Enqueue never blocks the caller: it hands the event to a background writer
// and reports whether there was room. Publish writes before it returns, for the
// few events whose order against a read matters, such as a presence change
// followed by a snapshot.
type RealtimePublisher interface {
	Enqueue(env *RealtimeEnvelope) bool
	Publish(ctx context.Context, env *RealtimeEnvelope) error
}

type RealtimeSubscribeRequest struct {
	Connection  *RealtimeConnection
	JoinUsers   bool
	LastEventID string
}

// RealtimeBroker is the transport behind the gateway: the bus, the local
// fan-out to open streams, and the presence register.
type RealtimeBroker interface {
	RealtimePublisher
	Subscribe(ctx context.Context, req *RealtimeSubscribeRequest) (RealtimeStream, error)
	Connection(ctx context.Context, connectionID string) (*RealtimeConnection, error)
	JoinPresence(
		ctx context.Context,
		conn *RealtimeConnection,
		scope string,
	) (*RealtimePresenceSnapshot, error)
	LeavePresence(ctx context.Context, conn *RealtimeConnection, scope string) error
	// Throttle reports whether key may act now, and if so holds it for window.
	Throttle(ctx context.Context, key string, window time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
	Drain()
}
