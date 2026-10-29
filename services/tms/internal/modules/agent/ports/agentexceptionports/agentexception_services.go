package agentexceptionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FlagAgentExceptionRequest struct {
	RunID          pulid.ID
	Category       agentexception.ExceptionCategory
	Severity       agentexception.Severity
	SubjectType    agent.SubjectType
	SubjectID      pulid.ID
	AttemptSummary string
	Evidence       []agentevidence.EvidenceRef
	BlastRadius    int
	TenantInfo     pagination.TenantInfo
}

type ResolveAgentExceptionRequest struct {
	ID              pulid.ID
	ResolutionState agentexception.ResolutionState
	ResolutionNotes string
	TenantInfo      pagination.TenantInfo
}

type AgentExceptionService interface {
	Flag(
		ctx context.Context,
		req *FlagAgentExceptionRequest,
		actor *permissionports.RequestActor,
	) (*agentexception.AgentException, error)
	List(
		ctx context.Context,
		req *ListAgentExceptionRequest,
	) (*pagination.ListResult[*agentexception.AgentException], error)
	ListConnection(
		ctx context.Context,
		req *ListAgentExceptionConnectionRequest,
	) (*pagination.CursorListResult[*agentexception.AgentException], error)
	GetByID(
		ctx context.Context,
		req GetAgentExceptionByIDRequest,
	) (*agentexception.AgentException, error)
	Resolve(
		ctx context.Context,
		req *ResolveAgentExceptionRequest,
		actor *permissionports.RequestActor,
	) (*agentexception.AgentException, error)
}
