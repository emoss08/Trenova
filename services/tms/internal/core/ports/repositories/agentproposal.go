package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentProposalRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	ExcludeShadowDefinitions bool                     `json:"-"`
}

type ListAgentProposalConnectionRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	Cursor                   pagination.CursorInfo    `json:"-"`
	Columns                  []string                 `json:"-"`
	ExcludeShadowDefinitions bool                     `json:"-"`
}

type GetAgentProposalByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type UpdateAgentProposalStatusRequest struct {
	ID     pulid.ID             `json:"id"`
	Status agent.ProposalStatus `json:"status"`
	// FromStatus, when set, makes the update conditional: the row changes only
	// if it still holds this status, and a race that lost is reported as a
	// conflict rather than silently overwriting the winner.
	FromStatus agent.ProposalStatus  `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// RecordAgentProposalExecutionRequest records what happened when an approved
// proposal's tool ran, so "approved" and "approved and done" stay distinguishable.
type RecordAgentProposalExecutionRequest struct {
	ID             pulid.ID             `json:"id"`
	Status         agent.ProposalStatus `json:"status"`
	ExecutedAt     *int64               `json:"executedAt"`
	ExecutionError string               `json:"executionError"`
	// ExecutionResult is what the run made, when the tool reported it. It
	// is written with the status, so a failure clears an earlier result.
	ExecutionResult *agent.ToolExecutionResult `json:"executionResult"`
	// EgressClass is where the write reached as it ran, with any change the
	// approver made. Empty leaves the proposal's as it was.
	EgressClass           agent.EgressClass     `json:"egressClass"`
	TenantInfo            pagination.TenantInfo `json:"-"`
	ExecutedByUserID      pulid.ID              `json:"executedByUserId"`
	ExecutedTargetVersion *int64                `json:"executedTargetVersion"`
}

// RecordAgentProposalSimulationRequest stores what a write would have
// changed, in place of an execution, for a proposal cleared while its agent
// was in simulation.
type RecordAgentProposalSimulationRequest struct {
	ID               pulid.ID
	TenantInfo       pagination.TenantInfo
	SimulatedAt      int64
	Simulation       *agent.ToolSimulation
	ExecutedByUserID pulid.ID
}

// CountExecutedToolRequest counts how many times one agent has executed one
// tool since an instant, for the daily tool cap. The agent is reached through
// the run, since a proposal carries only its run.
type CountExecutedToolRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	ToolName     string
	Since        int64
}

// ListAgentProposalsByThreadRequest fetches every proposal raised during one
// assistant conversation, so reopening a thread shows what is still waiting on a
// decision rather than only what the last turn returned.
type ListAgentProposalsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentProposalsByThreadRequest struct {
	ThreadID   pulid.ID              `json:"threadId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// ListAgentProposalsByRunRequest reads every proposal one run raised, in
// the order it raised them.
type ListAgentProposalsByRunRequest struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ExpireAgentProposalsByRunRequest struct {
	RunID      pulid.ID              `json:"runId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// ExpireAgentProposalsRequest expires every pending proposal, in any tenant,
// whose expiry has passed. It is deliberately unscoped: it is the sweeper's
// request, and the sweeper runs for the whole system.
type ExpireAgentProposalsRequest struct {
	Before int64 `json:"before"`
}

// ListPendingProposalsForReminderRequest finds proposals from background
// runs that have waited since before Before with nobody told twice. Unscoped,
// like the expiry: it is the sweeper's request.
type ListPendingProposalsForReminderRequest struct {
	Before int64 `json:"before"`
	Now    int64 `json:"now"`
	Limit  int   `json:"limit"`
}

type MarkProposalsRemindedRequest struct {
	IDs []pulid.ID `json:"ids"`
	At  int64      `json:"at"`
}

type ListAgentProposalsByPlanRequest struct {
	PlanID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// SkipPendingByPlanRequest marks every still-pending step of a plan as
// skipped, once an earlier step has failed.
type SkipPendingByPlanRequest struct {
	PlanID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentProposalRepository interface {
	List(
		ctx context.Context,
		req *ListAgentProposalRequest,
	) (*pagination.ListResult[*agent.AgentProposal], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentProposalConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentProposal], error)
	GetByID(ctx context.Context, req GetAgentProposalByIDRequest) (*agent.AgentProposal, error)
	Create(ctx context.Context, entity *agent.AgentProposal) (*agent.AgentProposal, error)
	ListByThread(
		ctx context.Context,
		req ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
	ListByIDs(
		ctx context.Context,
		req ListAgentProposalsByIDsRequest,
	) ([]*agent.AgentProposal, error)
	UpdateStatus(
		ctx context.Context,
		req UpdateAgentProposalStatusRequest,
	) (*agent.AgentProposal, error)
	ExpirePendingByRun(ctx context.Context, req ExpireAgentProposalsByRunRequest) (int, error)
	ExpirePending(ctx context.Context, req ExpireAgentProposalsRequest) (int, error)
	ListPendingForReminder(
		ctx context.Context,
		req ListPendingProposalsForReminderRequest,
	) ([]*agent.AgentProposal, error)
	MarkReminded(ctx context.Context, req MarkProposalsRemindedRequest) (int, error)
	ListByPlan(
		ctx context.Context,
		req ListAgentProposalsByPlanRequest,
	) ([]*agent.AgentProposal, error)
	ListByRun(
		ctx context.Context,
		req ListAgentProposalsByRunRequest,
	) ([]*agent.AgentProposal, error)
	SkipPendingByPlan(ctx context.Context, req SkipPendingByPlanRequest) (int, error)
	RecordExecution(
		ctx context.Context,
		req RecordAgentProposalExecutionRequest,
	) (*agent.AgentProposal, error)
	RecordSimulation(
		ctx context.Context,
		req RecordAgentProposalSimulationRequest,
	) (*agent.AgentProposal, error)
	CountExecutedTool(ctx context.Context, req CountExecutedToolRequest) (int, error)
}
