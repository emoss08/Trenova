package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAgentTestPromptsRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
}

type DeleteAgentTestPromptRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	ID                pulid.ID
}

type AgentTestPromptRepository interface {
	// List returns an agent's prompts, oldest first.
	List(ctx context.Context, req *ListAgentTestPromptsRequest) ([]*agentdefinition.TestPrompt, error)
	Count(ctx context.Context, req *ListAgentTestPromptsRequest) (int, error)
	Create(ctx context.Context, prompt *agentdefinition.TestPrompt) error
	Delete(ctx context.Context, req *DeleteAgentTestPromptRequest) error
}
