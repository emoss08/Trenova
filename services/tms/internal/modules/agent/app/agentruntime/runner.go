package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	Completion  agentports.CompletionService
	QueryTools  agentports.AgentQueryToolRegistry
	ActionTools agentports.AgentToolRegistry
	Permissions permissionports.PermissionEngine
	Catalog     *agenttoolcatalog.Catalog
	// Versions is optional. Without it proposals are still made, just without
	// the record pinned; with it the executor can refuse a change to a record
	// that moved on since the proposal.
	Versions agentports.RecordVersionReader `optional:"true"`
	// Budgets is optional. With it an automatic write past its tool's daily
	// cap is refused before it runs.
	Budgets agentports.AgentBudgetService       `optional:"true"`
	Trust   agentports.AgentToolTrustRepository `optional:"true"`
	// Extensions is optional. Without it no extension's tools are offered.
	Extensions agentextensionports.AgentExtensionGate `optional:"true"`
	// Memories is optional. With it every memory a prompt carries is
	// counted as used; without it the prompt still carries them.
	Memories       agentports.AgentMemoryService `optional:"true"`
	Vectorizer     agentports.QueryVectorizer    `optional:"true"`
	CatalogVectors agentports.CatalogVectorIndex `optional:"true"`
	// Previews is optional. With it a held write's target is pinned, and
	// what it would do kept as its baseline, in one read-only snapshot;
	// without it the target is pinned alone.
	Previews agentports.ProposalPreviewService `optional:"true"`
	// Rules is optional. With it a tool an organization holds lower than its
	// declared rule is decided and tainted by the organization's rule.
	Rules agentports.ToolRuleOverrides `optional:"true"`
	// Changes is optional. With it a turn is told when a record it is
	// working with changes elsewhere while it works.
	Changes agentports.RecordChangeFeed `optional:"true"`
	// Metrics is optional. With it every tool call is counted by tool and by
	// what became of it; without it nothing is counted.
	Metrics *metrics.Registry `optional:"true"`
	// Definitions is optional. With it a search for a tool the agent does not
	// hold names the person's other agents that hold it.
	Definitions agentdefinitionports.AgentDefinitionRepository `optional:"true"`
	RateLimits  kernelports.RateLimitStore                     `optional:"true"`
	Config      *config.Config                                 `optional:"true"`
	Disclosures agentports.ToolDisclosureObserver              `optional:"true"`
}

type Service struct {
	logger      *zap.Logger
	completion  agentports.CompletionService
	queryTools  agentports.AgentQueryToolRegistry
	actionTools agentports.AgentToolRegistry
	permissions permissionports.PermissionEngine
	definitions agentdefinitionports.AgentDefinitionRepository
	catalog     *agenttoolcatalog.Catalog
	versions    agentports.RecordVersionReader
	budgets     agentports.AgentBudgetService
	trust       agentports.AgentToolTrustRepository
	extensions  agentextensionports.AgentExtensionGate
	memories    agentports.AgentMemoryService
	vectorizer  agentports.QueryVectorizer
	vectors     agentports.CatalogVectorIndex
	previews    agentports.ProposalPreviewService
	rules       agentports.ToolRuleOverrides
	changes     agentports.RecordChangeFeed
	metrics     *metrics.Assistant
	disclosures agentports.ToolDisclosureObserver
	threshold   *int
	// arguments holds each tool's compiled schema, against which every
	// call's arguments are checked before the tool sees them.
	arguments  *toolschema.Validator
	rateLimits kernelports.RateLimitStore
	pools      matchPools
	matchable  func() []string
	registry   *permission.Registry
	codeMode   codemode.Gate
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
		disclosures: p.Disclosures,
		threshold:   configuredThreshold(p.Config),
		rateLimits:  p.RateLimits,
		pools:       newMatchPools(),
		matchable:   matchableTools(p.Catalog),
		registry:    permission.NewRegistry(),
		codeMode:    codeModeGate(p.Config),
	}
}

// codeModeGate is the shipped model ledger, forced open on request only in a
// development or test deployment, where the desk bench runs.
func codeModeGate(cfg *config.Config) codemode.Gate {
	return codemode.Gate{
		Ledger: codemode.Shipped(),
		Forced: cfg != nil && (cfg.App.IsDevelopment() || cfg.App.IsTest()),
	}
}

func (s *Service) Run(
	ctx context.Context,
	req *agentports.RunRequest,
) (*agentports.RunResult, error) {
	emit := req.Emit
	if emit == nil {
		emit = func(kernelports.StreamEvent) {}
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
func (s *Service) Drive(t *Turn, fx TurnEffects) (*agentports.RunResult, error) {
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
	commandHeld := false
	placement := placementState{}
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
				fx.Emit(kernelports.StreamEvent{
					Event: agentports.AssistantEventRetrying,
					Data: agentports.AssistantRetryingEvent{
						Attempt: retries,
						Reason:  loopRestartReason,
						Kind:    agentports.RetryKindRestart,
					},
				})
				s.reask(t, fx, ReaskLoop, completion)
				continue
			}
			fx.Emit(kernelports.StreamEvent{
				Event: agentports.AssistantEventRetrying,
				Data: agentports.AssistantRetryingEvent{
					Attempt: retries + 1,
					Reason:  loopRestartReason,
					Kind:    agentports.RetryKindRestart,
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
			s.reask(t, fx, ReaskCutOff, completion)
			continue
		}

		result.Model = completion.ModelIdentifier
		result.ProviderID = completion.ProviderID
		result.ContextWindow = completion.ContextWindow
		t.providerKind = completion.ProviderKind
		tagReasoning(completion)
		if stop {
			return s.finishCutOff(t, fx, completion), nil
		}
		tagToolCalls(completion)
		distinctCallIDs(completion, t.callIDs, fx.NewCallID)

		if len(completion.ToolCalls) == 0 {
			// A turn that ends in silence reads as a hung screen. One that
			// asked the person a question has said what it needed to; any
			// other is asked once more, then given a plain line.
			if strings.TrimSpace(completion.Text) == "" && !asked {
				if retries < maxReplyRetries {
					retries++
					s.reask(t, fx, ReaskEmptyReply, completion)
					continue
				}
				fx.Emit(deltaEvent(emptyReply))
				return s.finish(result, cannedCompletion(completion, emptyReply), fx), nil
			}
			if !asked && s.holdToCommand(t, fx, completion, &commandHeld) {
				s.reask(t, fx, ReaskCommand, completion)
				continue
			}
			if s.reground(t, fx, completion, &grounding) {
				s.reask(t, fx, ReaskGrounding, completion)
				continue
			}
			s.passReply(t, fx, completion)
			if s.placeArtifacts(t, fx, completion, &placement) {
				s.reask(t, fx, ReaskPlacement, completion)
				continue
			}

			return s.finish(result, completion, fx), nil
		}

		assistantTurn := conversation.Message{
			Role:              conversation.RoleAssistant,
			Content:           completion.Text,
			ToolCalls:         toToolCallRecords(completion.ToolCalls),
			Reasoning:         completion.Reasoning,
			Model:             completion.ModelIdentifier,
			ProviderID:        completion.ProviderID,
			FallbackFrom:      fallbackOf(completion),
			InputTokens:       completion.InputTokens,
			OutputTokens:      completion.OutputTokens,
			LatencyMs:         completion.LatencyMs,
			CostUSD:           completion.CostUSD,
			CreatedAt:         fx.Now(),
			ProviderMessageID: completion.ProviderMessageID,
		}
		result.Messages = append(result.Messages, assistantTurn)
		// The thinking goes back with the calls it produced. Anthropic and the
		// Responses API both refuse a tool result whose reasoning is missing.
		t.messages = append(t.messages, agentports.Message{
			Role:          agentports.RoleAssistant,
			Content:       completion.Text,
			ToolCalls:     completion.ToolCalls,
			Reasoning:     completion.Reasoning,
			ReplyID:       completion.ProviderMessageID,
			ReplyProvider: completion.ProviderID,
		})
		fx.Emit(kernelports.StreamEvent{
			Event: agentports.AssistantEventMessage,
			Data: agentports.AssistantMessageEvent{
				Content:      assistantTurn.Content,
				ToolCalls:    s.callsWithEffects(assistantTurn.ToolCalls),
				Model:        assistantTurn.Model,
				ProviderID:   assistantTurn.ProviderID,
				FallbackFrom: assistantTurn.FallbackFrom,
			},
		})

		batch := &readBatch{}
		shownNow := false
		fannedOut := s.fanOut(t, fx, completion.ToolCalls, result.ToolCallsUsed, budget)
		for _, call := range completion.ToolCalls {
			check := &joinCheck{
				call:   call,
				used:   result.ToolCallsUsed,
				budget: budget,
				stuck:  stuck,
			}
			key, joins := t.joins(batch, check)
			if !joins && batch.size() > 0 {
				// The waiting reads settle first; with them settled the call
				// may start a batch of its own.
				s.settleReads(t, fx, batch)
				check.used = result.ToolCallsUsed
				key, joins = t.joins(batch, check)
			}
			if joins {
				s.emitToolStarted(fx, &call)
				s.load(tools, call.Name)
				batch.calls = append(batch.calls, t.dispatchCall(&call, completion.Text))
				batch.keys = append(batch.keys, key)
				continue
			}

			if _, ran := fannedOut[call.ID]; !ran {
				s.emitToolStarted(fx, &call)
			}

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
			if result.ToolCallsUsed >= budget &&
				!s.escalatesPastBudget(t, call.Name, result.ToolCallsUsed-budget) {
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
				content, shown := resolveAsk(t.localNow(fx), call.Arguments)
				outcome := toolOutcome{content: content}
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
				case !shown:
					// Nothing reached the person, so the turn has not asked
					// anything: it may not end in silence on it, and the
					// question may be asked again as it should have been.
					outcome = refusedOutcome(aitrace.OutcomeInvalid,
						"the question could not be shown as asked", "%s", content)
				default:
					asked = true
					shownNow = true
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
				if tools.offers(withdrawProposalName) && t.hasFiledProposal() {
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
				outcome, ran := fannedOut[call.ID]
				if !ran {
					outcome = s.delegate(t, fx, call)
				}
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
				if t.repeats.refused() {
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

			dispatch := t.dispatchCall(&call, completion.Text)
			outcome := fx.Dispatch(t, dispatch)
			s.settleDispatched(t, fx, &dispatch, &outcome)
		}
		s.settleReads(t, fx, batch)
		if shownNow {
			return s.finish(result, askedEnding(completion), fx), nil
		}

		s.logger.Debug("agent tool iteration",
			zap.String("agent", definition.Name),
			zap.Int("used", result.ToolCallsUsed),
			zap.Int("budget", budget),
		)
	}

	note, ending, guard := budgetSpentNote, exhaustedReply, ReaskBudgetSpent
	if stuck {
		note, ending, guard = stuckNote, stuckReply, ReaskStuck
	}
	if final := s.finalAnswer(t, fx, result, note, guard); final != nil {
		grounding.rewritten = true
		s.reground(t, fx, final, &grounding)
		s.passReply(t, fx, final)
		placement.retried = true
		s.placeArtifacts(t, fx, final, &placement)

		return s.finish(result, final, fx), nil
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

func (s *Service) emitToolStarted(effects TurnEffects, call *agentports.ToolCall) {
	effects.Emit(kernelports.StreamEvent{
		Event: agentports.AssistantEventToolStarted,
		Data: agentports.AssistantToolStartedEvent{
			CallID:    call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
			Effect:    s.ToolEffect(call.Name),
		},
	})
}

// dispatchCall is a call the loop has decided to make, with what the turn
// knew when it decided.
func (t *Turn) dispatchCall(call *agentports.ToolCall, completionText string) DispatchCall {
	return DispatchCall{
		Call:                 *call,
		CompletionText:       completionText,
		ProposedSoFar:        unwithdrawn(t.result.Actions),
		Ordinal:              t.counts.next(*call),
		AfterExternalContent: t.external,
		Taint:                t.result.Taint,
		Earlier:              t.earlier(call.Name),
		MoneySpent:           t.moneySpent(),
		Beyond:               t.beyondTier(call.Name),
		Model:                t.result.Model,
		Provider:             t.providerKind,
		ScriptPosition:       t.scriptPosition(call),
	}
}

// scriptPosition is how many calls to the script tool the turn has made up to
// and including this one, zero for any other tool.
func (t *Turn) scriptPosition(call *agentports.ToolCall) int {
	if call.Name != agentextension.ToolRunScript {
		return 0
	}
	position := 0
	for idx := range t.result.Messages {
		for _, made := range t.result.Messages[idx].ToolCalls {
			if made.Name != agentextension.ToolRunScript {
				continue
			}
			position++
			if made.ID == call.ID {
				return position
			}
		}
	}

	return position + 1
}

// settleDispatched folds what a dispatched call came to into the turn: the
// repeat guard, what it read from outside, the memories it touched, the
// budget, the transcript and what the person was shown.
func (s *Service) settleDispatched(
	t *Turn,
	effects TurnEffects,
	dispatch *DispatchCall,
	reported *ToolOutcome,
) {
	call := dispatch.Call
	outcome := reported.internal()
	outcome.dispatched = true
	if !outcome.failed {
		outcome.content += fanOutNote(call.Name, len(dispatch.Earlier)+1)
	}
	if outcome.failed {
		t.repeats.record(call, outcome.content)
	} else {
		t.repeats.ran(call)
	}
	if !outcome.failed && ReadsExternalContent(call.Name) {
		t.external = true
	}
	t.absorbTaint(effects, outcome.taint)
	t.noteMemories(effects, &outcome)
	t.result.ToolCallsUsed++
	s.recordToolResult(t, effects, call, outcome)
	if !outcome.failed {
		t.noteToolMemories(effects, call.Name)
		t.noteShown(&call)
		t.noteWorld(effects, &call, &outcome)
	}
}

// maxLoopRefusalChars bounds what a loop refusal keeps of what the model
// was told: the reason is the figure, the text only what explains it.
const maxLoopRefusalChars = 1000

// budgetSpentNote is what the model is told when its tool budget is spent.
const budgetSpentNote = "You have used every tool call this turn allows. Answer the person " +
	"now from what the tools already returned. Say plainly what you could not finish or " +
	"check, and do not ask for another tool."

// finalAnswer asks the model once more with tool calls forbidden, so a turn
// that spent its budget answers from what it gathered. The tools stay declared:
// the history holds tool calls a request without tools cannot carry, and
// dropping them would throw away the cached prefix. Ending on "I could not finish" threw
// away every lookup the turn had paid for. Nothing is returned when that ask
// fails, loops, comes back empty or asks for a tool anyway: the canned line is
// still better than any of those.
func (s *Service) finalAnswer(
	t *Turn,
	fx TurnEffects,
	result *agentports.RunResult,
	note, guard string,
) *agentports.ChatCompletionResult {
	s.reask(t, fx, guard, nil)
	req := t.completionRequest()
	req.ToolChoiceNone = true
	req.Messages = append(slices.Clone(req.Messages), agentports.NoticeMessage(note))

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
	call agentports.ToolCall,
	outcome toolOutcome,
) {
	result := t.result
	if outcome.failed && !outcome.dispatched && outcome.verdict != "" {
		result.LoopRefusals = append(result.LoopRefusals, agentports.LoopRefusal{
			CallID:   call.ID,
			ToolName: call.Name,
			Verdict:  outcome.verdict,
			Reason:   outcome.reason,
			Content:  stringutils.Ellipsize(outcome.content, maxLoopRefusalChars),
		})
	}
	if outcome.action != nil {
		result.Actions = append(result.Actions, *outcome.action)
	}
	outcome = fx.Observe(t, &call, outcome.exported()).internal()
	s.countToolOutcome(t, fx, call.Name, &outcome)
	t.keepShown(&outcome)
	t.foldReplaced(outcome.shown)
	effect := s.ToolEffect(call.Name)
	summary := summarizeOutcome(call, outcome)
	var records []agent.EntityRef
	if !outcome.failed {
		records = citedRecords(outcome.content)
	}

	fx.Emit(kernelports.StreamEvent{
		Event: agentports.AssistantEventToolFinished,
		Data: agentports.AssistantToolFinishedEvent{
			CallID:        call.ID,
			Name:          call.Name,
			Failed:        outcome.failed,
			Proposed:      outcome.action != nil && !outcome.action.Executed,
			Content:       outcome.content,
			Effect:        effect,
			Summary:       summary,
			Verdict:       outcome.verdict,
			HandOffAgents: outcome.handOff,
			Records:       records,
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
	t.messages = append(t.messages, agentports.Message{
		Role:           agentports.RoleTool,
		Content:        outcome.content,
		ToolCallID:     call.ID,
		ToolName:       call.Name,
		IsError:        outcome.failed,
		ToolReferences: t.tools.takeLoaded(),
	})
}

// NewCallID mints a tool call id no provider will have used.
func NewCallID() string { return "call_" + pulid.MustNew("tc_").String() }

// observe hands a finished call to the observer and folds what it showed the
// person back into the result the model reads.
func (s *Service) observe(
	observe agentports.ToolObserver,
	call agentports.ToolCall,
	outcome toolOutcome,
	earlier []string,
) toolOutcome {
	if request, requested := outcome.data.(agentports.DecisionRequest); requested &&
		outcome.publishes {
		return decisionRequested(observe, &call, request)
	}
	if observe == nil {
		if outcome.publishes {
			return failedOutcome("%s", unpublishableRefusal)
		}
		return outcome
	}

	shown, err := observe(agentports.ToolObservation{
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
	result *agentports.RunResult,
	completion *agentports.ChatCompletionResult,
	fx TurnEffects,
) *agentports.RunResult {
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
		Role:              conversation.RoleAssistant,
		Content:           reply,
		Reasoning:         completion.Reasoning,
		Model:             completion.ModelIdentifier,
		ProviderID:        completion.ProviderID,
		Truncated:         completion.Truncated,
		FallbackFrom:      fallbackOf(completion),
		InputTokens:       completion.InputTokens,
		OutputTokens:      completion.OutputTokens,
		LatencyMs:         completion.LatencyMs,
		CostUSD:           completion.CostUSD,
		CreatedAt:         fx.Now(),
		ProviderMessageID: completion.ProviderMessageID,
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
func fallbackOf(completion *agentports.ChatCompletionResult) *conversation.ProviderFallback {
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
func askedEnding(asking *agentports.ChatCompletionResult) *agentports.ChatCompletionResult {
	return &agentports.ChatCompletionResult{
		ModelIdentifier: asking.ModelIdentifier,
		ProviderID:      asking.ProviderID,
		ProviderKind:    asking.ProviderKind,
		ContextWindow:   asking.ContextWindow,
	}
}

func cannedCompletion(
	attempt *agentports.ChatCompletionResult,
	text string,
) *agentports.ChatCompletionResult {
	canned := &agentports.ChatCompletionResult{Text: text}
	if attempt != nil {
		canned.ModelIdentifier = attempt.ModelIdentifier
		canned.ProviderID = attempt.ProviderID
		canned.ProviderKind = attempt.ProviderKind
		canned.InputTokens = attempt.InputTokens
		canned.OutputTokens = attempt.OutputTokens
		canned.CacheReadTokens = attempt.CacheReadTokens
		canned.CacheWriteTokens = attempt.CacheWriteTokens
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
				Lookup:      agentports.EffectOf(tool) == agent.ToolEffectLookup,
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
			Escalates:           agenttoolpolicy.Escalates(definition, &policy),
			Recipe:              recipeOf(tool),
		})
	}

	return summaries
}

func (s *Service) escalatesPastBudget(t *Turn, name string, over int) bool {
	if over >= escalationReserve {
		return false
	}
	tool, ok := s.actionTools.Get(name)
	if !ok {
		return false
	}
	policy := tool.Policy()

	return agenttoolpolicy.Escalates(t.req.Definition, &policy)
}

// tierSetFor reports a tool whose tier a person set on the agent, which
// takes away the personal exemption a tool like remember otherwise has.
func tierSetFor(definition *agentdefinition.Definition, name string) bool {
	_, set := definition.ToolTiers[name]

	return set
}

func recipeOf(tool any) []string {
	if recipe, ok := tool.(agentports.RecipeTool); ok {
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
	req *agentports.RunRequest,
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
func tagToolCalls(completion *agentports.ChatCompletionResult) {
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
// always replaced. A provider's own unique ids are kept as they are. mint
// makes a new id: the turn's effects supply it, because a fresh id is random
// and workflow code may not be.
func distinctCallIDs(
	completion *agentports.ChatCompletionResult,
	used map[string]struct{},
	mint func() string,
) {
	for idx := range completion.ToolCalls {
		call := &completion.ToolCalls[idx]
		_, taken := used[call.ID]
		if call.ID == "" || taken || call.SynthesizedID {
			call.ID = mint()
			call.SynthesizedID = false
		}
		used[call.ID] = struct{}{}
	}
}

func tagReasoning(completion *agentports.ChatCompletionResult) {
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
