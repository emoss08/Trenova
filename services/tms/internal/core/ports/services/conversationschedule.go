package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ConversationSchedulesResource is the realtime resource a person's schedules
// are announced under when one runs or changes, so a schedule card shows its
// last and next run without being polled.
const ConversationSchedulesResource = "conversation_schedules"

// ConversationScheduleSyncer keeps the Temporal schedule behind a
// conversation schedule in line with it. Neither call fails the request that
// made it: a schedule a failed call left behind is repaired by the next
// reconcile, and a run that finds its schedule gone removes it.
type ConversationScheduleSyncer interface {
	Sync(ctx context.Context, schedule *conversationschedule.Schedule)
	Remove(ctx context.Context, scheduleID pulid.ID)
}

// FireConversationScheduleRequest is one slot of a schedule coming round.
type FireConversationScheduleRequest struct {
	ScheduleID pulid.ID
	TenantInfo pagination.TenantInfo
	// FiredAt is when the slot fired. A retried attempt finds the run it
	// already started recorded at or after it and starts no second one.
	FiredAt int64
}

// FireConversationScheduleResult is what came of a slot.
type FireConversationScheduleResult struct {
	// TurnID is the turn the run started, nil when it did not start one.
	TurnID pulid.ID `json:"turnId,omitempty"`
	// Skipped says why no turn was started: the schedule was deleted or
	// paused, already ran for this slot, or its owner may no longer ask the
	// agent.
	Skipped string `json:"skipped,omitempty"`
	// Gone says the schedule no longer exists, so its Temporal schedule
	// should go too.
	Gone bool `json:"gone,omitempty"`
}

// ConversationScheduleRunner starts the turn a schedule's slot asks for. The
// worker calls it when a Temporal schedule fires.
type ConversationScheduleRunner interface {
	Fire(
		ctx context.Context,
		req FireConversationScheduleRequest,
	) (*FireConversationScheduleResult, error)
}
