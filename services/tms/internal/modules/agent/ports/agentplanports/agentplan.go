package agentplanports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
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
}

// UpdateAgentPlanStatusRequest moves a plan between statuses. FromStatus, when
// set, makes the move conditional so two decisions racing each other cannot
// both win.
type UpdateAgentPlanStatusRequest struct {
	ID              pulid.ID
	TenantInfo      pagination.TenantInfo
	Status          agentplan.PlanStatus
	FromStatus      agentplan.PlanStatus
	DecidedByUserID pulid.ID
	DecidedAt       int64
	// CommitsAt, when set, is when an approval in its undo window goes
	// through; it clears any earlier undo.
	CommitsAt int64
	// Approval is what the person approved with, kept with a window that
	// opens so a commit made without its workflow carries the same approval.
	Approval PlanApproval
	// Reopen clears the decision an approval in its undo window recorded, for
	// a plan going back to waiting on its decider. UndoneByUserID, when set,
	// says it was the decider who took it back, and when.
	Reopen         bool
	UndoneByUserID pulid.ID
	UndoneAt       int64
}

type PlanApproval struct {
	ReasonCode    string
	Note          string
	PreviewDigest string
}

// RecordAgentPlanProgressRequest writes how far execution got: the count of
// steps that ran, and the step that failed with its error when one did.
type RecordAgentPlanProgressRequest struct {
	ID             pulid.ID
	TenantInfo     pagination.TenantInfo
	Status         agentplan.PlanStatus
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
	ListByIDs(ctx context.Context, req ListAgentPlansByIDsRequest) ([]*agentplan.AgentPlan, error)
	Create(ctx context.Context, entity *agentplan.AgentPlan) (*agentplan.AgentPlan, error)
	GetByID(ctx context.Context, req GetAgentPlanByIDRequest) (*agentplan.AgentPlan, error)
	ListByThread(
		ctx context.Context,
		req ListAgentPlansByThreadRequest,
	) ([]*agentplan.AgentPlan, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentPlanConnectionRequest,
	) (*pagination.CursorListResult[*agentplan.AgentPlan], error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateAgentPlanStatusRequest,
	) (*agentplan.AgentPlan, error)
	RecordProgress(
		ctx context.Context,
		req *RecordAgentPlanProgressRequest,
	) (*agentplan.AgentPlan, error)
	ExpirePending(ctx context.Context, req ExpireAgentPlansRequest) (int, error)
	ExpirePendingByDefinition(
		ctx context.Context,
		req ExpireAgentPlansByDefinitionRequest,
	) (int, error)
	ExpirePendingByThread(ctx context.Context, req ExpireAgentPlansByThreadRequest) (int, error)
}

type ExpireAgentPlansByThreadRequest struct {
	ThreadID   pulid.ID              `json:"threadId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}
