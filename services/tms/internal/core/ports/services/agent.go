package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StartAgentRunRequest struct {
	AgentType   agent.Type
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
	TenantInfo  pagination.TenantInfo
}

type AgentRunService interface {
	Start(
		ctx context.Context,
		req *StartAgentRunRequest,
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

type AgentDecisionService interface {
	Decide(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *RequestActor,
	) (*agent.AgentDecision, error)
}

type UpdateAgentControlRequest struct {
	ShadowMode             bool
	BillingAgentEnabled    bool
	DecisionTimeoutSeconds int
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
