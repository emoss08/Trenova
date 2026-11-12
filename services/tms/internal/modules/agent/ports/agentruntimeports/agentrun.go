package agentruntimeports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentRunRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Reader RunReader                `json:"-"`
}

type ListAgentRunConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
	Reader  RunReader                `json:"-"`
}

type GetAgentRunByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
	Reader     RunReader              `json:"-"`
}

type CountOpenAgentRunsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	SubjectID    pulid.ID
	// Triggers limits the count to runs started these ways; empty counts all.
	Triggers []agent.RunTrigger
	// WorkingOnly counts only runs still working, not those waiting on a
	// person's decision: an event agent's cap is on what it is doing.
	WorkingOnly bool
	// WorkflowOnly counts only runs a workflow carries, as the subject's
	// one-open-run index does; an inline run is not the subject's run.
	WorkflowOnly bool
}

// CountAgentRunsSinceRequest counts an agent's runs started at or after an
// instant, whatever became of them; a run that failed still spent its start.
type CountAgentRunsSinceRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	Since        int64
}

// ListAgentRunsByIDsRequest reads several runs at once, for a loader that
// resolves the run behind a page of proposals or plans.
type ListAgentRunsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
	Reader     RunReader
}

type AgentRunRepository interface {
	ListByIDs(ctx context.Context, req *ListAgentRunsByIDsRequest) ([]*agent.AgentRun, error)
	ListTranscriptsByIDs(
		ctx context.Context,
		req *ListAgentRunsByIDsRequest,
	) ([]*agent.AgentRun, error)
	List(
		ctx context.Context,
		req *ListAgentRunRequest,
	) (*pagination.ListResult[*agent.AgentRun], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentRunConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRun], error)
	GetByID(ctx context.Context, req GetAgentRunByIDRequest) (*agent.AgentRun, error)
	Create(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
	Update(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
	CountOpen(ctx context.Context, req *CountOpenAgentRunsRequest) (int, error)
	// ListOpenSubjects is which of the subjects already have an open run of
	// the agent, in one read: what a batch skips rather than doubles up on.
	ListOpenSubjects(ctx context.Context, req *ListOpenSubjectsRequest) ([]pulid.ID, error)
	// FailAbandoned ends, in every organization, the runs still marked as
	// working long after any run finishes: their workflow is gone, and left
	// as they are they would say an agent is working for ever.
	FailAbandoned(ctx context.Context, req FailAbandonedRunsRequest) (int, error)
	// CloseDecided completes a workflow-carried run that waits on decisions
	// once none of its proposals is pending. It returns nil when the run was
	// not closed: it is not waiting, or something is still pending.
	CloseDecided(ctx context.Context, req *CloseDecidedRunRequest) (*agent.AgentRun, error)
	// CloseSettled completes, in every organization, the waiting runs whose
	// proposals all expired.
	CloseSettled(ctx context.Context, req *CloseSettledRunsRequest) ([]*agent.AgentRun, error)
	// FailWaiting fails a run that waits on decisions; nil when it was not.
	FailWaiting(ctx context.Context, req *FailWaitingRunRequest) (*agent.AgentRun, error)
	CountSince(ctx context.Context, req CountAgentRunsSinceRequest) (int, error)
}

type ListOpenSubjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	SubjectIDs   []pulid.ID
}

type CloseDecidedRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Now        int64
}

type FailWaitingRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Reason     string
	Now        int64
}

type CloseSettledRunsRequest struct {
	Now   int64
	Limit int
}

// FailAbandonedRunsRequest names the runs to end: those still working that
// were created before StartedBefore.
type FailAbandonedRunsRequest struct {
	StartedBefore int64
	Now           int64
	Reason        string
}
