package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// askAgentPageLimit bounds the definitions read to pick the assistant a
// quick question goes to; an organization has a handful of chat agents.
const askAgentPageLimit = 100

// StartAsk opens the thread a quick question from the palette is answered on.
// It is the same turn as any other, on a thread created for the question and
// hidden until it is kept: the general assistant answers it, and what the turn
// produced is saved there so "Open in Desk" has somewhere to go.
func (s *Service) StartAsk(
	ctx context.Context,
	req *services.AskRequest,
	actor *services.RequestActor,
) (*conversation.Thread, error) {
	content := strings.TrimSpace(req.Content)
	multiErr := errortypes.NewMultiError()
	if content == "" {
		multiErr.Add("content", errortypes.ErrRequired, "Question cannot be empty")
	}
	if page := req.Page.Normalized(); page != nil {
		page.Validate("context", multiErr)
	}
	validateMentions(req.Mentions, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	definition, err := s.askAgent(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	return s.StartThread(ctx, &services.StartThreadRequest{
		AgentDefinitionID: definition.ID,
		Title:             truncateRunes(content, maxTitleRunes),
		TenantInfo:        req.TenantInfo,
		Origin:            conversation.ThreadOriginAsk,
	}, actor)
}

// askAgent picks who answers a quick question, among the chat agents the
// person may use: the organization's general assistant when it is one of
// them, else the first. A person with none to ask is told why rather than
// handed a run that would refuse.
func (s *Service) askAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	usable, err := s.usableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, errAgentsWithheld()
	}

	result, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: askAgentPageLimit},
		},
		EnabledOnly: true,
		ChatOnly:    true,
		Audience:    usable.Audience(),
	})
	if err != nil {
		return nil, err
	}

	var fallback *agentdefinition.Definition
	for _, definition := range result.Items {
		if definition == nil || !usable.Allows(definition) {
			continue
		}
		if definition.Template == agentdefinition.TemplateGeneralAssistant {
			return definition, nil
		}
		if fallback == nil {
			fallback = definition
		}
	}
	if fallback == nil {
		return nil, s.noAskAgent(ctx, tenant)
	}

	return fallback, nil
}

// noAskAgent says why there is nobody to ask: no chat agent is enabled at
// all, or none of the enabled ones is open to the person.
func (s *Service) noAskAgent(ctx context.Context, tenant pagination.TenantInfo) error {
	enabled, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 1},
		},
		EnabledOnly: true,
		ChatOnly:    true,
	})
	if err != nil {
		return err
	}
	if len(enabled.Items) > 0 {
		return errortypes.NewAuthorizationError(
			"None of the assistants is open to you for quick questions. An administrator can " +
				"give one of your roles access to one.",
		)
	}

	return errortypes.NewBusinessError(
		"No assistant is enabled for quick questions. Enable a chat agent in AI Control.",
	)
}
