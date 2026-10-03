// Package conversationschedulejobs runs the requests people scheduled in
// their conversations.
//
// Each schedule is a Temporal Schedule of its own, named
// conversation-schedule/{id}, which fires on the schedule's cron in its
// timezone, is paused while the schedule is, skips a slot that would overlap
// the one before it, and starts a short workflow that asks the service to
// start the turn. Temporal decides when; the service decides whether, as
// whom, and records what came of it.
package conversationschedulejobs

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// RunWorkflowName is what a conversation schedule starts on each slot.
	RunWorkflowName = "ConversationScheduleRunWorkflow"
	// ReconcileWorkflowName makes the Temporal schedules match the rows.
	ReconcileWorkflowName = "ReconcileConversationSchedulesWorkflow"

	// ReconcileScheduleID names the registry schedule that reconciles.
	ReconcileScheduleID = "reconcile-conversation-schedules"
)

// RunPayload names the schedule a slot belongs to.
type RunPayload struct {
	temporaltype.BasePayload

	ScheduleID pulid.ID `json:"scheduleId"`
}

// FireInput is one slot, for the activity that starts its turn.
type FireInput struct {
	Payload *RunPayload `json:"payload"`
	// FiredAt is when the workflow for the slot started, in Unix seconds.
	FiredAt int64 `json:"firedAt"`
}

// ReconcileResult is what one reconcile changed.
type ReconcileResult struct {
	Synced  int `json:"synced"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
}
