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

// Ask answers a quick question from the palette. It is the same turn as any
// other, on a thread created for the question and hidden until it is kept:
// the general assistant answers it, the page and the mentions ride along,
// and what the turn produced is saved so "Open in Desk" has somewhere to go.
func (s *Service) Ask(
	ctx context.Context,
	req *services.AskRequest,
	actor *services.RequestActor,
	emit services.AssistantStreamEmitter,
) (*services.SendMessageResult, error) {
	if emit == nil {
		emit = func(services.StreamEvent) {}
	}

	content := strings.TrimSpace(req.Content)
	multiErr := errortypes.NewMultiError()
	if content == "" {
		multiErr.Add("content", errortypes.ErrRequired, "Question cannot be empty")
	}
	page := req.Page.Normalized()
	if page != nil {
		page.Validate("context", multiErr)
	}
	validateMentions(req.Mentions, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	definition, err := s.askAgent(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	thread, err := s.StartThread(ctx, &services.StartThreadRequest{
		AgentDefinitionID: definition.ID,
		Title:             truncateRunes(content, maxTitleRunes),
		TenantInfo:        req.TenantInfo,
		Origin:            conversation.ThreadOriginAsk,
	}, actor)
	if err != nil {
		return nil, err
	}

	emit(services.StreamEvent{Event: services.AssistantEventThread, Data: thread})

	return s.SendMessageStream(ctx, &services.SendMessageRequest{
		ThreadID:   thread.ID,
		Content:    content,
		Page:       page,
		TenantInfo: req.TenantInfo,
		Mentions:   req.Mentions,
	}, actor, emit)
}

// askAgent picks who answers a quick question: the organization's general
// assistant when one is enabled for chat, else the first chat agent that is.
// An organization with no chat agent enabled has nothing to ask, and is told
// so rather than handed a run that would refuse.
func (s *Service) askAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentdefinition.Definition, error) {
	result, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: askAgentPageLimit},
		},
		EnabledOnly: true,
		ChatOnly:    true,
	})
	if err != nil {
		return nil, err
	}

	var fallback *agentdefinition.Definition
	for _, definition := range result.Items {
		if definition == nil {
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
		return nil, errortypes.NewBusinessError(
			"No assistant is enabled for quick questions. Enable a chat agent in AI Control.",
		)
	}

	return fallback, nil
}
