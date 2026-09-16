package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentDefinitionByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentDefinitionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	// EnabledOnly narrows to agents a person can actually start a conversation
	// with.
	EnabledOnly bool
}

type DeleteAgentDefinitionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentDefinitionRepository interface {
	List(
		ctx context.Context,
		req *ListAgentDefinitionRequest,
	) (*pagination.ListResult[*agentdefinition.Definition], error)
	GetByID(
		ctx context.Context,
		req GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
	Create(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	Update(
		ctx context.Context,
		entity *agentdefinition.Definition,
	) (*agentdefinition.Definition, error)
	Delete(ctx context.Context, req DeleteAgentDefinitionRequest) error
}
