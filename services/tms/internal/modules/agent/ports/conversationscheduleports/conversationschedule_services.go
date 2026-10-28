package conversationscheduleports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

// ThreadSchedulesRequest names one person's conversation.
type ThreadSchedulesRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// ConversationThreadSchedules settles the schedules of a conversation being
// deleted. Their rows go with the conversation's, by cascade; this reads them
// first and, once the conversation is gone, takes their Temporal schedules
// off the scheduler and tells the owner's tabs, so none fires again and no
// open list keeps showing one.
type ConversationThreadSchedules interface {
	// OfThread reads every schedule the conversation keeps. It is called
	// before the conversation is deleted, while there is still something to
	// read.
	OfThread(
		ctx context.Context,
		req ThreadSchedulesRequest,
	) ([]*conversationschedule.Schedule, error)
	// Forget removes the Temporal schedules of schedules deleted with their
	// conversation and announces each one gone. Like the syncer it never
	// fails the delete: what it could not remove the next reconcile does.
	Forget(ctx context.Context, schedules []*conversationschedule.Schedule)
}

// PlanConversationScheduleRequest is a request to repeat in one of the
// person's conversations, as fields rather than a message: what to ask, how
// often and at what time (Edit), and the clock the time is on, an IANA name
// or one a person says ("EST", "Pacific"), empty for the person's own.
// ProposalID is the approved proposal that asks for it, which its card names.
type PlanConversationScheduleRequest struct {
	ThreadID   pulid.ID
	Edit       conversationschedule.Edit
	Timezone   string
	ProposalID pulid.ID
	Actor      permissionports.RequestActor
}

// PlannedConversationSchedule is the schedule a planned request keeps, and
// Existing the one the conversation already keeps for the same request, time
// and clock, when there is one; then nothing new is kept.
type PlannedConversationSchedule struct {
	Schedule *conversationschedule.Schedule
	Existing *conversationschedule.Schedule
}

// ConversationSchedulePlanner checks and keeps a schedule an agent proposed
// in the conversation it was answering. Plan writes nothing; CreatePlanned
// keeps it, with its card, once the person approved it.
type ConversationSchedulePlanner interface {
	Plan(
		ctx context.Context,
		req PlanConversationScheduleRequest,
	) (*PlannedConversationSchedule, error)
	CreatePlanned(
		ctx context.Context,
		req PlanConversationScheduleRequest,
	) (*PlannedConversationSchedule, error)
}
