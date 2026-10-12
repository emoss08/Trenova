package agenttestpromptservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Definitions agentdefinitionports.AgentDefinitionRepository
	Prompts     agentdefinitionports.AgentTestPromptRepository
}

type Service struct {
	definitions agentdefinitionports.AgentDefinitionRepository
	prompts     agentdefinitionports.AgentTestPromptRepository
}

var _ agentdefinitionports.AgentTestPromptService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{definitions: p.Definitions, prompts: p.Prompts}
}

func (s *Service) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentTestPromptsRequest,
) ([]*agentdefinition.TestPrompt, error) {
	if err := s.agentExists(ctx, req); err != nil {
		return nil, err
	}
	return s.prompts.List(ctx, req)
}

func (s *Service) Keep(
	ctx context.Context,
	req *agentdefinitionports.KeepAgentTestPromptRequest,
) (*agentdefinition.TestPrompt, error) {
	prompt := &agentdefinition.TestPrompt{
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		AgentDefinitionID: req.AgentDefinitionID,
		Prompt:            req.Prompt,
	}
	prompt.CreatedByID = typeutils.IDPtr(req.CreatedByID)

	multiErr := errortypes.NewMultiError()
	prompt.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	scope := &agentdefinitionports.ListAgentTestPromptsRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
	}
	if err := s.agentExists(ctx, scope); err != nil {
		return nil, err
	}
	count, err := s.prompts.Count(ctx, scope)
	if err != nil {
		return nil, err
	}
	if count >= agentdefinition.MaxTestPrompts {
		return nil, errortypes.NewBusinessError(
			"An agent keeps at most 20 prompts. Remove one before keeping another.",
		)
	}

	if err = s.prompts.Create(ctx, prompt); err != nil {
		return nil, err
	}
	return prompt, nil
}

func (s *Service) Delete(ctx context.Context, req *agentdefinitionports.DeleteAgentTestPromptRequest) error {
	return s.prompts.Delete(ctx, req)
}

func (s *Service) agentExists(ctx context.Context, req *agentdefinitionports.ListAgentTestPromptsRequest) error {
	_, err := s.definitions.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	return err
}
