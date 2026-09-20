package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type TurnRequest struct {
	Definition *agentdefinition.Definition
	Actor      *serviceports.RequestActor
	History    []conversation.Message
	Input      string
	Page       *agentdefinition.PageContext
	// PreferredProviderID is the reader's chosen model for this conversation.
	PreferredProviderID pulid.ID
}

type TurnResult struct {
	Reply    string
	Decision agentguard.Decision
	Messages []conversation.Message
	Actions  []serviceports.PendingAction
	Model    string
	Provider pulid.ID
}

func (s *Service) Run(ctx context.Context, req *TurnRequest) (*TurnResult, error) {
	return s.RunObserved(ctx, req, nil)
}

func (s *Service) RunObserved(
	ctx context.Context,
	req *TurnRequest,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	decision := s.guard.Evaluate(ctx, req.Actor.TenantInfo(), req.Input)
	if !decision.Allowed {
		emit(refusedEvent(decision))

		return &TurnResult{
			Reply:    decision.Message,
			Decision: decision,
			Messages: []conversation.Message{
				scopedMessage(conversation.RoleUser, req.Input, decision, true),
				scopedMessage(conversation.RoleAssistant, decision.Message, decision, true),
			},
		}, nil
	}

	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventAccepted,
		Data: serviceports.AssistantAcceptedEvent{
			Content:       req.Input,
			ScopeStage:    string(decision.Stage),
			ScopeCategory: string(decision.Category),
		},
	})

	runtimeContext := s.buildContext(ctx, req)

	run, err := s.runtime.Run(ctx, &serviceports.RunRequest{
		Definition:          req.Definition,
		Actor:               req.Actor,
		Context:             runtimeContext,
		History:             req.History,
		Input:               req.Input,
		Emit:                emit,
		PreferredProviderID: req.PreferredProviderID,
	})
	if err != nil {
		return nil, err
	}

	result := &TurnResult{
		Reply:    run.Reply,
		Decision: decision,
		Messages: run.Messages,
		Actions:  run.Actions,
		Model:    run.Model,
		Provider: run.ProviderID,
	}

	if len(result.Messages) > 0 {
		result.Messages[0] = scopedMessage(conversation.RoleUser, req.Input, decision, false)
	}

	if run.OutputRefused {
		result.Decision = outputDecision(run, result.Messages)
	}

	return result, nil
}

func (s *Service) buildContext(
	ctx context.Context,
	req *TurnRequest,
) agentdefinition.RuntimeContext {
	if s.contexts == nil {
		return agentdefinition.RuntimeContext{Trigger: agent.RunTriggerChat, Page: req.Page}
	}

	runtimeContext, err := s.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: req.Definition,
		Actor:      req.Actor,
		Trigger:    agent.RunTriggerChat,
		Page:       req.Page,
	})
	if err != nil {
		s.logger.Warn("assistant context could not be built", zap.Error(err))

		return agentdefinition.RuntimeContext{Trigger: agent.RunTriggerChat, Page: req.Page}
	}

	return runtimeContext
}

func scopedMessage(
	role conversation.Role,
	content string,
	decision agentguard.Decision,
	refused bool,
) conversation.Message {
	return conversation.Message{
		Role:          role,
		Content:       content,
		ScopeStage:    string(decision.Stage),
		ScopeCategory: string(decision.Category),
		ScopeReason:   string(decision.Reason),
		Refused:       refused,
	}
}

func outputDecision(run *serviceports.RunResult, messages []conversation.Message) agentguard.Decision {
	decision := agentguard.Decision{
		Allowed:     false,
		Stage:       agentguard.StageOutput,
		Message:     run.Reply,
		MatchedRule: run.OutputRule,
	}
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role == conversation.RoleAssistant && message.Refused {
			decision.Category = agentguard.Category(message.ScopeCategory)
			decision.Reason = agentguard.Reason(message.ScopeReason)

			break
		}
	}

	return decision
}

func refusedEvent(decision agentguard.Decision) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventRefused,
		Data: serviceports.AssistantRefusedEvent{
			Message:  decision.Message,
			Stage:    string(decision.Stage),
			Category: string(decision.Category),
			Reason:   string(decision.Reason),
		},
	}
}
