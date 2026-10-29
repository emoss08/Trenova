package agentruntime

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/scriptbroker"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnmemory"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	Completion  llmports.CompletionService
	QueryTools  agenttoolports.AgentQueryToolRegistry
	ActionTools agenttoolports.AgentToolRegistry
	Permissions permissionports.PermissionEngine
	Catalog     *agenttoolcatalog.Catalog
	// Versions is optional. Without it proposals are still made, just without
	// the record pinned; with it the executor can refuse a change to a record
	// that moved on since the proposal.
	Versions agenttoolports.RecordVersionReader `optional:"true"`
	// Budgets is optional. With it an automatic write past its tool's daily
	// cap is refused before it runs.
	Budgets agentbudgetports.AgentBudgetService     `optional:"true"`
	Trust   agenttoolports.AgentToolTrustRepository `optional:"true"`
	// Extensions is optional. Without it no extension's tools are offered.
	Extensions agentextensionports.AgentExtensionGate `optional:"true"`
	// Memories is optional. With it every memory a prompt carries is
	// counted as used; without it the prompt still carries them.
	Memories       agentmemoryports.AgentMemoryService `optional:"true"`
	Vectorizer     retrievalports.QueryVectorizer      `optional:"true"`
	CatalogVectors retrievalports.CatalogVectorIndex   `optional:"true"`
	// Previews is optional. With it a held write's target is pinned, and
	// what it would do kept as its baseline, in one read-only snapshot;
	// without it the target is pinned alone.
	Previews agentruntimeports.ProposalPreviewService `optional:"true"`
	// Rules is optional. With it a tool an organization holds lower than its
	// declared rule is decided and tainted by the organization's rule.
	Rules agenttoolports.ToolRuleOverrides `optional:"true"`
	// Changes is optional. With it a turn is told when a record it is
	// working with changes elsewhere while it works.
	Changes realtimeports.RecordChangeFeed `optional:"true"`
	// Metrics is optional. With it every tool call is counted by tool and by
	// what became of it; without it nothing is counted.
	Metrics *metrics.Registry `optional:"true"`
	// Definitions is optional. With it a search for a tool the agent does not
	// hold names the person's other agents that hold it.
	Definitions agentdefinitionports.AgentDefinitionRepository `optional:"true"`
	AIConfig    *aiconfig.Config                               `optional:"true"`
	AppConfig   *appconfig.Config                              `optional:"true"`
	Disclosures agenttoolports.ToolDisclosureObserver          `optional:"true"`
	Shadow      *agentshadow.Resolver                          `optional:"true"`
	// Scripts is optional. With it the code-mode gate reads the chat
	// providers' script settings; without it no model is offered scripts
	// unless the bench forces them.
	Scripts llmports.ScriptProviders `optional:"true"`
}

type Service struct {
	logger      *zap.Logger
	actionTools agenttoolports.AgentToolRegistry
	memories    agentmemoryports.AgentMemoryService
	changes     realtimeports.RecordChangeFeed
	metrics     *aimetrics.Assistant
	codeMode    codemode.Gate
	access      *toolaccess.Access
	dispatcher  *tooldispatch.Dispatcher
	streamer    *modelstream.Streamer
}

func New(p Params) *Service {
	accessParts, dispatchParts := accessParams(&p), dispatcherParams(&p)

	return assemble(&p, &accessParts, &dispatchParts)
}

// assemble builds the runtime from its parts: the tool access every part
// checks against, the dispatcher that runs tool calls and the streamer that
// asks the model.
func assemble(
	p *Params,
	accessParts *toolaccess.AccessParams,
	dispatchParts *tooldispatch.DispatcherParams,
) *Service {
	access := toolaccess.NewAccess(accessParts)
	dispatchParts.Access = access

	return &Service{
		logger:      p.Logger.Named("service.agentruntime"),
		actionTools: p.ActionTools,
		memories:    p.Memories,
		changes:     p.Changes,
		metrics:     metrics.AssistantFrom(p.Metrics),
		codeMode:    dispatchParts.CodeMode,
		access:      access,
		streamer:    modelstream.NewStreamer(p.Completion, access),
		dispatcher:  tooldispatch.NewDispatcher(dispatchParts),
	}
}

func accessParams(p *Params) toolaccess.AccessParams {
	return toolaccess.AccessParams{
		Logger:         p.Logger,
		QueryTools:     p.QueryTools,
		ActionTools:    p.ActionTools,
		Permissions:    p.Permissions,
		Catalog:        p.Catalog,
		Extensions:     p.Extensions,
		Vectorizer:     p.Vectorizer,
		CatalogVectors: p.CatalogVectors,
		Definitions:    p.Definitions,
		Threshold:      toolaccess.ConfiguredThreshold(p.AIConfig),
		Disclosures:    p.Disclosures,
		Scripts:        p.Scripts,
	}
}

func dispatcherParams(p *Params) tooldispatch.DispatcherParams {
	return tooldispatch.DispatcherParams{
		Logger:      p.Logger,
		QueryTools:  p.QueryTools,
		ActionTools: p.ActionTools,
		Permissions: p.Permissions,
		Versions:    p.Versions,
		Budgets:     p.Budgets,
		Trust:       p.Trust,
		Previews:    p.Previews,
		Rules:       p.Rules,
		CodeMode:    tooldispatch.CodeModeGate(p.AppConfig),
		Shadow:      tooldispatch.ShadowFrom(p.Shadow),
		Scripts:     p.Scripts,
		Brokers:     scriptbroker.Open,
	}
}

func (s *Service) Run(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
) (*agentruntimeports.RunResult, error) {
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
// Nothing here reaches outside the turn except through effects, which is what lets
// the same loop run in process and as durable workflow code.
func (s *Service) Drive(t *Turn, effects TurnEffects) (*agentruntimeports.RunResult, error) {
	t.announceOpened(effects)
	turnmemory.Announce(t, effects)
	if t.directed != nil {
		return s.driveDirected(t, effects), nil
	}

	loop := &driveLoop{}
	for t.result.ToolCallsUsed < t.budget && !loop.stuck {
		if step := s.driveRound(t, effects, loop); step.done {
			return step.result, step.err
		}
	}

	return s.driveSpent(t, effects, loop), nil
}

// driveLoop is what the loop carries from one round to the next.
type driveLoop struct {
	retries     int
	asked       bool
	stuck       bool
	grounding   replycheck.GroundingState
	commandHeld bool
	placement   replycheck.PlacementState
}

// driveStep is how a round ended: done ends the turn with result and err,
// otherwise the loop asks the model again.
type driveStep struct {
	done   bool
	result *agentruntimeports.RunResult
	err    error
}

func finished(result *agentruntimeports.RunResult) driveStep {
	return driveStep{done: true, result: result}
}

func (s *Service) driveRound(t *Turn, effects TurnEffects, loop *driveLoop) driveStep {
	result := t.result
	s.interject(t, effects)
	reply, err := effects.Complete(t, t.completionRequest())
	result.Usage = result.Usage.Add(reply.TotalUsage())
	completion := reply.Completion
	if err == nil && reply.Looped {
		return s.loopedReply(t, effects, loop, &reply)
	}
	if err != nil {
		// What ran travels with the error. The caller decides whether to
		// keep it, but it cannot keep what it was never handed: a model that
		// died after a tool wrote something used to take the record of that
		// write with it.
		return driveStep{done: true, result: result, err: err}
	}

	retry, stop := s.handleCutOff(t, effects, completion)
	if retry {
		s.reask(t, effects, ReaskCutOff, completion)
		return driveStep{}
	}

	result.Model = completion.ModelIdentifier
	result.ProviderID = completion.ProviderID
	result.ContextWindow = completion.ContextWindow
	t.providerKind = completion.ProviderKind
	modelstream.TagReasoning(completion)
	if stop {
		return finished(s.finishCutOff(t, effects, completion))
	}
	modelstream.TagToolCalls(completion)
	turnhistory.DistinctCallIDs(completion, t.callIDs, effects.NewCallID)

	if len(completion.ToolCalls) == 0 {
		return s.driveReply(t, effects, loop, completion)
	}

	s.recordAssistantTurn(t, effects, completion)
	if s.runToolCalls(t, effects, loop, completion) {
		return finished(s.finish(result, askedEnding(completion), effects))
	}

	s.logger.Debug("agent tool iteration",
		zap.String("agent", t.req.Definition.Name),
		zap.Int("used", result.ToolCallsUsed),
		zap.Int("budget", t.budget),
	)

	return driveStep{}
}

func (s *Service) loopedReply(
	t *Turn,
	effects TurnEffects,
	loop *driveLoop,
	reply *modelstream.Reply,
) driveStep {
	completion := reply.Completion
	s.logger.Warn("agent reply fell into a loop; discarding it",
		zap.String("agent", t.req.Definition.Name),
		zap.String("model", completion.ModelIdentifier),
		zap.Bool("in_reasoning", reply.InReasoning),
		zap.Int("retry", loop.retries+1),
	)
	if loop.retries < loopguard.MaxReplyRetries {
		loop.retries++
		effects.Emit(kernelports.StreamEvent{
			Event: assistantstreamports.AssistantEventRetrying,
			Data: assistantstreamports.AssistantRetryingEvent{
				Attempt: loop.retries,
				Reason:  loopguard.LoopRestartReason,
				Kind:    llmports.RetryKindRestart,
			},
		})
		s.reask(t, effects, ReaskLoop, completion)
		return driveStep{}
	}
	effects.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventRetrying,
		Data: assistantstreamports.AssistantRetryingEvent{
			Attempt: loop.retries + 1,
			Reason:  loopguard.LoopRestartReason,
			Kind:    llmports.RetryKindRestart,
		},
	})
	effects.Emit(modelstream.DeltaEvent(loopguard.LoopedReply))

	return finished(s.finish(
		t.result,
		modelstream.CannedCompletion(completion, loopguard.LoopedReply),
		effects,
	))
}

// driveReply settles a completion that called no tools: it is the reply,
// unless a guard sends the model back to fix it.
func (s *Service) driveReply(
	t *Turn,
	effects TurnEffects,
	loop *driveLoop,
	completion *llmports.ChatCompletionResult,
) driveStep {
	// A turn that ends in silence reads as a hung screen. One that
	// asked the person a question has said what it needed to; any
	// other is asked once more, then given a plain line.
	if strings.TrimSpace(completion.Text) == "" && !loop.asked {
		if loop.retries < loopguard.MaxReplyRetries {
			loop.retries++
			s.reask(t, effects, ReaskEmptyReply, completion)
			return driveStep{}
		}
		effects.Emit(modelstream.DeltaEvent(loopguard.EmptyReply))
		return finished(s.finish(
			t.result,
			modelstream.CannedCompletion(completion, loopguard.EmptyReply),
			effects,
		))
	}
	if !loop.asked && s.holdToCommand(t, effects, completion, &loop.commandHeld) {
		s.reask(t, effects, ReaskCommand, completion)
		return driveStep{}
	}
	if s.reground(t, effects, completion, &loop.grounding) {
		s.reask(t, effects, ReaskGrounding, completion)
		return driveStep{}
	}
	s.passReply(t, effects, completion)
	if s.placeArtifacts(t, effects, completion, &loop.placement) {
		s.reask(t, effects, ReaskPlacement, completion)
		return driveStep{}
	}

	return finished(s.finish(t.result, completion, effects))
}

func (s *Service) recordAssistantTurn(
	t *Turn,
	effects TurnEffects,
	completion *llmports.ChatCompletionResult,
) {
	assistantTurn := conversation.Message{
		Role:              conversation.RoleAssistant,
		Content:           completion.Text,
		ToolCalls:         turnhistory.ToToolCallRecords(completion.ToolCalls),
		Reasoning:         completion.Reasoning,
		Model:             completion.ModelIdentifier,
		ProviderID:        completion.ProviderID,
		FallbackFrom:      modelstream.FallbackOf(completion),
		InputTokens:       completion.InputTokens,
		OutputTokens:      completion.OutputTokens,
		LatencyMs:         completion.LatencyMs,
		CostUSD:           completion.CostUSD,
		CreatedAt:         effects.Now(),
		ProviderMessageID: completion.ProviderMessageID,
	}
	t.result.Messages = append(t.result.Messages, assistantTurn)
	// The thinking goes back with the calls it produced. Anthropic and the
	// Responses API both refuse a tool result whose reasoning is missing.
	t.messages = append(t.messages, llmports.Message{
		Role:          llmports.RoleAssistant,
		Content:       completion.Text,
		ToolCalls:     completion.ToolCalls,
		Reasoning:     completion.Reasoning,
		ReplyID:       completion.ProviderMessageID,
		ReplyProvider: completion.ProviderID,
	})
	effects.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventMessage,
		Data: assistantstreamports.AssistantMessageEvent{
			Content:      assistantTurn.Content,
			ToolCalls:    s.access.CallsWithEffects(assistantTurn.ToolCalls),
			Model:        assistantTurn.Model,
			ProviderID:   assistantTurn.ProviderID,
			FallbackFrom: assistantTurn.FallbackFrom,
		},
	})
}

// roundCall is one tool call of a completion, with what the round knows
// about it.
type roundCall struct {
	call      *llmports.ToolCall
	text      string
	fannedOut map[string]toolcall.Result
}

// runToolCalls runs every tool call of a completion and reports whether a
// question reached the person, which ends the turn.
func (s *Service) runToolCalls(
	t *Turn,
	effects TurnEffects,
	loop *driveLoop,
	completion *llmports.ChatCompletionResult,
) bool {
	result := t.result
	batch := &tooldispatch.ReadBatch{}
	shownNow := false
	fannedOut := delegation.FanOut(
		t, s.handing(t, effects), completion.ToolCalls, result.ToolCallsUsed, t.budget,
	)
	for _, call := range completion.ToolCalls {
		check := &tooldispatch.JoinCheck{
			Call:   call,
			Used:   result.ToolCallsUsed,
			Budget: t.budget,
			Stuck:  loop.stuck,
		}
		key, joins := t.joins(batch, check)
		if !joins && batch.Size() > 0 {
			// The waiting reads settle first; with them settled the call
			// may start a batch of its own.
			s.settleReads(t, effects, batch)
			check.Used = result.ToolCallsUsed
			key, joins = t.joins(batch, check)
		}
		if joins {
			s.emitToolStarted(effects, &call)
			s.access.Load(t.tools, call.Name)
			batch.Calls = append(batch.Calls, t.dispatchCall(&call, completion.Text))
			batch.Keys = append(batch.Keys, key)
			continue
		}

		if _, ran := fannedOut[call.ID]; !ran {
			s.emitToolStarted(effects, &call)
		}
		if s.runToolCall(t, effects, loop, &roundCall{
			call:      &call,
			text:      completion.Text,
			fannedOut: fannedOut,
		}) {
			shownNow = true
		}
	}
	s.settleReads(t, effects, batch)

	return shownNow
}

// runToolCall answers one tool call the loop runs on its own, and reports
// whether it was a question that reached the person.
func (s *Service) runToolCall(t *Turn, effects TurnEffects, loop *driveLoop, rc *roundCall) bool {
	call := rc.call
	// Arguments that did not parse are not arguments. The tool used to
	// run on the empty map that stood in for them, and a list tool given
	// no filters lists everything.
	if call.ArgumentsError != "" {
		outcome := invalidArguments(call)
		s.recordToolResult(t, effects, call, &outcome)
		// It counts against the budget like any call. A model whose output
		// limit cuts every call short sends the same broken call each time,
		// and a loop that did not count it asked the model again, and paid for
		// it, without end.
		t.result.ToolCallsUsed++
		return false
	}
	if outcome, refused := s.overBudget(t, call); refused {
		s.recordToolResult(t, effects, call, &outcome)
		return false
	}

	switch call.Name {
	case runtimetools.FindToolsName:
		outcome := s.findTools(t, effects, call)
		s.recordToolResult(t, effects, call, &outcome)
		return false
	case askuser.Name:
		outcome, shown := s.askUser(t, effects, loop, call)
		t.result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		return shown
	}

	if outcome, handled := runtimetools.Answer(t, t.tools.Offers, call); handled {
		t.result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		return false
	}

	// Only a turn that holds delegate_task hands out work, and a turn
	// working for another agent is told why it cannot. Any other turn
	// that names it is answered as for any tool it does not hold, as
	// it was before the tool existed.
	if call.Name == runtimetools.DelegateTaskName &&
		(t.Holds(runtimetools.DelegateTaskName) || t.req.Delegation != nil) {
		outcome, ran := rc.fannedOut[call.ID]
		if !ran {
			outcome = delegation.Delegate(t, s.handing(t, effects), call)
		}
		t.result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		return false
	}

	s.runHeldCall(t, effects, loop, rc)

	return false
}

func invalidArguments(call *llmports.ToolCall) toolcall.Result {
	return toolcall.Refused(
		aitrace.OutcomeInvalid,
		"its arguments were not valid JSON",
		"Tool %q was not run: its arguments were not valid JSON (%s). "+
			"This usually means the reply hit its output limit partway through "+
			"the call. Send it again with complete arguments.",
		call.Name, call.ArgumentsError,
	)
}

// overBudget refuses a call past the turn's tool budget. The budget is per
// call, not per batch. A model that asks for ten tools in one completion does
// not get ten when it was allowed one; the calls past the line are answered,
// so the model knows, but not run.
func (s *Service) overBudget(t *Turn, call *llmports.ToolCall) (toolcall.Result, bool) {
	used := t.result.ToolCallsUsed
	if used < t.budget || s.escalatesPastBudget(t, call.Name, used-t.budget) {
		return toolcall.Result{}, false
	}

	return toolcall.Refused(
		aitrace.OutcomeOverBudget,
		"the turn's tool budget is spent",
		"Tool %q was not run: this turn's tool budget of %d is spent. "+
			"Answer with what you have.",
		call.Name, t.budget,
	), true
}

func (s *Service) findTools(t *Turn, effects TurnEffects, call *llmports.ToolCall) toolcall.Result {
	tools := t.tools
	tools.FindCalls++
	if tools.FindCalls > toolaccess.MaxFindCalls {
		// Past the cap the search is charged, so a model that
		// only ever searches still runs out of turn.
		t.result.ToolCallsUsed++
		return toolcall.Refused(
			aitrace.OutcomeOverBudget,
			"the turn's tool searches are spent",
			"You have searched for tools %d times this turn. Use what is "+
				"loaded, or tell the person what you could not find.",
			toolaccess.MaxFindCalls,
		)
	}

	answer := effects.Find(t, call.Arguments)

	return toolcall.Result{
		Content: answer.Content + toolaccess.SearchesLeft(tools.FindCalls),
		Found:   answer.Found,
		HandOff: answer.HandOff,
	}
}

func (s *Service) askUser(
	t *Turn,
	effects TurnEffects,
	loop *driveLoop,
	call *llmports.ToolCall,
) (toolcall.Result, bool) {
	tools := t.tools
	question := askuser.ComparableQuestion(argcheck.String(call.Arguments, "question"))
	_, repeated := t.questions[question]
	content, shown := askuser.Resolve(t.localNow(effects), call.Arguments)
	switch {
	case !tools.Offers(askuser.Name) && t.req.Delegation != nil:
		return toolcall.Refused(
			aitrace.OutcomeDenied,
			"a delegated task cannot ask the person",
			"%s",
			delegation.DelegatedAskRefusal,
		), false
	case !tools.Offers(askuser.Name):
		return toolcall.Refused(aitrace.OutcomeDenied,
			"nobody is in the conversation to ask", "%s", askuser.UnattendedRefusal), false
	case question != "" && repeated:
		return toolcall.Refused(aitrace.OutcomeDuplicate,
			"the same question was already asked", "%s", askuser.RepeatedRefusal), false
	case !shown:
		// Nothing reached the person, so the turn has not asked
		// anything: it may not end in silence on it, and the
		// question may be asked again as it should have been.
		return toolcall.Refused(aitrace.OutcomeInvalid,
			"the question could not be shown as asked", "%s", content), false
	default:
		loop.asked = true
		if question != "" {
			t.questions[question] = struct{}{}
		}

		return toolcall.Result{Content: content}, true
	}
}

// runHeldCall runs a call to a tool the loop dispatches: one the agent holds,
// unless the turn stopped for repeats or the same call already failed or read.
func (s *Service) runHeldCall(t *Turn, effects TurnEffects, loop *driveLoop, rc *roundCall) {
	call := rc.call
	result := t.result
	if !t.Holds(call.Name) {
		outcome := s.access.UnheldOutcome(t.tools, call.Name)
		result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		return
	}
	if loop.stuck {
		outcome := toolcall.Refused(
			aitrace.OutcomeDuplicate,
			"the turn stopped calling tools after repeated failed calls",
			"%s",
			loopguard.StoppedCallText(call.Name),
		)
		s.recordToolResult(t, effects, call, &outcome)
		return
	}

	// A tool the agent holds but was not sent runs anyway — the
	// configuration is the grant, disclosure only decides what was
	// shown — and is loaded, so the next request carries its schema.
	s.access.Load(t.tools, call.Name)

	if previous, repeated := t.repeats.Seen(call); repeated {
		outcome := toolcall.Refused(
			aitrace.OutcomeDuplicate,
			"the same call already failed this turn",
			"%s",
			loopguard.RepeatRefusal(call.Name, previous),
		)
		result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		if t.repeats.Refused() {
			loop.stuck = true
		}
		return
	}

	if t.repeats.ReadBefore(call) {
		outcome := toolcall.Result{Content: loopguard.RepeatedRead(call.Name)}
		result.ToolCallsUsed++
		s.recordToolResult(t, effects, call, &outcome)
		return
	}

	dispatch := t.dispatchCall(call, rc.text)
	outcome := effects.Dispatch(t, &dispatch)
	s.settleDispatched(t, effects, &dispatch, &outcome)
}

// driveSpent ends a turn whose tool budget is spent, or whose tools closed
// for repeats, with an answer from what the tools returned.
func (s *Service) driveSpent(
	t *Turn,
	effects TurnEffects,
	loop *driveLoop,
) *agentruntimeports.RunResult {
	result := t.result
	note, ending, guard := budgetSpentNote, exhaustedReply, ReaskBudgetSpent
	if loop.stuck {
		note, ending, guard = loopguard.StuckNote, loopguard.StuckReply, ReaskStuck
	}
	if final := s.finalAnswer(t, effects, result, note, guard); final != nil {
		loop.grounding.Rewritten = true
		s.reground(t, effects, final, &loop.grounding)
		s.passReply(t, effects, final)
		loop.placement.Retried = true
		s.placeArtifacts(t, effects, final, &loop.placement)

		return s.finish(result, final, effects)
	}

	result.Exhausted = true
	result.Reply = ending
	result.Messages = append(result.Messages, conversation.Message{
		Role:      conversation.RoleAssistant,
		Content:   ending,
		CreatedAt: effects.Now(),
	})
	effects.Emit(modelstream.DeltaEvent(ending))

	return result
}

func (s *Service) emitToolStarted(effects TurnEffects, call *llmports.ToolCall) {
	effects.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventToolStarted,
		Data: assistantstreamports.AssistantToolStartedEvent{
			CallID:    call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
			Effect:    s.access.ToolEffect(call.Name),
		},
	})
}

// dispatchCall is a call the loop has decided to make, with what the turn
// knew when it decided.
func (t *Turn) dispatchCall(call *llmports.ToolCall, completionText string) toolcall.Call {
	continues := t.scripts.Continue(call, t.result.Messages)

	return toolcall.Call{
		Call:                 *call,
		CompletionText:       completionText,
		ProposedSoFar:        runtimetools.Unwithdrawn(t.result.Actions),
		Ordinal:              t.counts.Next(call),
		AfterExternalContent: t.external,
		Taint:                t.result.Taint,
		Earlier:              t.earlier(call.Name),
		MoneySpent:           t.MoneySpent(),
		Beyond:               t.beyondTier(call.Name),
		Model:                t.result.Model,
		Provider:             t.providerKind,
		ProviderID:           t.result.ProviderID,
		ScriptContinues:      continues,
		ScriptPosition:       t.scripts.Position(call, t.result.Messages),
		ScriptReach:          t.scriptReach(call),
	}
}

// scriptReach is what the turn reached beyond the agent's own tools, for the
// call that runs a script. find_tools told the model a read from another desk
// was callable here, and the script, which saw only the agent's own, had no
// such name: Haiku guessed three spellings of the fuel read and spent the
// turn's scripts on them.
func (t *Turn) scriptReach(call *llmports.ToolCall) map[string]agent.AutonomyTier {
	if call.Name != codemode.ToolRunScript || len(t.beyond) == 0 {
		return nil
	}

	return maps.Clone(t.beyond)
}

// settleDispatched folds what a dispatched call came to into the turn: the
// repeat guard, what it read from outside, the memories it touched, the
// budget, the transcript and what the person was shown.
func (s *Service) settleDispatched(
	t *Turn,
	effects TurnEffects,
	dispatch *toolcall.Call,
	reported *toolcall.Outcome,
) {
	call := dispatch.Call
	outcome := reported.Result()
	outcome.Dispatched = true
	if !outcome.Failed {
		outcome.Content += loopguard.FanOutNote(call.Name, len(dispatch.Earlier)+1)
	}
	if outcome.Failed {
		t.repeats.Record(&call, outcome.Content)
	} else {
		t.repeats.Ran(&call)
	}
	if !outcome.Failed && toolaccess.ReadsExternalContent(call.Name) {
		t.external = true
	}
	t.absorbTaint(effects, outcome.Taint)
	turnmemory.Note(t, effects, &outcome)
	t.result.ToolCallsUsed++
	s.recordToolResult(t, effects, &call, &outcome)
	if !outcome.Failed {
		turnmemory.NoteTool(t, effects, call.Name)
		t.noteShown(&call)
		t.scripts.Note(&call, &outcome)
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
	effects TurnEffects,
	result *agentruntimeports.RunResult,
	note, guard string,
) *llmports.ChatCompletionResult {
	s.reask(t, effects, guard, nil)
	req := t.completionRequest()
	req.ToolChoiceNone = true
	req.Messages = append(slices.Clone(req.Messages), llmports.NoticeMessage(note))

	reply, err := effects.Complete(t, req)
	result.Usage = result.Usage.Add(reply.TotalUsage())
	completion := reply.Completion
	if err != nil || reply.Looped || completion == nil ||
		len(completion.ToolCalls) > 0 || strings.TrimSpace(completion.Text) == "" {
		return nil
	}

	result.Model = completion.ModelIdentifier
	result.ProviderID = completion.ProviderID
	result.ContextWindow = completion.ContextWindow
	modelstream.TagReasoning(completion)

	return completion
}

// recordToolResult files one tool's outcome into the run, the adapter history
// and the stream. find_tools and a dispatched tool both come through here so a
// loaded-tools answer is recorded exactly like any other tool result — it is one
// to the model, and a transcript that hid it would not explain the turn.
func (s *Service) recordToolResult(
	t *Turn,
	effects TurnEffects,
	call *llmports.ToolCall,
	reported *toolcall.Result,
) {
	result := t.result
	if reported.Failed && !reported.Dispatched && reported.Verdict != "" {
		result.LoopRefusals = append(result.LoopRefusals, agentruntimeports.LoopRefusal{
			CallID:   call.ID,
			ToolName: call.Name,
			Verdict:  reported.Verdict,
			Reason:   reported.Reason,
			Content:  stringutils.Ellipsize(reported.Content, maxLoopRefusalChars),
		})
	}
	if reported.Action != nil {
		result.Actions = append(result.Actions, *reported.Action)
	}
	result.Actions = append(result.Actions, reported.Batch...)
	observable := reported.Outcome()
	observed := effects.Observe(t, call, &observable)
	outcome := observed.Result()
	s.countToolOutcome(t, effects, call.Name, &outcome)
	t.keepShown(&outcome)
	t.foldReplaced(outcome.Shown)
	effect := s.access.ToolEffect(call.Name)
	summary := toolsummary.Outcome(call, &outcome)
	var records []agentpagecontext.EntityRef
	if !outcome.Failed {
		records = toolresult.CitedRecords(outcome.Content)
	}

	effects.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventToolFinished,
		Data: assistantstreamports.AssistantToolFinishedEvent{
			CallID:        call.ID,
			Name:          call.Name,
			Failed:        outcome.Failed,
			Proposed:      outcome.Action != nil && !outcome.Action.Executed,
			Content:       outcome.Content,
			Effect:        effect,
			Summary:       summary,
			Verdict:       outcome.Verdict,
			HandOffAgents: outcome.HandOff,
			Records:       records,
		},
	})

	result.Messages = append(result.Messages, conversation.Message{
		Role:           conversation.RoleTool,
		Content:        outcome.Content,
		ToolCallID:     call.ID,
		ToolName:       call.Name,
		ToolFailed:     outcome.Failed,
		ToolVerdict:    outcome.Verdict,
		ToolEffect:     effect,
		ToolSummary:    summary,
		FoundTools:     outcome.Found,
		HandOffAgents:  outcome.HandOff,
		DelegateReport: outcome.DelegateReport,
		CreatedAt:      effects.Now(),
	})
	t.messages = append(t.messages, llmports.Message{
		Role:           llmports.RoleTool,
		Content:        outcome.Content,
		ToolCallID:     call.ID,
		ToolName:       call.Name,
		IsError:        outcome.Failed,
		ToolReferences: t.tools.TakeLoaded(),
	})
}

// NewCallID mints a tool call id no provider will have used.
func NewCallID() string { return "call_" + pulid.MustNew("tc_").String() }

func (s *Service) finish(
	result *agentruntimeports.RunResult,
	completion *llmports.ChatCompletionResult,
	effects TurnEffects,
) *agentruntimeports.RunResult {
	guarded, outputDecision := agentguard.EvaluateOutput(completion.Text)
	if outputDecision.Altered {
		s.logger.Warn("agent reply altered by output guard",
			zap.String("rule", outputDecision.MatchedRule),
			zap.String("model", completion.ModelIdentifier),
		)
		result.OutputAltered = true
		result.OutputRule = outputDecision.MatchedRule
		effects.Emit(replycheck.ReplyReplacedEvent(guarded, outputAlteredReason))
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
		FallbackFrom:      modelstream.FallbackOf(completion),
		InputTokens:       completion.InputTokens,
		OutputTokens:      completion.OutputTokens,
		LatencyMs:         completion.LatencyMs,
		CostUSD:           completion.CostUSD,
		CreatedAt:         effects.Now(),
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

// cannedCompletion stands a fixed line in for a reply the model could not
// give, keeping the provider and usage the attempt was charged under.
func askedEnding(asking *llmports.ChatCompletionResult) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		ModelIdentifier: asking.ModelIdentifier,
		ProviderID:      asking.ProviderID,
		ProviderKind:    asking.ProviderKind,
		ContextWindow:   asking.ContextWindow,
	}
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

func (s *Service) escalatesPastBudget(t *Turn, name string, over int) bool {
	if over >= toolaccess.EscalationReserve {
		return false
	}
	tool, ok := s.actionTools.Get(name)
	if !ok {
		return false
	}
	policy := tool.Policy()

	return agenttoolpolicy.Escalates(t.req.Definition, &policy)
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
	req *agentruntimeports.RunRequest,
	definition *agentdefinition.Definition,
) pulid.ID {
	if !req.PreferredProviderID.IsNil() {
		return req.PreferredProviderID
	}

	return definition.PreferredProviderID
}

// usableSummaries keeps the summaries of the tools the turn may call and
// marks which are loaded, so the prompt and the request agree.
func usableSummaries(
	summaries []agentturn.ToolSummary,
	tools *toolaccess.ToolSet,
) []agentturn.ToolSummary {
	kept := make([]agentturn.ToolSummary, 0, len(summaries))
	for _, summary := range summaries {
		if !slices.Contains(tools.Allowed, summary.Name) {
			continue
		}
		_, loaded := tools.Loaded[summary.Name]
		summary.Loaded = loaded
		kept = append(kept, summary)
	}

	return kept
}
