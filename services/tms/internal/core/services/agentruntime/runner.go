package agentruntime

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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
	Budgets serviceports.AgentBudgetService       `optional:"true"`
	Trust   repositories.AgentToolTrustRepository `optional:"true"`
	// Extensions is optional. Without it no extension's tools are offered.
	Extensions serviceports.AgentExtensionGate `optional:"true"`
	// Memories is optional. With it every memory a prompt carries is
	// counted as used; without it the prompt still carries them.
	Memories       serviceports.AgentMemoryService `optional:"true"`
	Vectorizer     serviceports.QueryVectorizer    `optional:"true"`
	CatalogVectors serviceports.CatalogVectorIndex `optional:"true"`
	// Previews is optional. With it a held write's target is pinned, and
	// what it would do kept as its baseline, in one read-only snapshot;
	// without it the target is pinned alone.
	Previews serviceports.ProposalPreviewService `optional:"true"`
	// Rules is optional. With it a tool an organization holds lower than its
	// declared rule is decided and tainted by the organization's rule.
	Rules serviceports.ToolRuleOverrides `optional:"true"`
	// Changes is optional. With it a turn is told when a record it is
	// working with changes elsewhere while it works.
	Changes serviceports.RecordChangeFeed `optional:"true"`
	// Metrics is optional. With it every tool call is counted by tool and by
	// what became of it; without it nothing is counted.
	Metrics *metrics.Registry `optional:"true"`
	// Definitions is optional. With it a search for a tool the agent does not
	// hold names the person's other agents that hold it.
	Definitions repositories.AgentDefinitionRepository `optional:"true"`
}

type Service struct {
	logger      *zap.Logger
	completion  serviceports.CompletionService
	queryTools  serviceports.AgentQueryToolRegistry
	actionTools serviceports.AgentToolRegistry
	permissions serviceports.PermissionEngine
	definitions repositories.AgentDefinitionRepository
	catalog     *agenttoolcatalog.Catalog
	versions    serviceports.RecordVersionReader
	budgets     serviceports.AgentBudgetService
	trust       repositories.AgentToolTrustRepository
	extensions  serviceports.AgentExtensionGate
	memories    serviceports.AgentMemoryService
	vectorizer  serviceports.QueryVectorizer
	vectors     serviceports.CatalogVectorIndex
	previews    serviceports.ProposalPreviewService
	rules       serviceports.ToolRuleOverrides
	changes     serviceports.RecordChangeFeed
	metrics     *metrics.Assistant
	// arguments holds each tool's compiled schema, against which every
	// call's arguments are checked before the tool sees them.
	arguments *toolschema.Validator
}

func New(p Params) *Service {
	return &Service{
		arguments:   toolschema.NewValidator(),
		logger:      p.Logger.Named("service.agentruntime"),
		completion:  p.Completion,
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
		permissions: p.Permissions,
		definitions: p.Definitions,
		catalog:     p.Catalog,
		versions:    p.Versions,
		budgets:     p.Budgets,
		trust:       p.Trust,
		extensions:  p.Extensions,
		memories:    p.Memories,
		vectorizer:  p.Vectorizer,
		vectors:     p.CatalogVectors,
		previews:    p.Previews,
		rules:       p.Rules,
		changes:     p.Changes,
		metrics:     metrics.AssistantFrom(p.Metrics),
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

	t.announceOpened(fx)
	t.announceMemories(fx)
	if t.directed != nil {
		return s.driveDirected(t, fx), nil
	}

	retries := 0
	asked := false
	stuck := false
	grounding := groundingState{}
	for result.ToolCallsUsed < budget && !stuck {
		s.interject(t, fx)
		reply, err := fx.Complete(t, t.completionRequest())
		result.Usage = result.Usage.Add(reply.usage())
		completion := reply.Completion
		if err == nil && reply.Looped {
			s.logger.Warn("agent reply fell into a loop; discarding it",
				zap.String("agent", definition.Name),
				zap.String("model", completion.ModelIdentifier),
				zap.Bool("in_reasoning", reply.InReasoning),
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

		retry, stop := s.handleCutOff(t, fx, completion)
		if retry {
			continue
		}

		result.Model = completion.ModelIdentifier
		result.ProviderID = completion.ProviderID
		result.ContextWindow = completion.ContextWindow
		tagReasoning(completion)
		if stop {
			return s.finishCutOff(t, fx, completion), nil
		}
		tagToolCalls(completion)
		distinctCallIDs(completion, t.callIDs, callIDMinter{
			mint: fx.NewCallID,
			renewSynthesized: func() bool {
				return fx.Supports(changeFreshSynthesizedCallIDs)
			},
		})

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
			if s.reground(t, fx, completion, &grounding) {
				continue
			}
			s.passReply(t, fx, completion)

			return s.finish(result, completion, fx), nil
		}

		liftRationales(completion.ToolCalls)
		assistantTurn := conversation.Message{
			Role:         conversation.RoleAssistant,
			Content:      completion.Text,
			ToolCalls:    toToolCallRecords(completion.ToolCalls),
			Reasoning:    completion.Reasoning,
			Model:        completion.ModelIdentifier,
			ProviderID:   completion.ProviderID,
			FallbackFrom: fallbackOf(completion),
			InputTokens:  completion.InputTokens,
			OutputTokens: completion.OutputTokens,
			LatencyMs:    completion.LatencyMs,
			CostUSD:      completion.CostUSD,
			CreatedAt:    fx.Now(),
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
				Content:      assistantTurn.Content,
				ToolCalls:    s.callsWithEffects(assistantTurn.ToolCalls),
				Model:        assistantTurn.Model,
				ProviderID:   assistantTurn.ProviderID,
				FallbackFrom: assistantTurn.FallbackFrom,
			},
		})

		for _, call := range completion.ToolCalls {
			fx.Emit(serviceports.StreamEvent{
				Event: serviceports.AssistantEventToolStarted,
				Data: serviceports.AssistantToolStartedEvent{
					CallID:    call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
					Effect:    s.ToolEffect(call.Name),
					Why:       call.Why,
				},
			})

			// Arguments that did not parse are not arguments. The tool used to
			// run on the empty map that stood in for them, and a list tool given
			// no filters lists everything.
			if call.ArgumentsError != "" {
				outcome := refusedOutcome(
					aitrace.OutcomeInvalid,
					"its arguments were not valid JSON",
					"Tool %q was not run: its arguments were not valid JSON (%s). "+
						"This usually means the reply hit its output limit partway through "+
						"the call. Send it again with complete arguments.",
					call.Name, call.ArgumentsError,
				)
				s.recordToolResult(t, fx, call, outcome)
				// It counts against the budget like any call. A model whose output
				// limit cuts every call short sends the same broken call each time,
				// and a loop that did not count it asked the model again, and paid for
				// it, without end.
				result.ToolCallsUsed++
				continue
			}

			// The budget is per call, not per batch. A model that asks for ten
			// tools in one completion does not get ten when it was allowed one;
			// the calls past the line are answered, so the model knows, but not
			// run.
			if result.ToolCallsUsed >= budget {
				outcome := refusedOutcome(
					aitrace.OutcomeOverBudget,
					"the turn's tool budget is spent",
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
					outcome = refusedOutcome(
						aitrace.OutcomeOverBudget,
						"the turn's tool searches are spent",
						"You have searched for tools %d times this turn. Use what is "+
							"loaded, or tell the person what you could not find.",
						maxFindCalls,
					)
				} else {
					answer := fx.Find(t, call.Arguments)
					outcome = toolOutcome{
						content: answer.Content + searchesLeft(tools.findCalls),
						found:   answer.Found,
						handOff: answer.HandOff,
					}
				}
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if call.Name == askUserName {
				question := comparableQuestion(stringArg(call.Arguments, "question"))
				_, repeated := t.questions[question]
				outcome := toolOutcome{content: resolveAsk(t.localNow(fx), call.Arguments)}
				switch {
				case !tools.offers(askUserName) && t.req.Delegation != nil:
					outcome = refusedOutcome(aitrace.OutcomeDenied,
						"a delegated task cannot ask the person", "%s", delegatedAskRefusal)
				case !tools.offers(askUserName):
					outcome = refusedOutcome(aitrace.OutcomeDenied,
						"nobody is in the conversation to ask", "%s", unattendedAskRefusal)
				case question != "" && repeated:
					outcome = refusedOutcome(aitrace.OutcomeDuplicate,
						"the same question was already asked", "%s", repeatedAskRefusal)
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

			if call.Name == requestDecisionName {
				outcome := refusedOutcome(aitrace.OutcomeDenied,
					"the turn is not offered the approval box", "%s", unofferedDecisionRefusal)
				if tools.offers(requestDecisionName) {
					outcome = t.requestDecision(call.Arguments)
				}
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if call.Name == withdrawProposalName {
				outcome := refusedOutcome(aitrace.OutcomeDenied,
					"the turn has filed no proposal to withdraw", "%s", unofferedWithdrawal)
				if t.req.Delegation != nil {
					outcome = refusedOutcome(aitrace.OutcomeDenied,
						"a delegated task cannot withdraw a proposal", "%s", delegatedWithdrawal)
				}
				if tools.offers(withdrawProposalName) {
					outcome = t.withdrawProposal(call.Arguments)
				}
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if call.Name == publishArtifactName {
				outcome := publishOutcome(call.Arguments)
				if !tools.offers(publishArtifactName) {
					outcome = refusedOutcome(aitrace.OutcomeDenied,
						"the turn is not offered documents", "%s", unpublishableRefusal)
				}
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			// Only a turn that holds delegate_task hands out work, and a turn
			// working for another agent is told why it cannot. Any other turn
			// that names it is answered as for any tool it does not hold, as
			// it was before the tool existed.
			if call.Name == delegateTaskName &&
				(t.holds(delegateTaskName) || t.req.Delegation != nil) {
				outcome := s.delegate(t, fx, call)
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			if !t.holds(call.Name) {
				outcome := s.unheldOutcome(tools, call.Name)
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}
			if stuck {
				s.recordToolResult(t, fx, call, refusedOutcome(
					aitrace.OutcomeDuplicate,
					"the turn stopped calling tools after repeated failed calls",
					"%s",
					stoppedCallText(call.Name),
				))
				continue
			}

			// A tool the agent holds but was not sent runs anyway — the
			// configuration is the grant, disclosure only decides what was
			// shown — and is loaded, so the next request carries its schema.
			s.load(tools, call.Name)

			if previous, repeated := t.repeats.seen(call); repeated {
				outcome := refusedOutcome(
					aitrace.OutcomeDuplicate,
					"the same call already failed this turn",
					"%s",
					repeatRefusal(call.Name, previous),
				)
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				if t.repeats.refused() && fx.Supports(changeStopOnRepeats) {
					stuck = true
				}
				continue
			}

			if t.repeats.readBefore(call) {
				outcome := toolOutcome{content: repeatedRead(call.Name)}
				result.ToolCallsUsed++
				s.recordToolResult(t, fx, call, outcome)
				continue
			}

			earlier := t.earlier(call.Name)
			outcome := fx.Dispatch(t, DispatchCall{
				Call:                 call,
				CompletionText:       completion.Text,
				ProposedSoFar:        unwithdrawn(result.Actions),
				Ordinal:              t.counts.next(call),
				AfterExternalContent: t.external,
				Taint:                result.Taint,
				Earlier:              earlier,
			}).internal()
			outcome.dispatched = true
			if !outcome.failed {
				outcome.content += fanOutNote(call.Name, len(earlier)+1)
			}
			if outcome.failed {
				t.repeats.record(call, outcome.content)
			} else {
				t.repeats.ran(call)
			}
			if !outcome.failed && ReadsExternalContent(call.Name) {
				t.external = true
			}
			t.absorbTaint(fx, outcome.taint)
			t.noteMemories(fx, &outcome)
			result.ToolCallsUsed++
			s.recordToolResult(t, fx, call, outcome)
			if !outcome.failed {
				t.noteToolMemories(fx, call.Name)
				t.noteShown(&call)
				t.noteWorld(fx, &call, &outcome)
			}
		}

		s.logger.Debug("agent tool iteration",
			zap.String("agent", definition.Name),
			zap.Int("used", result.ToolCallsUsed),
			zap.Int("budget", budget),
		)
	}

	note, ending := budgetSpentNote, exhaustedReply
	if stuck {
		note, ending = stuckNote, stuckReply
	}
	if fx.Supports(changeFinalAnswer) {
		if final := s.finalAnswer(t, fx, result, note); final != nil {
			grounding.rewritten = true
			s.reground(t, fx, final, &grounding)
			s.passReply(t, fx, final)

			return s.finish(result, final, fx), nil
		}
	}

	result.Exhausted = true
	result.Reply = ending
	result.Messages = append(result.Messages, conversation.Message{
		Role:      conversation.RoleAssistant,
		Content:   ending,
		CreatedAt: fx.Now(),
	})
	fx.Emit(deltaEvent(ending))

	return result, nil
}

// maxLoopRefusalChars bounds what a loop refusal keeps of what the model
// was told: the reason is the figure, the text only what explains it.
const maxLoopRefusalChars = 1000

// changeFinalAnswer is the loop asking for an answer once its tool budget is
// spent, rather than ending on a canned line.
const changeFinalAnswer = "agent-loop-final-answer"

// budgetSpentNote is what the model is told when its tool budget is spent.
const budgetSpentNote = "You have used every tool call this turn allows. Answer the person " +
	"now from what the tools already returned. Say plainly what you could not finish or " +
	"check, and do not ask for another tool."

// finalAnswer asks the model once more with no tools, so a turn that spent its
// budget answers from what it gathered. Ending on "I could not finish" threw
// away every lookup the turn had paid for. Nothing is returned when that ask
// fails, loops, comes back empty or asks for a tool anyway: the canned line is
// still better than any of those.
func (s *Service) finalAnswer(
	t *Turn,
	fx TurnEffects,
	result *serviceports.RunResult,
	note string,
) *serviceports.ChatCompletionResult {
	req := t.completionRequest()
	req.Tools = nil
	req.Messages = append(slices.Clone(req.Messages), serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: note,
	})

	reply, err := fx.Complete(t, req)
	result.Usage = result.Usage.Add(reply.usage())
	completion := reply.Completion
	if err != nil || reply.Looped || completion == nil ||
		len(completion.ToolCalls) > 0 || strings.TrimSpace(completion.Text) == "" {
		return nil
	}

	result.Model = completion.ModelIdentifier
	result.ProviderID = completion.ProviderID
	result.ContextWindow = completion.ContextWindow
	tagReasoning(completion)

	return completion
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
	if outcome.failed && !outcome.dispatched && outcome.verdict != "" {
		result.LoopRefusals = append(result.LoopRefusals, serviceports.LoopRefusal{
			CallID:   call.ID,
			ToolName: call.Name,
			Verdict:  outcome.verdict,
			Reason:   outcome.reason,
			Content:  stringutils.Ellipsize(outcome.content, maxLoopRefusalChars),
		})
	}
	if outcome.action != nil {
		result.Actions = append(result.Actions, *outcome.action)
		t.offerWithdrawal(outcome.action)
	}
	outcome = fx.Observe(t, &call, outcome.exported()).internal()
	s.countToolOutcome(fx, call.Name, &outcome)
	t.keepTable(&outcome)
	effect := s.ToolEffect(call.Name)
	summary := summarizeOutcome(call, outcome)

	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventToolFinished,
		Data: serviceports.AssistantToolFinishedEvent{
			CallID:        call.ID,
			Name:          call.Name,
			Failed:        outcome.failed,
			Proposed:      outcome.action != nil && !outcome.action.Executed,
			Content:       outcome.content,
			Effect:        effect,
			Summary:       summary,
			Verdict:       outcome.verdict,
			HandOffAgents: outcome.handOff,
		},
	})

	result.Messages = append(result.Messages, conversation.Message{
		Role:           conversation.RoleTool,
		Content:        outcome.content,
		ToolCallID:     call.ID,
		ToolName:       call.Name,
		ToolFailed:     outcome.failed,
		ToolVerdict:    outcome.verdict,
		ToolEffect:     effect,
		ToolSummary:    summary,
		FoundTools:     outcome.found,
		HandOffAgents:  outcome.handOff,
		DelegateReport: outcome.delegateReport,
		CreatedAt:      fx.Now(),
	})
	t.messages = append(t.messages, serviceports.Message{
		Role:       serviceports.RoleTool,
		Content:    outcome.content,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		IsError:    outcome.failed,
	})
}

// NewCallID mints a tool call id no provider will have used.
func NewCallID() string { return "call_" + pulid.MustNew("tc_").String() }

// observe hands a finished call to the observer and folds what it showed the
// person back into the result the model reads.
func (s *Service) observe(
	observe serviceports.ToolObserver,
	call serviceports.ToolCall,
	outcome toolOutcome,
	earlier []string,
) toolOutcome {
	if request, requested := outcome.data.(serviceports.DecisionRequest); requested &&
		outcome.publishes {
		return decisionRequested(observe, &call, request)
	}
	if observe == nil {
		if outcome.publishes {
			return failedOutcome("%s", unpublishableRefusal)
		}
		return outcome
	}

	shown, err := observe(serviceports.ToolObservation{
		Call:    call,
		Data:    outcome.data,
		Failed:  outcome.failed,
		Action:  outcome.action,
		Earlier: earlier,
	})

	if shown != nil {
		outcome.shown = shown
	}

	switch {
	case outcome.publishes && err != nil:
		return failedOutcome("Tool %q could not keep the document: %s. Put the text in your "+
			"reply instead.", publishArtifactName, err.Error())
	case outcome.publishes && shown == nil:
		return failedOutcome("Tool %q could not keep the document. Put the text in your "+
			"reply instead.", publishArtifactName)
	case outcome.publishes:
		outcome.content = publishedContent(shown)
		outcome.summary = summaryLine(shown.Title)
	case shown != nil && !outcome.failed:
		outcome.content += shownNote(shown)
		if outcome.summary == "" {
			outcome.summary = summaryLine(shown.Title)
		}
	}

	return outcome
}

func (s *Service) finish(
	result *serviceports.RunResult,
	completion *serviceports.ChatCompletionResult,
	fx TurnEffects,
) *serviceports.RunResult {
	guarded, outputDecision := agentguard.EvaluateOutput(completion.Text)
	if outputDecision.Altered {
		s.logger.Warn("agent reply altered by output guard",
			zap.String("rule", outputDecision.MatchedRule),
			zap.String("model", completion.ModelIdentifier),
		)
		result.OutputAltered = true
		result.OutputRule = outputDecision.MatchedRule
		fx.Emit(replyReplacedEvent(guarded, outputAlteredReason))
		completion.Text = guarded
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
	message := conversation.Message{
		Role:         conversation.RoleAssistant,
		Content:      reply,
		Reasoning:    completion.Reasoning,
		Model:        completion.ModelIdentifier,
		ProviderID:   completion.ProviderID,
		Truncated:    completion.Truncated,
		FallbackFrom: fallbackOf(completion),
		InputTokens:  completion.InputTokens,
		OutputTokens: completion.OutputTokens,
		LatencyMs:    completion.LatencyMs,
		CostUSD:      completion.CostUSD,
		CreatedAt:    fx.Now(),
	}
	if outputDecision.Altered {
		message.ScopeStage = string(outputDecision.Stage)
		message.ScopeCategory = string(outputDecision.Category)
		message.ScopeReason = string(outputDecision.Reason)
	}
	result.Messages = append(result.Messages, message)

	return result
}

// outputAlteredReason is why a streamed reply is replaced by the recorded one
// when the output guard took code out of it.
const outputAlteredReason = "The reply held a code block, which was left out."

// fallbackOf is the provider asked first, when another one gave the reply.
func fallbackOf(completion *serviceports.ChatCompletionResult) *conversation.ProviderFallback {
	if completion == nil || completion.FallbackFrom == nil {
		return nil
	}
	from := completion.FallbackFrom

	return &conversation.ProviderFallback{
		ProviderID: from.ProviderID,
		Name:       from.Name,
		Model:      from.Model,
		Status:     from.Status,
	}
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
		canned.ContextWindow = attempt.ContextWindow
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
	return s.summarize(definition, s.heldTools(definition))
}

func (s *Service) summarize(
	definition *agentdefinition.Definition,
	names []string,
) []agentdefinition.ToolSummary {
	summaries := make([]agentdefinition.ToolSummary, 0, len(names))

	for _, name := range names {
		if tool, ok := s.queryTools.Get(name); ok {
			summaries = append(summaries, agentdefinition.ToolSummary{
				Name:        tool.Name(),
				Description: tool.Description(),
				Query:       true,
				BatchOf:     batchOf(tool),
				Recipe:      recipeOf(tool),
			})
			continue
		}

		tool, ok := s.actionTools.Get(name)
		if !ok {
			continue
		}

		policy := tool.Policy()
		summaries = append(summaries, agentdefinition.ToolSummary{
			Name:                tool.Name(),
			Description:         tool.Description(),
			Tier:                agenttoolpolicy.StaticTier(definition, policy),
			PersonalRunsUnasked: policy.PersonalRunsUnasked && !tierSetFor(definition, name),
			AlwaysProposes:      agenttoolpolicy.AlwaysProposes(definition, policy),
			BatchOf:             batchOf(tool),
			Recipe:              recipeOf(tool),
		})
	}

	return summaries
}

func batchOf(tool any) string {
	if batch, ok := tool.(serviceports.BatchTool); ok {
		return batch.BatchOf()
	}

	return ""
}

// tierSetFor reports a tool whose tier a person set on the agent, which
// takes away the personal exemption a tool like remember otherwise has.
func tierSetFor(definition *agentdefinition.Definition, name string) bool {
	_, set := definition.ToolTiers[name]

	return set
}

func recipeOf(tool any) []string {
	if recipe, ok := tool.(serviceports.RecipeTool); ok {
		return recipe.Recipe()
	}

	return nil
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

// usedCallIDs is every tool call id the conversation already holds.
func usedCallIDs(history []conversation.Message) map[string]struct{} {
	used := make(map[string]struct{})
	for idx := range history {
		for _, call := range history[idx].ToolCalls {
			if call.ID != "" {
				used[call.ID] = struct{}{}
			}
		}
	}

	return used
}

// changeFreshSynthesizedCallIDs is the loop giving every call whose id the
// adapter made up an id of its own, not only one the conversation already
// holds.
const changeFreshSynthesizedCallIDs = "agent-loop-fresh-synthesized-call-ids"

// callIDMinter is how distinctCallIDs comes by a new id. mint makes one: the
// turn's effects supply it, because a fresh id is random and workflow code may
// not be. renewSynthesized reports whether an id the adapter made up is
// replaced even when nothing else in the conversation holds it; it is asked
// only when a completion carries one, so a turn that never meets one records
// no change marker.
type callIDMinter struct {
	mint             func() string
	renewSynthesized func() bool
}

// distinctCallIDs gives every call in a completion an id no other call in the
// conversation has.
//
// Providers that return no id get one synthesized from the call's position,
// call_0 in every completion, so the same id named a different call on every
// turn. The artifacts a call produces are keyed on it, the transcript pairs
// results with calls by it, and a provider handed a history with two calls
// under one id pairs the wrong result with the wrong call. Checking the id
// against the conversation is not enough: the conversation the turn replays
// is only its most recent messages, and an artifact keyed on a call_0 from
// before them was overwritten by the next call_0. So a synthesized id is
// always replaced. A provider's own unique ids are kept as they are.
func distinctCallIDs(
	completion *serviceports.ChatCompletionResult,
	used map[string]struct{},
	minter callIDMinter,
) {
	decided, renew := false, false
	for idx := range completion.ToolCalls {
		call := &completion.ToolCalls[idx]
		if call.SynthesizedID && !decided {
			decided, renew = true, minter.renewSynthesized()
		}
		_, taken := used[call.ID]
		if call.ID == "" || taken || (call.SynthesizedID && renew) {
			call.ID = minter.mint()
			call.SynthesizedID = false
		}
		used[call.ID] = struct{}{}
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
