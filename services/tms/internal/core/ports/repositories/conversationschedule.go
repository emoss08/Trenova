package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
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

// ListConversationSchedulesRequest pages through one person's schedules,
// newest first, in one conversation when ThreadID is set.
type ListConversationSchedulesRequest struct {
	UserID     pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
	Offset     int
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
// message that draws its card in the conversation.
type CreateConversationScheduleRequest struct {
	Schedule *conversationschedule.Schedule
	Message  conversation.Message
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
	) (*pagination.ListResult[*conversationschedule.Schedule], error)
	Count(ctx context.Context, req CountConversationSchedulesRequest) (int, error)
	ListAcrossTenants(
		ctx context.Context,
		req ListConversationSchedulesAcrossTenantsRequest,
	) ([]*conversationschedule.Schedule, error)
	// UpdateState writes whether the schedule is on and its next slot,
	// guarded by its version.
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
