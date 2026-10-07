package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentDefinitionVersionsRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Limit             int
}

type GetAgentDefinitionVersionRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Version           int64
}

// GetAgentDefinitionVersionAtRequest asks for the latest version saved at or
// before a version number: the agent as a person last loaded it, even when a
// save in between left no version of its own (an earned tier, a removed
// delegate).
type GetAgentDefinitionVersionAtRequest = GetAgentDefinitionVersionRequest

type AgentDefinitionVersionRepository interface {
	Create(ctx context.Context, version *agentdefinition.DefinitionVersion) error
	// List returns the newest versions first, with their authors and
	// without their snapshots.
	List(
		ctx context.Context,
		req *ListAgentDefinitionVersionsRequest,
	) ([]*agentdefinition.DefinitionVersion, error)
	Get(
		ctx context.Context,
		req *GetAgentDefinitionVersionRequest,
	) (*agentdefinition.DefinitionVersion, error)
	// LatestAt returns the newest version at or before the one asked for, with
	// its author, or nil when there is none.
	LatestAt(
		ctx context.Context,
		req *GetAgentDefinitionVersionAtRequest,
	) (*agentdefinition.DefinitionVersion, error)
}
