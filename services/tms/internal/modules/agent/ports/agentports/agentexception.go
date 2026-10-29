package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentExceptionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListAgentExceptionConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type GetAgentExceptionByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type UpdateAgentExceptionResolutionRequest struct {
	ID              pulid.ID                       `json:"id"`
	ResolutionState agentexception.ResolutionState `json:"resolutionState"`
	ResolutionNotes string                         `json:"resolutionNotes"`
	TenantInfo      pagination.TenantInfo          `json:"-"`
}

type ListAgentExceptionsByIDsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentExceptionRepository interface {
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
	ListByIDs(
		ctx context.Context,
		req ListAgentExceptionsByIDsRequest,
	) ([]*agentexception.AgentException, error)
	Create(
		ctx context.Context,
		entity *agentexception.AgentException,
	) (*agentexception.AgentException, error)
	UpdateResolution(
		ctx context.Context,
		req UpdateAgentExceptionResolutionRequest,
	) (*agentexception.AgentException, error)
}
