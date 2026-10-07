package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type KeepAgentTestPromptRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Prompt            string
	CreatedByID       pulid.ID
}

// AgentTestPromptService keeps the prompts a person tries an agent with, so
// an edit can be checked against the same questions each time.
type AgentTestPromptService interface {
	List(
		ctx context.Context,
		req *repositories.ListAgentTestPromptsRequest,
	) ([]*agentdefinition.TestPrompt, error)
	Keep(ctx context.Context, req *KeepAgentTestPromptRequest) (*agentdefinition.TestPrompt, error)
	Delete(ctx context.Context, req *repositories.DeleteAgentTestPromptRequest) error
}
