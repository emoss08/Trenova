package agentruntime

import (
	"context"
	"slices"
	"strings"

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
	// Versions is optional. Without it proposals are still made, just without
	// the record pinned; with it the executor can refuse a change to a record
	// that moved on since the proposal.
	Versions serviceports.RecordVersionReader `optional:"true"`
	// Budgets is optional. With it an automatic write past its tool's daily
	// cap is refused before it runs.
	Budgets serviceports.AgentBudgetService `optional:"true"`
}

type Service struct {
	logger      *zap.Logger
	completion  serviceports.CompletionService
	queryTools  serviceports.AgentQueryToolRegistry
	actionTools serviceports.AgentToolRegistry
	permissions serviceports.PermissionEngine
	catalog     *agenttoolcatalog.Catalog
	versions    serviceports.RecordVersionReader
	budgets     serviceports.AgentBudgetService
}

func New(p Params) serviceports.AgentRuntime {
	return &Service{
		logger:      p.Logger.Named("service.agentruntime"),
		completion:  p.Completion,
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
		permissions: p.Permissions,
		catalog:     p.Catalog,
		versions:    p.Versions,
		budgets:     p.Budgets,
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

	return s.Drive(s.OpenTurn(ctx, req), &localEffects{
		s:        s,
		ctx:      ctx,
		emit:     emit,
		observer: req.ToolObserver,
	})
}

// Drive runs a turn to its end: the model is asked, the tools it calls are run,
// and it is asked again, until it answers or runs out of tool calls.
//
// Nothing here reaches outside the turn except through fx, which is what lets
// the same loop run in process and as durable workflow code.
func (s *Service) Drive(t *Turn, fx TurnEffects) (*serviceports.RunResult, error) {
	definition := t.req.Definition
	budget := t.budget
	result := t.result
	tools := t.tools

	retries := 0
	asked := false
	for result.ToolCallsUsed < budget {
		reply, err := fx.Complete(t, t.completionRequest())
		completion := reply.Completion
		if err == nil && reply.Looped {
			s.logger.Warn("agent reply fell into a loop; discarding it",
				zap.String("agent", definition.Name),
				zap.String("model", completion.ModelIdentifier),
				zap.Int("retry", retries+1),
			)
			if retries < maxReplyRetries {
				retries++
				fx.Emit(serviceports.StreamEvent{
					Event: serviceports.AssistantEventRetrying,
					Data: serviceports.AssistantRetryingEvent{
						Attempt: retries,
						Reason:  loopRestartReason,
						Kind:    serviceports.RetryKindRestart,
					},
				})
				continue
			}
			fx.Emit(serviceports.StreamEvent{
				Event: serviceports.AssistantEventRetrying,
				Data: serviceports.AssistantRetryingEvent{
					Attempt: retries + 1,
					Reason:  loopRestartReason,
					Kind:    serviceports.RetryKindRestart,
				},
			})
			fx.Emit(deltaEvent(loopedReply))
			return s.finish(result, cannedCompletion(completion, loopedReply), fx), nil
		}
		if err != nil {
			// What ran travels with the error. The caller decides whether to
			// keep it, but it cannot keep what it was never handed: a model that
			// died after a tool wrote something used to take the record of that
			// write with it.
			return result, err
		}

		result.Model = completion.ModelIdentifier
		result.ProviderID = completion.ProviderID
		tagReasoning(completion)
		tagToolCalls(completion)

		if len(completion.ToolCalls) == 0 {
			// A turn that ends in silence reads as a hung screen. One that
			// asked the person a question has said what it needed to; any
			// other is asked once more, then given a plain line.
			if strings.TrimSpace(completion.Text) == "" && !asked {
				if retries < maxReplyRetries {
					retries++
					continue
				}
				fx.Emit(deltaEvent(emptyReply))
				return s.finish(result, cannedCompletion(completion, emptyReply), fx), nil
			}

			return s.finish(result, completion, fx), nil
		}

		assistantTurn := conversation.Message{
			Role:         conversation.RoleAssistant,
			Content:      completion.Text,
			ToolCalls:    toToolCallRecords(completion.ToolCalls),
			Reasoning:    completion.Reasoning,
			Model:        completion.ModelIdentifier,
			ProviderID:   completion.ProviderID,
			InputTokens:  completion.InputTokens,
			OutputTokens: completion.OutputTokens,
			LatencyMs:    completion.LatencyMs,
			CostUSD:      completion.CostUSD,
		}
		result.Messages = append(result.Messages, assistantTurn)
		// The thinking goes back with the calls it produced. Anthropic and the
		// Responses API both refuse a tool result whose reasoning is missing.
		t.messages = append(t.messages, serviceports.Message{
			Role:      serviceports.RoleAssistant,
			Content:   completion.Text,
			ToolCalls: completion.ToolCalls,
			Reasoning: completion.Reasoning,
		})
		fx.Emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventMessage,
			Data: serviceports.AssistantMessageEvent{
				Content:   assistantTurn.Content,
				ToolCalls: assistantTurn.ToolCalls,
				Model:     assistantTurn.Model,
			},
		})

		for _, call := range completion.ToolCalls {
			fx.Emit(serviceports.StreamEvent{
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
				s.recordToolResult(t, fx, call, outcome)
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
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if call.Name == findToolsName {
				tools.findCalls++
				var outcome toolOutcome
				if tools.findCalls > maxFindCalls {
					// Past the cap the search is charged, so a model that
					// only ever searches still runs out of turn.
					result.ToolCallsUsed++
					outcome = failedOutcome(
						"You have searched for tools %d times this turn. Use what is "+
							"loaded, or tell the person what you could not find.",
						maxFindCalls,
					)
				} else {
					outcome = toolOutcome{content: fx.Find(t, call.Arguments)}
				}
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if call.Name == askUserName {
				question := comparableQuestion(stringArg(call.Arguments, "question"))
				_, repeated := t.questions[question]
				outcome := toolOutcome{content: resolveAsk(call.Arguments)}
				switch {
				case !tools.offers(askUserName):
					outcome = failedOutcome("%s", unattendedAskRefusal)
				case question != "" && repeated:
					outcome = failedOutcome("%s", repeatedAskRefusal)
				default:
					asked = true
					if question != "" {
						t.questions[question] = struct{}{}
					}
				}
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if !s.holds(definition, call.Name) {
				outcome := failedOutcome("%s", s.unheldRefusal(tools, call.Name))
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}
			// A tool the agent holds but was not sent runs anyway — the
			// configuration is the grant, disclosure only decides what was
			// shown — and is loaded, so the next request carries its schema.
			s.load(tools, call.Name)

			if previous, repeated := t.repeats.seen(call); repeated {
				outcome := failedOutcome("%s", repeatRefusal(call.Name, previous))
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			outcome := fx.Dispatch(t, DispatchCall{
				Call:           call,
				CompletionText: completion.Text,
				ProposedSoFar:  result.Actions,
				Ordinal:        t.counts.next(call),
			}).internal()
			if outcome.failed {
				t.repeats.record(call, outcome.content)
			}
			result.ToolCallsUsed++
			s.recordToolResult(t, fx, call, outcome)
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
	fx.Emit(deltaEvent(exhaustedReply))

	return result, nil
}

// recordToolResult files one tool's outcome into the run, the adapter history
// and the stream. find_tools and a dispatched tool both come through here so a
// loaded-tools answer is recorded exactly like any other tool result — it is one
// to the model, and a transcript that hid it would not explain the turn.
func (s *Service) recordToolResult(
	t *Turn,
	fx TurnEffects,
	call serviceports.ToolCall,
	outcome toolOutcome,
) {
	result := t.result
	if outcome.action != nil {
		result.Actions = append(result.Actions, *outcome.action)
	}
	fx.Observe(serviceports.ToolObservation{
		Call:   call,
		Data:   outcome.data,
		Failed: outcome.failed,
		Action: outcome.action,
	})

	fx.Emit(serviceports.StreamEvent{
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
	t.messages = append(t.messages, serviceports.Message{
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
	fx TurnEffects,
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
		fx.Emit(serviceports.StreamEvent{
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
		Reasoning:    completion.Reasoning,
		Model:        completion.ModelIdentifier,
		ProviderID:   completion.ProviderID,
		InputTokens:  completion.InputTokens,
		OutputTokens: completion.OutputTokens,
		LatencyMs:    completion.LatencyMs,
		CostUSD:      completion.CostUSD,
	})

	return result
}

// cannedCompletion stands a fixed line in for a reply the model could not
// give, keeping the provider and usage the attempt was charged under.
func cannedCompletion(
	attempt *serviceports.ChatCompletionResult,
	text string,
) *serviceports.ChatCompletionResult {
	canned := &serviceports.ChatCompletionResult{Text: text}
	if attempt != nil {
		canned.ModelIdentifier = attempt.ModelIdentifier
		canned.ProviderID = attempt.ProviderID
		canned.ProviderKind = attempt.ProviderKind
		canned.InputTokens = attempt.InputTokens
		canned.OutputTokens = attempt.OutputTokens
		canned.LatencyMs = attempt.LatencyMs
		canned.CostUSD = attempt.CostUSD
	}

	return canned
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
	names := s.heldTools(definition)
	summaries := make([]agentdefinition.ToolSummary, 0, len(names))

	for _, name := range names {
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

// tagReasoning records which protocol produced a trace, so an adapter of
// another kind knows not to send it back as its own.
// tagToolCalls names the provider on each call it made, so what the
// provider attached to the call is replayed to it and to nobody else.
func tagToolCalls(completion *serviceports.ChatCompletionResult) {
	for idx := range completion.ToolCalls {
		completion.ToolCalls[idx].ProviderID = completion.ProviderID
	}
}

func tagReasoning(completion *serviceports.ChatCompletionResult) {
	if completion.Reasoning != nil && completion.Reasoning.ProviderKind == "" {
		completion.Reasoning.ProviderKind = string(completion.ProviderKind)
	}
}

// usableSummaries keeps the summaries of the tools the turn may call and
// marks which are loaded, so the prompt and the request agree.
func usableSummaries(
	summaries []agentdefinition.ToolSummary,
	tools *toolSet,
) []agentdefinition.ToolSummary {
	kept := make([]agentdefinition.ToolSummary, 0, len(summaries))
	for _, summary := range summaries {
		if !slices.Contains(tools.allowed, summary.Name) {
			continue
		}
		_, loaded := tools.loaded[summary.Name]
		summary.Loaded = loaded
		kept = append(kept, summary)
	}

	return kept
}
