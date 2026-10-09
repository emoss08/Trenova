package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RegisterWaitRequest parks the work of a conversation, or of a background
// run, until something happens.
type RegisterWaitRequest struct {
	Actor           *RequestActor
	DefinitionID    pulid.ID
	ThreadID        pulid.ID
	RunID           pulid.ID
	Delegated       bool
	Kind            agentwait.Kind
	Condition       agentwait.Condition
	Description     string
	Then            string
	LifetimeSeconds int64
}

type CancelWaitRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	// ThreadID and UserID, when set, cancel the wait only as one of that
	// person's conversation's.
	ThreadID pulid.ID
	UserID   pulid.ID
	// DefinitionID, when set without a thread, cancels only a wait that
	// agent set: a background run never reaches another agent's waits.
	DefinitionID pulid.ID
	By           string
}

type AgentWaitService interface {
	Register(ctx context.Context, req *RegisterWaitRequest) (*agentwait.Wait, error)
	Cancel(ctx context.Context, req *CancelWaitRequest) (*agentwait.Wait, error)
}

// AgentWaitNotifier hands what happened in the world to the waits watching
// for it. It never fails the work that reported it.
type AgentWaitNotifier interface {
	NotifyEvent(ctx context.Context, event *AgentEvent)
	NotifyHOS(
		ctx context.Context,
		tenant pagination.TenantInfo,
		states []*telematics.WorkerHOSState,
	)
	// NotifyPositions checks the polled truck positions against the stops
	// that open waits on an arrival or a departure are watching.
	NotifyPositions(
		ctx context.Context,
		tenant pagination.TenantInfo,
		positions []*telematics.VehiclePosition,
	)
}
