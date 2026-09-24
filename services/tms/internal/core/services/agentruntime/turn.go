package agentruntime

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
)

// Turn is one turn's working state: what the model is shown, what it may call,
// and what it has done so far.
//
// The loop that drives it is written once. Run drives a Turn in process, with
// every effect a direct call. The durable runtime drives the same Turn from
// workflow code, with every model call and every tool call its own activity.
// Everything a Turn does outside itself goes through TurnEffects, so the loop
// never reads a clock, a database or the network, which is what lets it replay.
type Turn struct {
	s        *Service
	req      *serviceports.RunRequest
	budget   int
	system   string
	messages []serviceports.Message
	tools    *toolSet
	// held is every tool the agent holds, taken when the turn opened. The
	// loop decides from it whether a call is dispatched or refused, and in
	// workflow code that decision has to replay the same way: working it out
	// again from the catalog would give a different answer to a replay run
	// by a release whose catalog or core tools changed.
	held      []string
	repeats   *repeatGuard
	counts    *ordinals
	questions map[string]struct{}
	// callIDs is every tool call id the conversation holds, so a provider
	// that reuses one is given a fresh id rather than a clash.
	callIDs map[string]struct{}
	result  *serviceports.RunResult
	// delegates are the agents the turn may hand a task to, taken when it
	// opened, and delegations how many tasks it has handed out.
	delegates   []agentdefinition.RuntimeDelegate
	delegations int
	// external says the turn has read content from outside the
	// organization. From then on every write it asks for waits for a person.
	external bool
}

// TurnEffects is everything a turn does outside itself.
type TurnEffects interface {
	// Complete asks the model for the next step, streaming the reply as it
	// arrives. Looped reports the reply fell into a repetition and was
	// stopped; it is only meaningful when the error is nil.
	Complete(t *Turn, req *serviceports.ChatCompletionRequest) (ModelReply, error)
	// Dispatch runs one tool call the loop has decided to make.
	Dispatch(t *Turn, call DispatchCall) ToolOutcome
	// Find answers find_tools and loads what it found into the turn.
	Find(t *Turn, arguments map[string]any) string
	Emit(event serviceports.StreamEvent)
	// Observe hands a finished call to whatever shows it to the person, and
	// returns the outcome with what it showed folded into what the model
	// reads. A published document is kept here.
	Observe(t *Turn, call *serviceports.ToolCall, outcome ToolOutcome) ToolOutcome
	// NewCallID mints a tool call id, for a provider that gave none or reused
	// one. Minting is random, so workflow code has it recorded.
	NewCallID() string
	// Delegate hands a task to another agent and runs that agent's turn to
	// its end, reporting what it did. The loop has already checked the call.
	Delegate(t *Turn, call DelegateCall) DelegateRun
	// Supports reports whether a change to the loop's shape applies to this
	// turn. In process every change does; in workflow code an execution
	// started before the change keeps the shape it started with, so its
	// history still replays.
	Supports(change string) bool
	Now() int64
}

// ModelReply is one completion as the loop sees it.
type ModelReply struct {
	Completion *serviceports.ChatCompletionResult `json:"completion"`
	Looped     bool                               `json:"looped"`
	// InReasoning says the loop was in the model's thinking rather than its
	// reply.
	InReasoning bool `json:"inReasoning,omitempty"`
}

// DispatchCall is one tool call the loop has decided to run.
type DispatchCall struct {
	Call           serviceports.ToolCall        `json:"call"`
	CompletionText string                       `json:"completionText"`
	ProposedSoFar  []serviceports.PendingAction `json:"proposedSoFar,omitempty"`
	// Ordinal numbers this exact call within the run, so the step key of a
	// legitimately repeated call differs from the first one's.
	Ordinal int `json:"ordinal"`
	// AfterExternalContent says the turn has read content from outside the
	// organization, so a write is proposed rather than run.
	AfterExternalContent bool `json:"afterExternalContent,omitempty"`
}

// ToolOutcome is what one tool call came to.
type ToolOutcome struct {
	Content string                      `json:"content"`
	Failed  bool                        `json:"failed"`
	Action  *serviceports.PendingAction `json:"action,omitempty"`
	// Publishes marks a document still to be kept. What the model reads is
	// written once it has been.
	Publishes bool   `json:"publishes,omitempty"`
	Summary   string `json:"summary,omitempty"`
	// DelegateReport is the bounded account of a delegate_task call.
	DelegateReport *conversation.DelegateReport `json:"delegateReport,omitempty"`
	// Data is what a query tool returned before it was encoded for the model.
	// It never crosses a durable boundary: whatever needs it runs where the
	// tool ran.
	Data any `json:"-"`
}

func (o toolOutcome) exported() ToolOutcome {
	return ToolOutcome{
		Content:        o.content,
		Failed:         o.failed,
		Action:         o.action,
		Publishes:      o.publishes,
		Summary:        o.summary,
		DelegateReport: o.delegateReport,
		Data:           o.data,
	}
}

func (o ToolOutcome) internal() toolOutcome {
	return toolOutcome{
		content:        o.Content,
		failed:         o.Failed,
		action:         o.Action,
		publishes:      o.Publishes,
		summary:        o.Summary,
		delegateReport: o.DelegateReport,
		data:           o.Data,
	}
}

// Request is what the turn was asked. Effects read it; the loop does not
// change it.
func (t *Turn) Request() *serviceports.RunRequest { return t.req }

// Definition is the agent the turn runs as.
func (t *Turn) Definition() *agentdefinition.Definition { return t.req.Definition }

// holds reports whether the agent held a tool when the turn opened.
func (t *Turn) holds(name string) bool { return slices.Contains(t.held, name) }

// TurnState is a Turn as data, for handing a turn built in one place to a loop
// running in another: built by an activity, which may read permissions and
// history, and driven by workflow code, which may not.
type TurnState struct {
	Budget   int                    `json:"budget"`
	System   string                 `json:"system"`
	Messages []serviceports.Message `json:"messages"`
	Tools    ToolSetState           `json:"tools"`
	// Held is every tool the agent holds, as the turn opened. A state from
	// before it was kept has none, and the turn works it out again.
	Held      []string               `json:"held,omitempty"`
	Failures  map[string]string      `json:"failures,omitempty"`
	Ordinals  map[string]int         `json:"ordinals,omitempty"`
	Questions []string               `json:"questions,omitempty"`
	CallIDs   []string               `json:"callIds,omitempty"`
	Result    serviceports.RunResult `json:"result"`
	// Delegates are the agents the turn may hand a task to, and Delegations
	// how many it has handed out.
	Delegates   []agentdefinition.RuntimeDelegate `json:"delegates,omitempty"`
	Delegations int                               `json:"delegations,omitempty"`
	// ExternalContent says the turn has read content from outside the
	// organization.
	ExternalContent bool `json:"externalContent,omitempty"`
}

// ToolSetState is a turn's tool set as data. Which tools are loaded follows
// from Specs.
type ToolSetState struct {
	Specs      []serviceports.ToolSpec `json:"specs"`
	Allowed    []string                `json:"allowed"`
	Disclosed  bool                    `json:"disclosed"`
	Unattended bool                    `json:"unattended"`
	FindCalls  int                     `json:"findCalls"`
	// Delegates are the agents the turn may ask, so a search that finds
	// nothing the turn can call can name one that holds it.
	Delegates []agentdefinition.RuntimeDelegate `json:"delegates,omitempty"`
}

// State captures the turn as data.
func (t *Turn) State() TurnState {
	// Sorted, because a map ranges in a different order every time and this
	// may be built in workflow code, which has to replay identically.
	questions := slices.Sorted(maps.Keys(t.questions))
	callIDs := slices.Sorted(maps.Keys(t.callIDs))

	return TurnState{
		Budget:          t.budget,
		System:          t.system,
		Messages:        t.messages,
		Tools:           t.tools.state(),
		Held:            slices.Clone(t.held),
		Failures:        maps.Clone(t.repeats.failures),
		Ordinals:        maps.Clone(t.counts.seen),
		Questions:       questions,
		CallIDs:         callIDs,
		Result:          *t.result,
		Delegates:       slices.Clone(t.delegates),
		Delegations:     t.delegations,
		ExternalContent: t.external,
	}
}

// ToolsState is the turn's current tool set as data, for answering find_tools
// somewhere that may check permissions.
func (t *Turn) ToolsState() ToolSetState { return t.tools.state() }

// LoadTools makes the named tools callable, as a find_tools answered elsewhere
// found them. A name the turn may not call is ignored.
func (t *Turn) LoadTools(names []string) {
	for _, name := range names {
		t.s.load(t.tools, name)
	}
}

// RestoreTurn rebuilds a Turn from its state. The rebuilt tool set cannot check
// permissions, so whatever answers its find_tools must: see TurnEffects.Find.
func (s *Service) RestoreTurn(req *serviceports.RunRequest, state TurnState) *Turn {
	result := state.Result
	questions := make(map[string]struct{}, len(state.Questions))
	for _, question := range state.Questions {
		questions[question] = struct{}{}
	}

	callIDs := make(map[string]struct{}, len(state.CallIDs))
	for _, id := range state.CallIDs {
		callIDs[id] = struct{}{}
	}

	failures := state.Failures
	if failures == nil {
		failures = make(map[string]string, 4)
	}
	seen := state.Ordinals
	if seen == nil {
		seen = make(map[string]int, 4)
	}
	// Every agent holds the core tools, so an empty set is a state written
	// before the set was kept, and the turn is decided the way it was then.
	held := state.Held
	if len(held) == 0 {
		held = s.heldTools(req.Definition)
	}

	return &Turn{
		s:           s,
		req:         req,
		budget:      state.Budget,
		system:      state.System,
		messages:    state.Messages,
		tools:       restoreToolSet(state.Tools),
		held:        held,
		repeats:     &repeatGuard{failures: failures},
		counts:      &ordinals{seen: seen},
		questions:   questions,
		callIDs:     callIDs,
		result:      &result,
		delegates:   state.Delegates,
		delegations: state.Delegations,
		external:    state.ExternalContent,
	}
}

func (t *toolSet) state() ToolSetState {
	return ToolSetState{
		Specs:      append([]serviceports.ToolSpec(nil), t.specs...),
		Allowed:    append([]string(nil), t.allowed...),
		Disclosed:  t.disclosed,
		Unattended: t.unattended,
		FindCalls:  t.findCalls,
		Delegates:  slices.Clone(t.delegates),
	}
}

func restoreToolSet(state ToolSetState) *toolSet {
	set := &toolSet{
		loaded:     make(map[string]struct{}, len(state.Specs)),
		allowed:    state.Allowed,
		disclosed:  state.Disclosed,
		unattended: state.Unattended,
		findCalls:  state.FindCalls,
		delegates:  state.Delegates,
	}
	for _, spec := range state.Specs {
		set.add(spec)
	}

	return set
}

// OpenTurn builds a turn: the tool set the actor may use, what earlier attempts
// of the run already learned, and the prompt. It reads permissions and the
// ledger, so it runs where those can be read: in process, or in an activity.
func (s *Service) OpenTurn(ctx context.Context, req *serviceports.RunRequest) *Turn {
	definition := req.Definition
	budget := definition.MaxToolCalls
	if budget <= 0 {
		budget = agentdefinition.DefaultMaxToolCalls
	}

	runtimeContext := req.Context
	if len(runtimeContext.Tools) == 0 {
		runtimeContext.Tools = s.ToolSummaries(definition)
	}
	granted := s.activeExtensions(ctx, req.Actor).grants()
	runtimeContext.Tools = withSummaries(runtimeContext.Tools, s.summarize(definition, granted))

	// Another agent's steps on a task this one handed it are never replayed:
	// the model only ever saw its own call and the answer it got back. The
	// thread's reader leaves them out already; this holds for any caller.
	history := modelHistory(req.History)

	held := withGrants(s.heldTools(definition), granted)
	// Moving the person around the app is for the agent they are talking
	// to. A delegate that navigated would pull them away mid-answer to a
	// page they never asked for.
	if req.Delegation != nil {
		held = slices.DeleteFunc(held, func(name string) bool { return name == agentdefinition.CoreToolOpenPage })
	}
	// Only the agent a person is talking to delegates, and only to agents
	// they may use; the tool is held on that turn and on no other, so a
	// turn working for another agent cannot hand its task on.
	var delegates []agentdefinition.RuntimeDelegate
	if req.MayDelegate() {
		delegates = slices.Clone(runtimeContext.Delegates)
		held = append(held, delegateTaskName)
	} else {
		runtimeContext.Delegates = nil
	}
	tools := s.newToolSet(ctx, toolSetRequest{
		definition: definition,
		held:       held,
		actor:      req.Actor,
		input:      req.Input,
		history:    history,
		unattended: req.Unattended,
		delegated:  req.Delegation != nil,
		publishes:  req.KeepsDocuments(),
		delegates:  delegates,
	})
	runtimeContext.ToolsDisclosed = tools.disclosed
	runtimeContext.Artifacts = tools.offers(publishArtifactName)
	// The prompt describes the set the person may use, not the agent's whole
	// configuration: a tool named there and refused when called reads as
	// the system refusing rather than the person lacking the right.
	runtimeContext.Tools = usableSummaries(runtimeContext.Tools, tools)
	repeats := newRepeatGuard()
	counts := newOrdinals()
	// A delegate's turn shares the ledger of the turn that delegated, and is
	// opened once per task, so there is no earlier attempt of it to learn
	// from; the ledger's lessons are the other turn's.
	if req.Delegation == nil {
		s.seedFromLedger(ctx, req, repeats)
	}

	messages := toAdapterMessages(history, req.Proposals)
	input := req.Input
	if req.Delegation == nil {
		input = outOfViewDecisions(history, req.Proposals) + input
	}
	messages = append(messages, serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: input,
	})

	return &Turn{
		s:         s,
		req:       req,
		budget:    budget,
		system:    definition.BuildSystemPrompt(runtimeContext),
		messages:  messages,
		tools:     tools,
		held:      held,
		repeats:   repeats,
		counts:    counts,
		questions: askedQuestions(history),
		callIDs:   usedCallIDs(req.History),
		result: &serviceports.RunResult{
			Messages: []conversation.Message{{
				Role:      conversation.RoleUser,
				Content:   req.Input,
				CreatedAt: timeutils.NowUnix(),
			}},
		},
		delegates: delegates,
	}
}

// completionRequest is what the turn is ready to send the model now.
func (t *Turn) completionRequest() *serviceports.ChatCompletionRequest {
	req := t.req
	definition := req.Definition

	return &serviceports.ChatCompletionRequest{
		TenantInfo:          req.Actor.TenantInfo(),
		System:              t.system,
		Messages:            t.messages,
		Tools:               t.tools.specs,
		PreferredProviderID: preferredProvider(req, definition),
		PinPreferred:        req.PinProvider && !req.PreferredProviderID.IsNil(),
		Attribution: serviceports.AIUsageAttribution{
			UserID:            req.Actor.UserID,
			AgentDefinitionID: definition.ID,
			ThreadID:          req.ThreadID,
			RunID:             req.RunID,
		},
	}
}

// localEffects runs every effect in process, as a turn always did before it
// could run durably.
type localEffects struct {
	s        *Service
	ctx      context.Context
	emit     serviceports.AssistantStreamEmitter
	observer serviceports.ToolObserver
}

func (fx *localEffects) Supports(string) bool { return true }

func (*localEffects) Now() int64 { return timeutils.NowUnix() }

func (fx *localEffects) Complete(
	_ *Turn,
	req *serviceports.ChatCompletionRequest,
) (ModelReply, error) {
	return fx.s.StreamCompletion(fx.ctx, req, fx.Emit)
}

// StreamCompletion asks the model for one reply and streams it to emit as it
// arrives: the text, the thinking, and any restart. A reply that falls into a
// loop is cut off the moment it is seen, so the provider stops writing, and is
// reported as Looped rather than returned as an answer.
//
// It is the model call wherever a turn runs: inline for a turn in process, and
// inside the model activity for a durable one, where emit publishes to the
// run's stream.
func (s *Service) StreamCompletion(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	emit serviceports.AssistantStreamEmitter,
) (ModelReply, error) {
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	guard := newReplyGuard(cancelStream)
	// The thinking is watched the way the reply is. A small model that falls
	// into a loop does it in its reasoning as readily as in its answer, and an
	// unwatched trace streamed thousands of repeated fragments to the reader
	// before the provider ran out of tokens.
	thinking := newReplyGuard(cancelStream)
	req.ReasoningSink = func(delta string) {
		if thinking.feed(delta) {
			emit(reasoningEvent(delta))
		}
	}
	req.RetrySink = func(notice serviceports.ChatRetryNotice) { emit(retryEvent(notice)) }

	completion, err := s.completion.StreamChat(streamCtx, req, func(delta string) {
		if guard.feed(delta) {
			emit(deltaEvent(delta))
		}
	})
	cancelStream()
	// A loop in the reply or the thinking cancels the stream, so the call
	// returns the cancellation rather than a completion. The turn itself is
	// fine and is asked again.
	if (guard.tripped || thinking.tripped) && ctx.Err() == nil {
		if completion == nil {
			completion = &serviceports.ChatCompletionResult{}
		}

		return ModelReply{Completion: completion, Looped: true, InReasoning: thinking.tripped}, nil
	}
	if err != nil {
		return ModelReply{Completion: completion}, err
	}

	return ModelReply{Completion: completion, Looped: guard.looped(completion.Text)}, nil
}

// KnowsTool reports whether name is a registered tool, read or write.
func (s *Service) KnowsTool(name string) bool { return s.toolNamed(name) != nil }

func (fx *localEffects) Dispatch(t *Turn, call DispatchCall) ToolOutcome {
	return fx.s.DispatchStep(fx.ctx, t.req, call)
}

func (fx *localEffects) Find(t *Turn, arguments map[string]any) string {
	return fx.s.resolveFind(t.tools, arguments)
}

func (fx *localEffects) Emit(event serviceports.StreamEvent) { fx.emit(event) }

func (fx *localEffects) Observe(
	_ *Turn,
	call *serviceports.ToolCall,
	outcome ToolOutcome,
) ToolOutcome {
	return fx.s.observe(fx.observer, *call, outcome.internal()).exported()
}

func (*localEffects) NewCallID() string { return NewCallID() }

// Delegate declines in process. A turn is offered delegate_task only when it
// is driven as a workflow, where the delegate's turn runs as activities of its
// own; this answers a model that names the tool anyway.
func (*localEffects) Delegate(_ *Turn, call DelegateCall) DelegateRun {
	return DelegateRun{
		Declined: "handing a task to " + call.Delegate.Name + " is not available here. " +
			"Do what you can with your own tools.",
	}
}

// ObserveCall hands a finished call to observe and folds what it showed the
// person into what the model reads. It is what a durable tool activity runs
// after the call, where the call's raw result exists.
func (s *Service) ObserveCall(
	observe serviceports.ToolObserver,
	call *serviceports.ToolCall,
	outcome ToolOutcome,
) ToolOutcome {
	return s.observe(observe, *call, outcome.internal()).exported()
}

// PublishStep keeps a document a publish call asked for, and says what the
// model reads about it. A durable turn runs it in an activity: the document is
// read again from the call, because what the loop parsed stays in workflow
// code.
func (s *Service) PublishStep(
	observe serviceports.ToolObserver,
	call *serviceports.ToolCall,
) ToolOutcome {
	return s.observe(observe, *call, publishOutcome(call.Arguments)).exported()
}

func deltaEvent(text string) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventDelta,
		Data:  serviceports.AssistantDeltaEvent{Text: text},
	}
}

func reasoningEvent(text string) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventReasoning,
		Data:  serviceports.AssistantReasoningEvent{Text: text},
	}
}

func retryEvent(notice serviceports.ChatRetryNotice) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventRetrying,
		Data: serviceports.AssistantRetryingEvent{
			Attempt:     notice.Attempt,
			Provider:    notice.Provider,
			Reason:      notice.Reason,
			Kind:        notice.Kind,
			WaitSeconds: notice.WaitSeconds,
		},
	}
}

// DispatchStep runs one tool call the loop decided to make, claimed in the
// run's ledger when req carries one. It is what a durable tool activity calls:
// the loop decides in workflow code, and the call runs here, where a database
// and the tools can be reached.
func (s *Service) DispatchStep(
	ctx context.Context,
	req *serviceports.RunRequest,
	call DispatchCall,
) ToolOutcome {
	outcome := s.guardedDispatch(ctx, guardedDispatchParams{
		req:            req,
		call:           call.Call,
		completionText: call.CompletionText,
		proposedSoFar:  call.ProposedSoFar,
		ordinal:        call.Ordinal,
		afterExternal:  call.AfterExternalContent,
	})
	outcome.summary = summarizeOutcome(call.Call, outcome)

	return outcome.exported()
}

// FindFor answers find_tools for a turn held as data, and names the tools it
// made callable so the turn can load the same ones. An empty search names what
// exists beyond the agent only when the actor could use it, which takes a
// permission check, so this runs where permissions can be read.
func (s *Service) FindFor(
	ctx context.Context,
	req *serviceports.RunRequest,
	state ToolSetState,
	arguments map[string]any,
) (string, []string) {
	set := restoreToolSet(state)
	set.usable = func(name string) bool {
		return len(s.permittedTools(ctx, req.Actor, []string{name})) == 1
	}

	before := len(set.specs)
	content := s.resolveFind(set, arguments)

	loaded := make([]string, 0, len(set.specs)-before)
	for _, spec := range set.specs[before:] {
		loaded = append(loaded, spec.Name)
	}

	return content, loaded
}
