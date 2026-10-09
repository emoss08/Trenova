package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrTooManyWaits = errors.New("the agent already has as many open waits here as it may")

type GetAgentWaitRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	// ThreadID and UserID, when set, read the wait only as one of that
	// person's conversation's.
	ThreadID pulid.ID
	UserID   pulid.ID
}

type ListThreadWaitsRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
}

type ListOpenWaitsWatchingRequest struct {
	TenantInfo pagination.TenantInfo
	Kinds      []agentwait.Kind
	WatchIDs   []pulid.ID
}

type ListOpenWaitsOfKindsRequest struct {
	TenantInfo pagination.TenantInfo
	Kinds      []agentwait.Kind
	Limit      int
}

type ListOpenWaitsByThreadsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ThreadIDs  []pulid.ID
}

type ListOverdueWaitsRequest struct {
	ExpiredBefore int64
	Limit         int
}

type UpdateAgentWaitConditionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Condition  *agentwait.Condition
}

type ResolveAgentWaitRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agentwait.Status
	Outcome    string
	ResolvedAt int64
}

type MarkAgentWaitResumedRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	TurnID     pulid.ID
	RunID      pulid.ID
}

type SetAgentWaitDueRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	DueAt      *int64
	WorkflowID string
}

type AgentWaitRepository interface {
	Insert(ctx context.Context, entity *agentwait.Wait) (*agentwait.Wait, error)
	Get(ctx context.Context, req *GetAgentWaitRequest) (*agentwait.Wait, error)
	ListByThread(ctx context.Context, req *ListThreadWaitsRequest) ([]*agentwait.Wait, error)
	ListOpenWatching(
		ctx context.Context,
		req *ListOpenWaitsWatchingRequest,
	) ([]*agentwait.Wait, error)
	ListOpenOfKinds(
		ctx context.Context,
		req *ListOpenWaitsOfKindsRequest,
	) ([]*agentwait.Wait, error)
	// ListOpenByThreads is the open waits of the person's conversations,
	// keyed by conversation, for the Desk's rail.
	ListOpenByThreads(
		ctx context.Context,
		req *ListOpenWaitsByThreadsRequest,
	) (map[pulid.ID][]*agentwait.Wait, error)
	// ListOverdueAcrossTenants is every open wait, in every organization,
	// that should have ended before the time given: one whose workflow is
	// gone without having closed it.
	ListOverdueAcrossTenants(
		ctx context.Context,
		req *ListOverdueWaitsRequest,
	) ([]*agentwait.Wait, error)
	UpdateCondition(ctx context.Context, req *UpdateAgentWaitConditionRequest) error
	SetDue(ctx context.Context, req *SetAgentWaitDueRequest) error
	// Resolve closes an open wait. It reports false, with the wait as it
	// stands, when the wait was no longer open: another attempt or a cancel
	// closed it first.
	Resolve(ctx context.Context, req *ResolveAgentWaitRequest) (*agentwait.Wait, bool, error)
	MarkResumed(ctx context.Context, req *MarkAgentWaitResumedRequest) error
}
