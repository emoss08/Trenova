package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/shared/pulid"
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
	repeats := newRepeatGuard()

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
			PreferredProviderID: preferredProvider(req, definition),
		}, sink)
		if err != nil {
			// What ran travels with the error. The caller decides whether to
			// keep it, but it cannot keep what it was never handed: a model that
			// died after a tool wrote something used to take the record of that
			// write with it.
			return result, err
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

			// Arguments that did not parse are not arguments. The tool used to
			// run on the empty map that stood in for them, and a list tool given
			// no filters lists everything.
			if call.ArgumentsError != "" {
				outcome := failedOutcome(
					"Tool %q was not run: its arguments were not valid JSON (%s). "+
						"This usually means the reply hit its output limit partway through "+
						"the call. Send it again with complete arguments.",
					call.Name, call.ArgumentsError,
				)
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			// The budget is per call, not per batch. A model that asks for ten
			// tools in one completion does not get ten when it was allowed one;
			// the calls past the line are answered, so the model knows, but not
			// run.
			if result.ToolCallsUsed >= budget {
				outcome := failedOutcome(
					"Tool %q was not run: this turn's tool budget of %d is spent. "+
						"Answer with what you have.",
					call.Name, budget,
				)
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			if call.Name == findToolsName {
				outcome := toolOutcome{content: s.resolveFind(tools, call.Arguments)}
				result.ToolCallsUsed++
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			if call.Name == askUserName {
				outcome := toolOutcome{content: resolveAsk(call.Arguments)}
				result.ToolCallsUsed++
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			if previous, repeated := repeats.seen(call); repeated {
				outcome := failedOutcome("%s", repeatRefusal(call.Name, previous))
				result.ToolCallsUsed++
				s.recordToolResult(result, &messages, call, outcome, emit)
				continue
			}

			outcome := s.dispatch(ctx, req, call, completion.Text)
			if outcome.failed {
				repeats.record(call, outcome.content)
			}
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

	reply := completion.Text
	if completion.Truncated {
		// Said in the reply rather than left to an error banner, because this
		// is saved and read back later: somebody opening the thread tomorrow
		// has to be able to tell a finished answer from one that stopped in the
		// middle of a sentence.
		reply += truncationNotice
		result.Truncated = true
	}

	result.Reply = reply
	result.Messages = append(result.Messages, conversation.Message{
		Role:         conversation.RoleAssistant,
		Content:      reply,
		Model:        completion.ModelIdentifier,
		ProviderID:   completion.ProviderID,
		InputTokens:  completion.InputTokens,
		OutputTokens: completion.OutputTokens,
	})

	return result
}

// truncationNotice marks a reply the provider stopped partway through.
//
// A model that dies mid-sentence used to take its whole answer with it: the
// reader watched a correct list of drivers appear and then be replaced by "the
// assistant could not finish this reply". Keeping the text is most of the fix;
// saying where it stopped is the rest, because an answer that ends mid-clause
// is one somebody could otherwise act on as though it were complete.
const truncationNotice = "\n\n_This reply was cut off before it finished. " +
	"Ask again to get the rest._"

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

// preferredProvider resolves whose choice of model wins.
//
// The reader's beats the administrator's default: the definition pins a
// provider for everyone using that agent, while a person picking in the
// composer is choosing for their own conversation. Neither can reach a provider
// the organization has not enabled for this task — the router filters its
// candidates before any preference is applied — so this decides ordering, never
// access.
func preferredProvider(
	req *serviceports.RunRequest,
	definition *agentdefinition.Definition,
) pulid.ID {
	if !req.PreferredProviderID.IsNil() {
		return req.PreferredProviderID
	}

	return definition.PreferredProviderID
}
