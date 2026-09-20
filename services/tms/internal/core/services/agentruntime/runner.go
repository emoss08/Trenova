package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const exhaustedReply = "I gathered information but could not finish within the allowed number of " +
	"tool calls. Try narrowing the request, or raise the agent's tool call limit."

type Params struct {
	fx.In

	Logger      *zap.Logger
	Completion  serviceports.CompletionService
	QueryTools  serviceports.AgentQueryToolRegistry
	ActionTools serviceports.AgentToolRegistry
	Permissions serviceports.PermissionEngine
	Catalog     *agenttoolcatalog.Catalog
}

type Service struct {
	logger      *zap.Logger
	completion  serviceports.CompletionService
	queryTools  serviceports.AgentQueryToolRegistry
	actionTools serviceports.AgentToolRegistry
	permissions serviceports.PermissionEngine
	catalog     *agenttoolcatalog.Catalog
}

func New(p Params) serviceports.AgentRuntime {
	return &Service{
		logger:      p.Logger.Named("service.agentruntime"),
		completion:  p.Completion,
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
		permissions: p.Permissions,
		catalog:     p.Catalog,
	}
}

func (s *Service) Run(
	ctx context.Context,
	req *serviceports.RunRequest,
) (*serviceports.RunResult, error) {
	emit := req.Emit
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	definition := req.Definition
	budget := definition.MaxToolCalls
	if budget <= 0 {
		budget = agentdefinition.DefaultMaxToolCalls
	}

	runtimeContext := req.Context
	if len(runtimeContext.Tools) == 0 {
		runtimeContext.Tools = s.ToolSummaries(definition)
	}

	result := &serviceports.RunResult{
		Messages: []conversation.Message{{
			Role:    conversation.RoleUser,
			Content: req.Input,
		}},
	}

	tools := s.newToolSet(definition, req.Input)
	runtimeContext.ToolsDisclosed = tools.disclosed

	system := definition.BuildSystemPrompt(runtimeContext)
	messages := toAdapterMessages(req.History)
	messages = append(messages, serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: req.Input,
	})

	sink := func(delta string) {
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventDelta,
			Data:  serviceports.AssistantDeltaEvent{Text: delta},
		})
	}

	for result.ToolCallsUsed < budget {
		completion, err := s.completion.StreamChat(ctx, &serviceports.ChatCompletionRequest{
			TenantInfo:          req.Actor.TenantInfo(),
			System:              system,
			Messages:            messages,
			Tools:               tools.specs,
			PreferredProviderID: definition.PreferredProviderID,
		}, sink)
		if err != nil {
			return nil, err
		}

		result.Model = completion.ModelIdentifier
		result.ProviderID = completion.ProviderID

		if len(completion.ToolCalls) == 0 {
			return s.finish(result, completion, emit), nil
		}

		assistantTurn := conversation.Message{
			Role:         conversation.RoleAssistant,
			Content:      completion.Text,
			ToolCalls:    toToolCallRecords(completion.ToolCalls),
			Model:        completion.ModelIdentifier,
			ProviderID:   completion.ProviderID,
			InputTokens:  completion.InputTokens,
			OutputTokens: completion.OutputTokens,
		}
		result.Messages = append(result.Messages, assistantTurn)
		messages = append(messages, serviceports.Message{
			Role:      serviceports.RoleAssistant,
			Content:   completion.Text,
			ToolCalls: completion.ToolCalls,
		})
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventMessage,
			Data: serviceports.AssistantMessageEvent{
				Content:   assistantTurn.Content,
				ToolCalls: assistantTurn.ToolCalls,
				Model:     assistantTurn.Model,
			},
		})

		for _, call := range completion.ToolCalls {
			emit(serviceports.StreamEvent{
				Event: serviceports.AssistantEventToolStarted,
				Data: serviceports.AssistantToolStartedEvent{
					CallID:    call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
				},
			})

			if call.Name == findToolsName {
				outcome := toolOutcome{content: s.resolveFind(tools, call.Arguments)}
				result.ToolCallsUsed++
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			outcome := s.dispatch(ctx, req, call, completion.Text)
			result.ToolCallsUsed++
			s.recordToolResult(result, &messages, call, outcome, emit)
		}

		s.logger.Debug("agent tool iteration",
			zap.String("agent", definition.Name),
			zap.Int("used", result.ToolCallsUsed),
			zap.Int("budget", budget),
		)
	}

	result.Exhausted = true
	result.Reply = exhaustedReply
	result.Messages = append(result.Messages, conversation.Message{
		Role:    conversation.RoleAssistant,
		Content: exhaustedReply,
	})
	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventDelta,
		Data:  serviceports.AssistantDeltaEvent{Text: exhaustedReply},
	})

	return result, nil
}

// recordToolResult files one tool's outcome into the run, the adapter history
// and the stream. find_tools and a dispatched tool both come through here so a
// loaded-tools answer is recorded exactly like any other tool result — it is one
// to the model, and a transcript that hid it would not explain the turn.
func (s *Service) recordToolResult(
	result *serviceports.RunResult,
	messages *[]serviceports.Message,
	call serviceports.ToolCall,
	outcome toolOutcome,
	emit serviceports.AssistantStreamEmitter,
) {
	if outcome.action != nil {
		result.Actions = append(result.Actions, *outcome.action)
	}

	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventToolFinished,
		Data: serviceports.AssistantToolFinishedEvent{
			CallID:   call.ID,
			Name:     call.Name,
			Failed:   outcome.failed,
			Proposed: outcome.action != nil && !outcome.action.Executed,
			Content:  outcome.content,
		},
	})

	result.Messages = append(result.Messages, conversation.Message{
		Role:       conversation.RoleTool,
		Content:    outcome.content,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		ToolFailed: outcome.failed,
	})
	*messages = append(*messages, serviceports.Message{
		Role:       serviceports.RoleTool,
		Content:    outcome.content,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		IsError:    outcome.failed,
	})
}

func (s *Service) finish(
	result *serviceports.RunResult,
	completion *serviceports.ChatCompletionResult,
	emit serviceports.AssistantStreamEmitter,
) *serviceports.RunResult {
	outputDecision := agentguard.EvaluateOutput(completion.Text)
	if !outputDecision.Allowed {
		s.logger.Warn("agent reply refused by output guard",
			zap.String("rule", outputDecision.MatchedRule),
			zap.String("model", completion.ModelIdentifier),
		)

		result.OutputRefused = true
		result.OutputRule = outputDecision.MatchedRule
		result.Reply = outputDecision.Message
		result.Messages = append(result.Messages, conversation.Message{
			Role:          conversation.RoleAssistant,
			Content:       outputDecision.Message,
			Refused:       true,
			ScopeStage:    string(outputDecision.Stage),
			ScopeCategory: string(outputDecision.Category),
			ScopeReason:   string(outputDecision.Reason),
			Model:         completion.ModelIdentifier,
			ProviderID:    completion.ProviderID,
			InputTokens:   completion.InputTokens,
			OutputTokens:  completion.OutputTokens,
		})
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventRefused,
			Data: serviceports.AssistantRefusedEvent{
				Message:  outputDecision.Message,
				Stage:    string(outputDecision.Stage),
				Category: string(outputDecision.Category),
				Reason:   string(outputDecision.Reason),
			},
		})

		return result
	}

	result.Reply = completion.Text
	result.Messages = append(result.Messages, conversation.Message{
		Role:         conversation.RoleAssistant,
		Content:      completion.Text,
		Model:        completion.ModelIdentifier,
		ProviderID:   completion.ProviderID,
		InputTokens:  completion.InputTokens,
		OutputTokens: completion.OutputTokens,
	})

	return result
}

func (s *Service) ToolSummaries(
	definition *agentdefinition.Definition,
) []agentdefinition.ToolSummary {
	summaries := make([]agentdefinition.ToolSummary, 0, len(definition.ToolNames))

	for _, name := range definition.ToolNames {
		if tool, ok := s.queryTools.Get(name); ok {
			summaries = append(summaries, agentdefinition.ToolSummary{
				Name:        tool.Name(),
				Description: tool.Description(),
				Query:       true,
			})
			continue
		}

		tool, ok := s.actionTools.Get(name)
		if !ok {
			continue
		}

		summaries = append(summaries, agentdefinition.ToolSummary{
			Name:        tool.Name(),
			Description: tool.Description(),
			Tier:        definition.EffectiveTier(name, tool.DefaultAutonomyTier()),
		})
	}

	return summaries
}
