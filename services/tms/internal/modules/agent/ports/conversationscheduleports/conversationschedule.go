package conversationscheduleports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetConversationScheduleRequest reads one schedule. A person reads only
// their own, so UserID is set on every read made for a request; the worker
// firing a schedule leaves it empty and acts for whoever owns it.
type GetConversationScheduleRequest struct {
	ID         pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// ConversationScheduleOrder is the order a list of schedules is read in.
type ConversationScheduleOrder string

const (
	// ConversationScheduleOrderNewest reads the latest made first, the order
	// a conversation's cards were asked in.
	ConversationScheduleOrderNewest ConversationScheduleOrder = "Newest"

	// ConversationScheduleOrderNextRun reads the schedules that run soonest
	// first and the paused ones after them, the order a person scans what is
	// coming up.
	ConversationScheduleOrderNextRun ConversationScheduleOrder = "NextRun"
)

// ListConversationSchedulesRequest pages through one person's schedules, in
// one conversation when ThreadID is set, newest first unless Order says
// otherwise.
type ListConversationSchedulesRequest struct {
	UserID     pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Order      ConversationScheduleOrder
	Limit      int
	Offset     int
}

// ListedConversationSchedule is a schedule with the title of the
// conversation its answers post into, read in the same query, so a list
// across conversations can say where each one goes.
type ListedConversationSchedule struct {
	conversationschedule.Schedule

	ThreadTitle string `json:"threadTitle"`
}

// CountConversationSchedulesRequest counts one person's schedules, in one
// conversation when ThreadID is set.
type CountConversationSchedulesRequest struct {
	UserID     pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

// ListConversationSchedulesAcrossTenantsRequest pages through every schedule
// in every tenant, paused or not, in id order.
type ListConversationSchedulesAcrossTenantsRequest struct {
	AfterID pulid.ID
	Limit   int
}

// CreateConversationScheduleRequest saves a schedule together with the
// message that draws its card in the conversation. ThreadTitle names the
// conversation when it has no name yet: one opened with a schedule has no
// reply to be named from until the schedule first runs.
type CreateConversationScheduleRequest struct {
	Schedule    *conversationschedule.Schedule
	Message     conversation.Message
	ThreadTitle string
}

// RecordConversationScheduleRunRequest records that a run started.
type RecordConversationScheduleRunRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	RunAt      int64
	NextRunAt  *int64
	TurnID     pulid.ID
}

type ConversationScheduleRepository interface {
	// Create writes the schedule and its message in one transaction, the
	// message numbered after the conversation's last one. Neither exists
	// without the other: a card with no schedule says it was deleted, and a
	// schedule with no card would run in a conversation that never showed it.
	Create(
		ctx context.Context,
		req CreateConversationScheduleRequest,
	) (*conversationschedule.Schedule, *conversation.Message, error)
	Get(
		ctx context.Context,
		req GetConversationScheduleRequest,
	) (*conversationschedule.Schedule, error)
	List(
		ctx context.Context,
		req ListConversationSchedulesRequest,
	) (*pagination.ListResult[*ListedConversationSchedule], error)
	Count(ctx context.Context, req CountConversationSchedulesRequest) (int, error)
	ListAcrossTenants(
		ctx context.Context,
		req ListConversationSchedulesAcrossTenantsRequest,
	) ([]*conversationschedule.Schedule, error)
	// UpdateState writes what the schedule asks, its cadence, whether it is
	// on and its next slot, guarded by its version.
	UpdateState(
		ctx context.Context,
		schedule *conversationschedule.Schedule,
	) (*conversationschedule.Schedule, error)
	// RecordRun writes the latest run and the next slot. It leaves the
	// version alone: a run is not an edit, and a person pausing the schedule
	// while it fires must not be refused for it.
	RecordRun(ctx context.Context, req RecordConversationScheduleRunRequest) error
	Delete(ctx context.Context, req GetConversationScheduleRequest) error
}
