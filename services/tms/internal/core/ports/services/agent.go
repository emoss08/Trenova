package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// StartAgentRunForDefinitionRequest starts a background run of one agent.
// The definition is named by id or by its system key; the subject is optional
// and defaults to the organization itself.
type StartAgentRunForDefinitionRequest struct {
	DefinitionID pulid.ID
	SystemKey    string
	SubjectType  agent.SubjectType
	SubjectID    pulid.ID
	Trigger      agent.RunTrigger
	EventKind    agent.EventKind
	// Slot is the schedule slot a sweep claimed. It keys the workflow id so two
	// sweeps that both see the slot start one run, not two.
	Slot       int64
	TenantInfo pagination.TenantInfo
}

// StartInlineAgentRunRequest opens a run for an agent whose reasoning happens in-process
// rather than in a Temporal workflow.
type StartInlineAgentRunRequest struct {
	AgentType         agent.Type
	AgentDefinitionID pulid.ID
	SubjectType       agent.SubjectType
	SubjectID         pulid.ID
	PromptVersion     string
	Summary           string
	Trigger           agent.RunTrigger
	TenantInfo        pagination.TenantInfo
}

type AgentRunService interface {
	StartForDefinition(
		ctx context.Context,
		req *StartAgentRunForDefinitionRequest,
		actor *RequestActor,
	) (*agent.AgentRun, error)
	// StartInline records a run for a deterministic agent that reasons in-process. Such
	// agents still need the run row so their proposals, decisions, and audit trail are
	// indistinguishable from an LLM-backed agent's.
	StartInline(
		ctx context.Context,
		req *StartInlineAgentRunRequest,
		actor *RequestActor,
	) (*agent.AgentRun, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentRunConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRun], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type AgentProposalService interface {
	List(
		ctx context.Context,
		req *repositories.ListAgentProposalRequest,
	) (*pagination.ListResult[*agent.AgentProposal], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentProposalConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentProposal], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

type FlagAgentExceptionRequest struct {
	RunID          pulid.ID
	Category       agent.ExceptionCategory
	Severity       agent.Severity
	SubjectType    agent.SubjectType
	SubjectID      pulid.ID
	AttemptSummary string
	Evidence       []agent.EvidenceRef
	BlastRadius    int
	TenantInfo     pagination.TenantInfo
}

type ResolveAgentExceptionRequest struct {
	ID              pulid.ID
	ResolutionState agent.ResolutionState
	ResolutionNotes string
	TenantInfo      pagination.TenantInfo
}

type AgentExceptionService interface {
	Flag(
		ctx context.Context,
		req *FlagAgentExceptionRequest,
		actor *RequestActor,
	) (*agent.AgentException, error)
	List(
		ctx context.Context,
		req *repositories.ListAgentExceptionRequest,
	) (*pagination.ListResult[*agent.AgentException], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentExceptionConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentException], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentExceptionByIDRequest,
	) (*agent.AgentException, error)
	Resolve(
		ctx context.Context,
		req *ResolveAgentExceptionRequest,
		actor *RequestActor,
	) (*agent.AgentException, error)
}

type DecideAgentProposalRequest struct {
	ProposalID    pulid.ID
	Decision      agent.DecisionType
	Modifications map[string]any
	ReasonCode    string
	TenantInfo    pagination.TenantInfo
}

// AgentTrustService keeps the earned-autonomy ledger. Every decision on a
// proposal and every failed execution is an outcome for the agent and tool
// concerned; a long enough streak of clean approvals moves the tool up one
// tier on the agent when the organization allows it, and a setback takes an
// earned tier back.
type AgentTrustService interface {
	RecordDecision(ctx context.Context, proposal *agent.AgentProposal, decision *agent.AgentDecision) error
	RecordExecutionFailure(ctx context.Context, proposal *agent.AgentProposal) error
	ListForDefinition(ctx context.Context, req repositories.ListToolTrustRequest) ([]*agent.ToolTrust, error)
}

type AgentDecisionService interface {
	Decide(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *RequestActor,
	) (*agent.AgentDecision, error)
}

type UpdateAgentControlRequest struct {
	ShadowMode bool
	// EarnedAutonomy and PromotionThreshold are optional so a client that
	// only knows the pause switch leaves them as they are.
	EarnedAutonomy         *bool
	PromotionThreshold     *int
	BillingAgentEnabled    *bool
	DecisionTimeoutSeconds *int
	TenantInfo             pagination.TenantInfo
}

type AgentControlService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*tenant.AgentControl, error)
	Update(
		ctx context.Context,
		req *UpdateAgentControlRequest,
		actor *RequestActor,
	) (*tenant.AgentControl, error)
}
