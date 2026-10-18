package agentruntime

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const clockLineLayout = "Monday 2006-01-02 15:04"

// Turn is one turn's working state: what the model is shown, what it may call,
// and what it has done so far.
//
// The loop that drives it is written once. Run drives a Turn in process, with
// every effect a direct call. The durable runtime drives the same Turn from
// workflow code, with every model call and every tool call its own activity.
// Everything a Turn does outside itself goes through TurnEffects, so the loop
// never reads a clock, a database or the network, which is what lets it replay.
type Turn struct {
	s      *Service
	req    *agentports.RunRequest
	budget int
	system string
	// systemStable is how many leading bytes of system are the same on every
	// turn of the agent for the person, which a provider can cache.
	systemStable int
	messages     []agentports.Message
	// evidence is the figures of the tool results the replay shortened, which
	// the grounding guard still counts as read.
	evidence []string
	tools    *toolSet
	// held is every tool the agent holds, taken when the turn opened. The
	// loop decides from it whether a call is dispatched or refused, and in
	// workflow code that decision has to replay the same way: working it out
	// again from the catalog would give a different answer to a replay run
	// by a release whose catalog or core tools changed.
	held []string
	// reads are the held tools that only look something up, taken with
	// held, which the loop may run beside each other. Nil for a turn opened
	// before they were kept, which runs every call on its own.
	reads     map[string]struct{}
	repeats   *repeatGuard
	counts    *ordinals
	questions map[string]struct{}
	decisions map[pulid.ID]struct{}
	shown     map[string][]string
	// callIDs is every tool call id the conversation holds, so a provider
	// that reuses one is given a fresh id rather than a clash.
	callIDs map[string]struct{}
	result  *agentports.RunResult
	// delegates are the agents the turn may hand a task to, taken when it
	// opened, and delegations how many tasks it has handed out.
	delegates   []agentdefinition.RuntimeDelegate
	delegations int
	// directed is a task the person handed straight to another agent, which
	// the turn runs instead of asking its own model.
	directed *agentports.DirectedTask
	// command is the extension command the person opened the turn with, and
	// the turn is held to it.
	command agentextension.Command
	// opened are the marks the turn's own subject, attachments and memories
	// added when it opened, announced once the loop starts.
	opened []agent.TaintMark
	// external says the turn has read content from outside the
	// organization. From then on every write it asks for waits for a person.
	external bool
	cutOff   cutOffState
	// world is what the turn watches for changes made elsewhere while it
	// works. Nil when nothing is watched.
	world *WorldState
	// toolMemories are the carried memories about a tool that was only on
	// hand, by tool, counted as used when the turn calls it.
	toolMemories map[string][]pulid.ID
	// artifacts are what the turn put beside the conversation, in the order
	// it did.
	artifacts    []agentports.ShownArtifact
	providerKind aiprovider.Kind
	reask        string
}

// TurnEffects is everything a turn does outside itself.
type TurnEffects interface {
	// Complete asks the model for the next step, streaming the reply as it
	// arrives. Looped reports the reply fell into a repetition and was
	// stopped; it is only meaningful when the error is nil.
	Complete(t *Turn, req *agentports.ChatCompletionRequest) (ModelReply, error)
	// Dispatch runs one tool call the loop has decided to make.
	Dispatch(t *Turn, call DispatchCall) ToolOutcome
	// DispatchReads runs reads the loop has decided to make, which may run
	// at the same time, and returns their outcomes in the order given.
	DispatchReads(t *Turn, calls []DispatchCall) []ToolOutcome
	// Find answers find_tools and loads what it found into the turn.
	Find(t *Turn, arguments map[string]any) FindAnswer
	Emit(event kernelports.StreamEvent)
	// Observe hands a finished call to whatever shows it to the person, and
	// returns the outcome with what it showed folded into what the model
	// reads. A published document is kept here.
	Observe(t *Turn, call *agentports.ToolCall, outcome ToolOutcome) ToolOutcome
	// NewCallID mints a tool call id, for a provider that gave none or reused
	// one. Minting is random, so workflow code has it recorded.
	NewCallID() string
	// Delegate hands a task to another agent and runs that agent's turn to
	// its end, reporting what it did. The loop has already checked the call.
	Delegate(t *Turn, call DelegateCall) DelegateRun
	// Interject hands the loop, between steps, what the person said while
	// it worked and, when check is set, the changes made since the turn last
	// looked to the records it watches.
	Interject(t *Turn, check *WorldCheck) Interjections
	Now() int64
}

// ModelReply is one completion as the loop sees it.
type ModelReply struct {
	Completion *agentports.ChatCompletionResult `json:"completion"`
	Looped     bool                             `json:"looped"`
	// InReasoning says the loop was in the model's thinking rather than its
	// reply.
	InReasoning bool                 `json:"inReasoning,omitempty"`
	Usage       *agentports.RunUsage `json:"usage,omitempty"`
}

func (r ModelReply) usage() *agentports.RunUsage {
	if r.Usage != nil {
		return r.Usage
	}
	if r.Completion == nil {
		return nil
	}

	return &agentports.RunUsage{
		ModelCalls:       1,
		InputTokens:      int64(r.Completion.InputTokens),
		OutputTokens:     int64(r.Completion.OutputTokens),
		CacheReadTokens:  int64(r.Completion.CacheReadTokens),
		CacheWriteTokens: int64(r.Completion.CacheWriteTokens),
		CostUSD:          r.Completion.CostUSD,
	}
}

// DispatchCall is one tool call the loop has decided to run.
type DispatchCall struct {
	Call           agentports.ToolCall        `json:"call"`
	CompletionText string                     `json:"completionText"`
	ProposedSoFar  []agentports.PendingAction `json:"proposedSoFar,omitempty"`
	// Ordinal numbers this exact call within the run, so the step key of a
	// legitimately repeated call differs from the first one's.
	Ordinal int `json:"ordinal"`
	// Taint is the outside content the turn had read when it made the call.
	// Nil is a turn opened before taint was kept.
	Taint *agent.RunTaint `json:"taint,omitempty"`
	// AfterExternalContent says the turn has read content from outside the
	// organization, so a write is proposed rather than run.
	AfterExternalContent bool     `json:"afterExternalContent,omitempty"`
	Earlier              []string `json:"earlier,omitempty"`
}

// ToolOutcome is what one tool call came to.
type ToolOutcome struct {
	Content string                    `json:"content"`
	Failed  bool                      `json:"failed"`
	Action  *agentports.PendingAction `json:"action,omitempty"`
	// Publishes marks a document still to be kept. What the model reads is
	// written once it has been.
	Publishes bool   `json:"publishes,omitempty"`
	Summary   string `json:"summary,omitempty"`
	// DelegateReport is the bounded account of a delegate_task call.
	DelegateReport *conversation.DelegateReport `json:"delegateReport,omitempty"`
	// Taint is the outside content the call read.
	Taint   []agent.TaintMark `json:"taint,omitempty"`
	Found   []string          `json:"found,omitempty"`
	HandOff []pulid.ID        `json:"handOff,omitempty"`
	Verdict string            `json:"verdict,omitempty"`
	// Memories are what a recall read back; SavedMemory is what a remember
	// kept or offered to keep.
	Memories    []pulid.ID              `json:"memories,omitempty"`
	SavedMemory *agentports.SavedMemory `json:"savedMemory,omitempty"`
	// Shown is what the call put in front of the person, which the loop
	// keeps when it is a table, for the pass over the final reply.
	Shown *agentports.ShownArtifact `json:"shown,omitempty"`
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
		Taint:          o.taint,
		Found:          o.found,
		HandOff:        o.handOff,
		Verdict:        o.verdict,
		Memories:       o.memories,
		SavedMemory:    o.saved,
		Shown:          o.shown,
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
		taint:          o.Taint,
		found:          o.Found,
		handOff:        o.HandOff,
		verdict:        o.Verdict,
		memories:       o.Memories,
		saved:          o.SavedMemory,
		shown:          o.Shown,
		data:           o.Data,
	}
}

// Request is what the turn was asked. Effects read it; the loop does not
// change it.
func (t *Turn) Request() *agentports.RunRequest { return t.req }

// Definition is the agent the turn runs as.
func (t *Turn) Definition() *agentdefinition.Definition { return t.req.Definition }

// holds reports whether the agent held a tool when the turn opened.
func (t *Turn) holds(name string) bool { return slices.Contains(t.held, name) }

// TurnState is a Turn as data, for handing a turn built in one place to a loop
// running in another: built by an activity, which may read permissions and
// history, and driven by workflow code, which may not.
type TurnState struct {
	Budget int    `json:"budget"`
	System string `json:"system"`
	// SystemStable is how much of System leads every turn unchanged.
	SystemStable int                  `json:"systemStable,omitempty"`
	Messages     []agentports.Message `json:"messages"`
	// Evidence is the figures of the tool results the replay shortened.
	Evidence []string     `json:"evidence,omitempty"`
	Tools    ToolSetState `json:"tools"`
	// Held is every tool the agent holds, as the turn opened.
	Held []string `json:"held,omitempty"`
	// Reads are the held tools that only look something up.
	Reads     []string             `json:"reads,omitempty"`
	Failures  map[string]string    `json:"failures,omitempty"`
	Ordinals  map[string]int       `json:"ordinals,omitempty"`
	Questions []string             `json:"questions,omitempty"`
	CallIDs   []string             `json:"callIds,omitempty"`
	Result    agentports.RunResult `json:"result"`
	// Delegates are the agents the turn may hand a task to, and Delegations
	// how many it has handed out.
	Delegates   []agentdefinition.RuntimeDelegate `json:"delegates,omitempty"`
	Delegations int                               `json:"delegations,omitempty"`
	// Directed is a task the person handed straight to another agent.
	Directed *agentports.DirectedTask `json:"directed,omitempty"`
	// Command is the extension command the turn is held to.
	Command agentextension.Command `json:"command,omitempty"`
	// TaintOpened are the marks the turn's opening added, still to be
	// announced. The turn's whole taint is Result.Taint.
	TaintOpened []agent.TaintMark `json:"taintOpened,omitempty"`
	// ExternalContent says the turn has read content from outside the
	// organization.
	ExternalContent bool `json:"externalContent,omitempty"`
	// World is what the turn watches for changes made elsewhere.
	World *WorldState `json:"world,omitempty"`
	// ToolMemories are the carried memories about a tool the turn has not
	// called yet.
	ToolMemories map[string][]pulid.ID `json:"toolMemories,omitempty"`
}

// ToolSetState is a turn's tool set as data. Which tools are loaded follows
// from Specs.
type ToolSetState struct {
	Specs      []kernelports.ToolSpec `json:"specs"`
	Allowed    []string               `json:"allowed"`
	Disclosed  bool                   `json:"disclosed"`
	Unattended bool                   `json:"unattended"`
	FindCalls  int                    `json:"findCalls"`
	// Delegates are the agents the turn may ask, so a search that finds
	// nothing the turn can call can name one that holds it.
	Delegates []agentdefinition.RuntimeDelegate `json:"delegates,omitempty"`
	Query     *QueryVectorState                 `json:"query,omitempty"`
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
		SystemStable:    t.systemStable,
		Messages:        t.messages,
		Evidence:        slices.Clone(t.evidence),
		Tools:           t.tools.state(),
		Held:            slices.Clone(t.held),
		Reads:           readsState(t.reads),
		Failures:        maps.Clone(t.repeats.failures),
		Ordinals:        maps.Clone(t.counts.seen),
		Questions:       questions,
		CallIDs:         callIDs,
		Result:          *t.result,
		Delegates:       slices.Clone(t.delegates),
		Delegations:     t.delegations,
		Directed:        t.directed,
		Command:         t.command,
		ExternalContent: t.external,
		TaintOpened:     slices.Clone(t.opened),
		World:           t.world.clone(),
		ToolMemories:    cloneToolMemories(t.toolMemories),
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
func (s *Service) RestoreTurn(req *agentports.RunRequest, state TurnState) *Turn {
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
	return &Turn{
		s:            s,
		req:          req,
		budget:       state.Budget,
		system:       state.System,
		systemStable: state.SystemStable,
		messages:     state.Messages,
		evidence:     state.Evidence,
		tools:        restoreToolSet(state.Tools),
		held:         state.Held,
		reads:        restoreReads(state.Reads),
		repeats:      &repeatGuard{failures: failures, reads: make(map[string]bool, 4)},
		counts:       &ordinals{seen: seen},
		questions:    questions,
		decisions:    make(map[pulid.ID]struct{}, 1),
		callIDs:      callIDs,
		result:       &result,
		delegates:    state.Delegates,
		delegations:  state.Delegations,
		directed:     state.Directed,
		command:      state.Command,
		opened:       state.TaintOpened,
		external:     state.ExternalContent,
		world:        state.World.clone(),
		toolMemories: cloneToolMemories(state.ToolMemories),
	}
}

func cloneToolMemories(byTool map[string][]pulid.ID) map[string][]pulid.ID {
	if len(byTool) == 0 {
		return nil
	}

	cloned := make(map[string][]pulid.ID, len(byTool))
	for tool, ids := range byTool {
		cloned[tool] = slices.Clone(ids)
	}

	return cloned
}

func (t *toolSet) state() ToolSetState {
	return ToolSetState{
		Specs:      append([]kernelports.ToolSpec(nil), t.specs...),
		Allowed:    append([]string(nil), t.allowed...),
		Disclosed:  t.disclosed,
		Unattended: t.unattended,
		FindCalls:  t.findCalls,
		Delegates:  slices.Clone(t.delegates),
		Query:      packQueryVector(t.query),
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
		query:      state.Query.unpack(),
	}
	for _, spec := range state.Specs {
		set.add(spec)
	}

	return set
}

// OpenTurn builds a turn: the tool set the actor may use, what earlier attempts
// of the run already learned, and the prompt. It reads permissions and the
// ledger, so it runs where those can be read: in process, or in an activity.
func (s *Service) OpenTurn(ctx context.Context, req *agentports.RunRequest) *Turn {
	definition := req.Definition
	budget := definition.MaxToolCalls
	if budget <= 0 {
		budget = agentdefinition.DefaultMaxToolCalls
	}

	runtimeContext := req.Context
	if len(runtimeContext.Tools) == 0 {
		runtimeContext.Tools = s.ToolSummaries(definition)
	}
	active := s.activeExtensions(ctx, req.Actor)
	granted := active.Grants()
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
		held = slices.DeleteFunc(
			held,
			func(name string) bool { return name == agentdefinition.CoreToolOpenPage },
		)
	}
	if req.RunID.IsNil() {
		held = slices.DeleteFunc(
			held,
			func(name string) bool { return name == agentdefinition.CoreToolRaiseException },
		)
	}
	// Guidance on a profile is a person's words, written down by the agent
	// they are talking to: a run nobody is in, or a delegate working on a
	// handed-over task, has nobody's words to write down.
	if req.Unattended || req.Delegation != nil || req.Actor.PersonUserID().IsNil() {
		held = slices.DeleteFunc(
			held,
			func(name string) bool { return name == agentdefinition.CoreToolGuideProfile },
		)
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
		query:      agentports.TurnQueryRequest(req),
		history:    history,
		unattended: req.Unattended,
		delegated:  req.Delegation != nil,
		publishes:  req.KeepsDocuments(),
		delegates:  delegates,
		decisions:  pendingDecisions(req.Proposals),
		extensions: &active,
		attachments: len(runtimeContext.Attachments) > 0 ||
			conversation.CarriesAttachments(history),
	})
	s.notePreselected(ctx, req, tools)
	var command agentextension.Command
	if req.Command != "" && s.pinCommand(tools, req.Command) {
		command = req.Command
	}
	runtimeContext.ToolsDisclosed = tools.disclosed
	runtimeContext.Artifacts = tools.offers(publishArtifactName)
	runtimeContext.Inline = req.Inline
	runtimeContext.DecisionRequests = tools.offers(requestDecisionName)
	// The prompt describes the set the person may use, not the agent's whole
	// configuration: a tool named there and refused when called reads as
	// the system refusing rather than the person lacking the right.
	runtimeContext.Tools = usableSummaries(runtimeContext.Tools, tools)
	memoryFit := s.memoriesForPrompt(ctx, req, &runtimeContext)
	runtimeContext.Memories = memoryFit.Carried
	runtimeContext.MemoriesHeldBack = memoryFit.HeldBack
	// A task handed to another agent reads memories for that agent; the card
	// under the reply says what the conversation's own agent used.
	var (
		usedMemories []pulid.ID
		toolMemories map[string][]pulid.ID
	)
	if req.Delegation == nil {
		usedMemories = memoryFit.Used
		toolMemories = memoryFit.ByTool
	}
	repeats := newRepeatGuard()
	counts := newOrdinals()
	// A delegate's turn shares the ledger of the turn that delegated, and is
	// opened once per task, so there is no earlier attempt of it to learn
	// from; the ledger's lessons are the other turn's.
	if req.Delegation == nil {
		s.seedFromLedger(ctx, req, repeats)
	}

	messages, shortened := replayHistoryWithFiles(history, req.Proposals, req.FileTexts)
	now := turnNow(&runtimeContext)
	input := req.Input
	if command != "" {
		input = commandInput(command, input)
	}
	var attached agentdefinition.RenderedAttachments
	if req.Delegation == nil {
		attached = agentdefinition.RenderAttachments(
			runtimeContext.Attachments, agentdefinition.AttachmentTextBudget,
		)
		input = attached.Block + input
		input = outOfViewDecisions(history, req.Proposals) + input
		input = agentdefinition.DescribeAnchors(
			runtimeContext.Anchors, now, runtimeContext.Timezone,
		) + input
	}
	if definition.HasContextProvider(agentdefinition.ContextClock) {
		input = clockLine(now, runtimeContext.Timezone) + input
	}
	messages = append(messages, agentports.Message{
		Role:    agentports.RoleUser,
		Content: input,
		Files:   messageFiles(input, runtimeContext.Attachments, &attached),
	})
	taint, opened := openTaint(req, &runtimeContext, now)

	prompt := definition.BuildSystemPromptParts(&runtimeContext)

	return &Turn{
		s:            s,
		req:          req,
		budget:       budget,
		system:       prompt.Stable + prompt.Volatile,
		systemStable: len(prompt.Stable),
		messages:     messages,
		evidence:     figuresOf(append(shortened, maputils.SortedValues(req.FileTexts)...)),
		tools:        tools,
		held:         held,
		reads:        s.concurrentReads(held),
		repeats:      repeats,
		counts:       counts,
		questions:    askedQuestions(history),
		decisions:    make(map[pulid.ID]struct{}, 1),
		callIDs:      usedCallIDs(req.History),
		result: &agentports.RunResult{
			Messages: []conversation.Message{{
				Role:      conversation.RoleUser,
				Content:   req.Input,
				CreatedAt: now,
			}},
			Taint:         taint,
			Fingerprint:   s.Fingerprint(definition, preferredProvider(req, definition), ""),
			UsedMemoryIDs: usedMemories,
		},
		delegates:    delegates,
		directed:     directedTask(req),
		command:      command,
		opened:       opened,
		external:     req.AfterExternalContent,
		world:        s.watchWorld(ctx, req, now),
		toolMemories: toolMemories,
	}
}

func (t *Turn) localNow(fx TurnEffects) time.Time {
	return time.Unix(fx.Now(), 0).In(timeutils.LoadLocation(t.req.Context.Timezone))
}

// turnNow is the moment the turn's context was built, which is the clock the
// prompt already reads ("Today is"), so the question's clock line, its records'
// ages and its taint marks agree with it, and a replay with a fixed clock sends
// the same request every run.
func turnNow(rc *agentdefinition.RuntimeContext) int64 {
	if rc.Now > 0 {
		return rc.Now
	}

	return timeutils.NowUnix()
}

func clockLine(now int64, timezone string) string {
	loc, name := timeutils.ResolveZone(timezone)

	return "Now: " + time.Unix(now, 0).In(loc).Format(clockLineLayout) + " " + name + "\n\n"
}

// completionRequest is what the turn is ready to send the model now, with its
// older tool results shortened when they would no longer fit the window.
func (t *Turn) completionRequest() *agentports.ChatCompletionRequest {
	req := t.req
	definition := req.Definition

	completion := &agentports.ChatCompletionRequest{
		TenantInfo:          req.Actor.TenantInfo(),
		System:              t.system,
		SystemStable:        t.systemStable,
		Messages:            t.messages,
		Tools:               withRationale(t.tools.specs),
		PreferredProviderID: preferredProvider(req, definition),
		PinPreferred:        req.PinProvider && !req.PreferredProviderID.IsNil(),
		Attribution:         turnAttribution(req),
		MaxTokens:           t.cutOff.budget,
		Reask:               t.takeReask(),
	}
	fitWindow(completion, t.window())

	return completion
}

func turnAttribution(req *agentports.RunRequest) aiusageports.AIUsageAttribution {
	definition := req.Definition
	version := definition.Version
	attribution := aiusageports.AIUsageAttribution{
		UserID:            req.Actor.PersonUserID(),
		AgentDefinitionID: definition.ID,
		ThreadID:          req.ThreadID,
		RunID:             req.RunID,
		Purpose:           req.AttributedPurpose(),
		Feature:           aiusage.FeatureAgentTurn,
		DefinitionVersion: &version,
	}
	if req.StepOwner.ID.IsNotNil() {
		attribution.OwnerKind = req.StepOwner.Kind
		attribution.OwnerID = req.StepOwner.ID
	}
	if req.Delegation != nil {
		attribution.DelegateCallID = req.Delegation.CallID
	}

	return attribution
}

// localEffects runs every effect in process, as a turn always did before it
// could run durably.
type localEffects struct {
	s        *Service
	ctx      context.Context
	emit     kernelports.AssistantStreamEmitter
	observer agentports.ToolObserver
}

func (*localEffects) Now() int64 { return timeutils.NowUnix() }

func (fx *localEffects) Complete(
	_ *Turn,
	req *agentports.ChatCompletionRequest,
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
	req *agentports.ChatCompletionRequest,
	emit kernelports.AssistantStreamEmitter,
) (ModelReply, error) {
	ctx, tally := aitrace.WithCompletionTally(ctx)
	reply, err := s.streamCompletion(ctx, req, emit)
	reply.Usage = replyUsage(tally, reply)

	return reply, err
}

func replyUsage(tally *aitrace.CompletionTally, reply ModelReply) *agentports.RunUsage {
	totals := tally.Totals()
	if totals.Attempts == 0 {
		return reply.usage()
	}

	return &agentports.RunUsage{
		ModelCalls:       totals.Attempts,
		InputTokens:      totals.InputTokens,
		OutputTokens:     totals.OutputTokens,
		CacheReadTokens:  totals.CacheReadTokens,
		CacheWriteTokens: totals.CacheWriteTokens,
		CostUSD:          totals.CostUSD,
	}
}

func (s *Service) streamCompletion(
	ctx context.Context,
	req *agentports.ChatCompletionRequest,
	emit kernelports.AssistantStreamEmitter,
) (ModelReply, error) {
	origin := aitrace.CallOriginFrom(ctx)
	ctx, span := aitrace.StartModelCall(ctx, &aitrace.ModelCallSpec{
		Anchor:          aitrace.ForAttribution(&req.Attribution),
		Feature:         string(req.Attribution.Feature),
		ActivityAttempt: origin.ActivityAttempt,
		Stream:          origin.Stream,
	})
	defer span.End()
	defer aitrace.TallyFrom(ctx).Record(span)

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
	req.RetrySink = func(notice agentports.ChatRetryNotice) { emit(retryEvent(notice)) }

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
			completion = &agentports.ChatCompletionResult{}
		}
		span.SetAttributes(aitrace.AILooped.Bool(true))

		return ModelReply{Completion: completion, Looped: true, InReasoning: thinking.tripped}, nil
	}
	if err != nil {
		aitrace.MarkFailed(span, completionErrorType(ctx, err))

		return ModelReply{Completion: completion}, err
	}

	looped := guard.looped(completion.Text)
	span.SetAttributes(aitrace.AILooped.Bool(looped))

	return ModelReply{Completion: completion, Looped: looped}, nil
}

func completionErrorType(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, agentports.ErrNoProviderConfigured):
		return "no_provider"
	case errors.Is(err, agentports.ErrProvidersResting):
		return "providers_resting"
	default:
		return "provider_error"
	}
}

// KnowsTool reports whether name is a registered tool, read or write.
func (s *Service) KnowsTool(name string) bool { return s.toolNamed(name) != nil }

func (fx *localEffects) Dispatch(t *Turn, call DispatchCall) ToolOutcome {
	return fx.s.DispatchStep(fx.ctx, t.req, call)
}

// DispatchReads runs the reads one after another in process. Its context may
// carry a transaction, which concurrent queries must not share.
func (fx *localEffects) DispatchReads(t *Turn, calls []DispatchCall) []ToolOutcome {
	outcomes := make([]ToolOutcome, 0, len(calls))
	for idx := range calls {
		outcomes = append(outcomes, fx.Dispatch(t, calls[idx]))
	}

	return outcomes
}

func (fx *localEffects) Find(t *Turn, arguments map[string]any) FindAnswer {
	answer := fx.s.resolveFind(fx.ctx, t.tools, t.req.Actor, arguments)
	fx.s.noteFound(fx.ctx, t.req, t.tools, arguments, &answer)

	return answer
}

func (fx *localEffects) Emit(event kernelports.StreamEvent) { fx.emit(event) }

func (fx *localEffects) Observe(
	t *Turn,
	call *agentports.ToolCall,
	outcome ToolOutcome,
) ToolOutcome {
	return fx.s.observe(fx.observer, *call, outcome.internal(), t.earlier(call.Name)).exported()
}

func (*localEffects) NewCallID() string { return NewCallID() }

func (fx *localEffects) Interject(t *Turn, check *WorldCheck) Interjections {
	if check == nil || fx.s.changes == nil || t.req.Actor == nil {
		return Interjections{}
	}
	found, err := fx.s.changes.Since(fx.ctx, &agentports.RecordChangesRequest{
		OrganizationID: t.req.Actor.OrganizationID,
		BusinessUnitID: t.req.Actor.BusinessUnitID,
		Cursor:         check.Cursor,
		Records:        check.Records,
	})
	if err != nil {
		fx.s.logger.Debug("could not read changes to the turn's records", zap.Error(err))
		return Interjections{}
	}

	return Interjections{World: found}
}

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
	observe agentports.ToolObserver,
	call *agentports.ToolCall,
	outcome ToolOutcome,
) ToolOutcome {
	return s.observe(observe, *call, outcome.internal(), nil).exported()
}

func (s *Service) ObserveDispatch(
	observe agentports.ToolObserver,
	call *DispatchCall,
	outcome *ToolOutcome,
) ToolOutcome {
	return s.observe(observe, call.Call, outcome.internal(), call.Earlier).exported()
}

// PublishStep keeps a document a publish call asked for, and says what the
// model reads about it. A durable turn runs it in an activity: the document is
// read again from the call, because what the loop parsed stays in workflow
// code.
func (s *Service) PublishStep(
	observe agentports.ToolObserver,
	call *agentports.ToolCall,
) ToolOutcome {
	if call.Name == requestDecisionName {
		return s.observe(observe, *call, decisionStepOutcome(call.Arguments), nil).exported()
	}

	return s.observe(observe, *call, publishOutcome(call.Arguments), nil).exported()
}

func deltaEvent(text string) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: agentports.AssistantEventDelta,
		Data:  agentports.AssistantDeltaEvent{Text: text},
	}
}

func reasoningEvent(text string) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: agentports.AssistantEventReasoning,
		Data:  agentports.AssistantReasoningEvent{Text: text},
	}
}

func retryEvent(notice agentports.ChatRetryNotice) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: agentports.AssistantEventRetrying,
		Data: agentports.AssistantRetryingEvent{
			Attempt:     notice.Attempt,
			Provider:    notice.Provider,
			Reason:      notice.Reason,
			Kind:        notice.Kind,
			WaitSeconds: notice.WaitSeconds,
			MaxAttempts: notice.MaxAttempts,
		},
	}
}

// DispatchStep runs one tool call the loop decided to make, claimed in the
// run's ledger when req carries one. It is what a durable tool activity calls:
// the loop decides in workflow code, and the call runs here, where a database
// and the tools can be reached.
func (s *Service) DispatchStep(
	ctx context.Context,
	req *agentports.RunRequest,
	call DispatchCall,
) ToolOutcome {
	outcome := s.guardedDispatch(ctx, &guardedDispatchParams{
		req:            req,
		call:           call.Call,
		completionText: call.CompletionText,
		proposedSoFar:  call.ProposedSoFar,
		ordinal:        call.Ordinal,
		taint:          call.Taint,
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
	req *agentports.RunRequest,
	state ToolSetState,
	arguments map[string]any,
) FoundTools {
	set := restoreToolSet(state)
	set.self = req.Definition.ID
	set.usable = func(name string) bool {
		return len(s.permittedTools(ctx, req.Actor, []string{name})) == 1
	}

	before := len(set.specs)
	answer := s.resolveFind(ctx, set, req.Actor, arguments)
	s.noteFound(ctx, req, set, arguments, &answer)

	loaded := make([]string, 0, len(set.specs)-before)
	for _, spec := range set.specs[before:] {
		loaded = append(loaded, spec.Name)
	}

	return FoundTools{
		Content: answer.Content,
		Loaded:  loaded,
		Found:   answer.Found,
		HandOff: answer.HandOff,
	}
}

const getToolPrefix = "get_"

func (t *Turn) earlier(name string) []string {
	if !strings.HasPrefix(name, getToolPrefix) {
		return nil
	}

	return slices.Clone(t.shown[name])
}

func (t *Turn) noteShown(call *agentports.ToolCall) {
	if !strings.HasPrefix(call.Name, getToolPrefix) || call.ID == "" {
		return
	}
	if t.shown == nil {
		t.shown = make(map[string][]string, 1)
	}
	t.shown[call.Name] = append(t.shown[call.Name], call.ID)
}
