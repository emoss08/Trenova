package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentPlanByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentPlansByThreadRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentPlanConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
	// ExcludeShadowDefinitions hides plans whose agent is in shadow, the way
	// the proposal list does; a plan nobody may decide is not activity.
	ExcludeShadowDefinitions bool `json:"-"`
}

// UpdateAgentPlanStatusRequest moves a plan between statuses. FromStatus, when
// set, makes the move conditional so two decisions racing each other cannot
// both win.
type UpdateAgentPlanStatusRequest struct {
	ID              pulid.ID
	TenantInfo      pagination.TenantInfo
	Status          agent.PlanStatus
	FromStatus      agent.PlanStatus
	DecidedByUserID pulid.ID
	DecidedAt       int64
	// CommitsAt, when set, is when an approval in its undo window goes
	// through; it clears any earlier undo.
	CommitsAt int64
	// Reopen clears the decision an approval in its undo window recorded, for
	// a plan going back to waiting on its decider. UndoneByUserID, when set,
	// says it was the decider who took it back, and when.
	Reopen         bool
	UndoneByUserID pulid.ID
	UndoneAt       int64
}

// RecordAgentPlanProgressRequest writes how far execution got: the count of
// steps that ran, and the step that failed with its error when one did.
type RecordAgentPlanProgressRequest struct {
	ID             pulid.ID
	TenantInfo     pagination.TenantInfo
	Status         agent.PlanStatus
	CompletedSteps int
	FailedStep     *int
	FailureError   string
}

// ExpireAgentPlansRequest expires every pending plan, in any tenant, whose
// window has closed. Unscoped like the proposal expiry: it is the sweeper's.
type ExpireAgentPlansRequest struct {
	Before int64
}

type ExpireAgentPlansByDefinitionRequest struct {
	AgentDefinitionID pulid.ID
	TenantInfo        pagination.TenantInfo
}

type ListAgentPlansByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentPlanRepository interface {
	ListByIDs(ctx context.Context, req ListAgentPlansByIDsRequest) ([]*agent.AgentPlan, error)
	Create(ctx context.Context, entity *agent.AgentPlan) (*agent.AgentPlan, error)
	GetByID(ctx context.Context, req GetAgentPlanByIDRequest) (*agent.AgentPlan, error)
	ListByThread(ctx context.Context, req ListAgentPlansByThreadRequest) ([]*agent.AgentPlan, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentPlanConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentPlan], error)
	UpdateStatus(ctx context.Context, req UpdateAgentPlanStatusRequest) (*agent.AgentPlan, error)
	RecordProgress(
		ctx context.Context,
		req RecordAgentPlanProgressRequest,
	) (*agent.AgentPlan, error)
	ExpirePending(ctx context.Context, req ExpireAgentPlansRequest) (int, error)
	ExpirePendingByDefinition(
		ctx context.Context,
		req ExpireAgentPlansByDefinitionRequest,
	) (int, error)
}
