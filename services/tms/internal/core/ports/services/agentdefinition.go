package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// SaveAgentDefinitionRequest carries a create or update. There is no system
// prompt here by design; see the agentdefinition domain.
type SaveAgentDefinitionRequest struct {
	ID              pulid.ID
	Name            string
	Description     string
	Kind            agentdefinition.Kind
	Focus           string
	ToolNames       []string
	AutonomyCeiling agent.AutonomyTier
	Enabled         bool
	Version         int64
	TenantInfo      pagination.TenantInfo
}

// AgentTemplateDescriptor describes a template and the tools it permits, so the
// configuration UI offers exactly the choosable set.
type AgentTemplateDescriptor struct {
	Kind            agentdefinition.Kind  `json:"kind"`
	Label           string                `json:"label"`
	Description     string                `json:"description"`
	MutatingAllowed bool                  `json:"mutatingAllowed"`
	AvailableTools  []AgentToolDescriptor `json:"availableTools"`
}

type AgentDefinitionService interface {
	List(
		ctx context.Context,
		req *repositories.ListAgentDefinitionRequest,
	) (*pagination.ListResult[*agentdefinition.Definition], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
	Create(
		ctx context.Context,
		req *SaveAgentDefinitionRequest,
		actor *RequestActor,
	) (*agentdefinition.Definition, error)
	Update(
		ctx context.Context,
		req *SaveAgentDefinitionRequest,
		actor *RequestActor,
	) (*agentdefinition.Definition, error)
	Delete(
		ctx context.Context,
		req repositories.DeleteAgentDefinitionRequest,
		actor *RequestActor,
	) error
	// Templates lists what an organization may build an agent from.
	Templates() []AgentTemplateDescriptor
}
