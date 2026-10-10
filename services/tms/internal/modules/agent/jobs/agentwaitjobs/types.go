// Package agentwaitjobs holds a parked piece of agent work until what it
// waits for happens: a timer for a wait that comes due at a time, a signal
// for one an event ends, and the wait's own expiry for both. It costs nothing
// while it waits and survives every restart.
package agentwaitjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
)

const (

	// ReconcileWorkflowName closes the waits whose workflow vanished.
	ReconcileWorkflowName = "ReconcileAgentWaitsWorkflow"
	ReconcileScheduleID   = "agent-wait-reconcile"
	// errTypeBusy is a conversation still answering something else when its
	// wait ended; the pick-up is tried again until the conversation is free.
	errTypeBusy = "ConversationBusy"
)

type CheckInput struct {
	Payload *agentwaitcontract.Payload `json:"payload"`
	Now     int64                      `json:"now"`
}
