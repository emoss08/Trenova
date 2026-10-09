// Package agentwaitjobs holds a parked piece of agent work until what it
// waits for happens: a timer for a wait that comes due at a time, a signal
// for one an event ends, and the wait's own expiry for both. It costs nothing
// while it waits and survives every restart.
package agentwaitjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	WorkflowName = "AgentWaitWorkflow"
	// MetSignal tells a wait that what it waits for has happened.
	MetSignal = "agent-wait-met"
	// RescheduleSignal tells a wait that when it comes due moved, as a
	// driver's projected drive time does with each clock read.
	RescheduleSignal = "agent-wait-reschedule"
	// ReconcileWorkflowName closes the waits whose workflow vanished.
	ReconcileWorkflowName = "ReconcileAgentWaitsWorkflow"
	ReconcileScheduleID   = "agent-wait-reconcile"
	// errTypeBusy is a conversation still answering something else when its
	// wait ended; the pick-up is tried again until the conversation is free.
	errTypeBusy = "ConversationBusy"
)

// ErrConversationBusy is a wait that ended while its conversation was
// answering something else.
var ErrConversationBusy = errors.New("the conversation is answering something else")

func WorkflowIDFor(waitID pulid.ID) string { return "agent-wait:" + waitID.String() }

type Payload struct {
	temporaltype.BasePayload

	WaitID pulid.ID `json:"waitId"`
}

func (p *Payload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// Met is what ended a wait, as the event or poll that ended it said.
type Met struct {
	Detail string `json:"detail"`
	At     int64  `json:"at"`
}

// Schedule is when a wait next needs looking at: when it comes due, and when
// it runs out. Open is false for one already closed.
type Schedule struct {
	Open      bool   `json:"open"`
	DueAt     *int64 `json:"dueAt,omitempty"`
	ExpiresAt int64  `json:"expiresAt"`
}

// Check is a timed wait read again when it came due: met, due again later
// because the appointment or the free time moved, or closed meanwhile.
type Check struct {
	Met    bool   `json:"met"`
	Detail string `json:"detail,omitempty"`
	DueAt  *int64 `json:"dueAt,omitempty"`
	Closed bool   `json:"closed,omitempty"`
}

type CheckInput struct {
	Payload *Payload `json:"payload"`
	Now     int64    `json:"now"`
}

type FinishInput struct {
	Payload *Payload         `json:"payload"`
	Status  agentwait.Status `json:"status"`
	Outcome string           `json:"outcome"`
}

// Worker is what the workflow asks of the waits.
type Worker interface {
	Schedule(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*Schedule, error)
	Check(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID, now int64) (*Check, error)
	Finish(ctx context.Context, in *FinishInput) error
	// ReconcileOverdue closes and picks up every open wait past its expiry
	// whose workflow did not, and says how many.
	ReconcileOverdue(ctx context.Context) (int, error)
}
