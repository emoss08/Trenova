package agentruntime

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/proposalledger"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnscripts"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
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
	req    *agentruntimeports.RunRequest
	budget int
	system string
	// systemStable is how many leading bytes of system are the same on every
	// turn of the agent for the person, which a provider can cache.
	systemStable int
	messages     []llmports.Message
	// evidence is the figures of the tool results the replay shortened, which
	// the grounding guard still counts as read.
	evidence []string
	tools    *toolaccess.ToolSet
	// held is every tool the agent holds, taken when the turn opened. The
	// loop decides from it whether a call is dispatched or refused, and in
	// workflow code that decision has to replay the same way: working it out
	// again from the catalog would give a different answer to a replay run
	// by a release whose catalog or core tools changed.
	held   []string
	beyond map[string]agent.AutonomyTier
	mode   toolaccess.ProfileMode
	// reads are the held tools that only look something up, taken with
	// held, which the loop may run beside each other. Nil for a turn opened
	// before they were kept, which runs every call on its own.
	reads     map[string]struct{}
	repeats   *loopguard.RepeatGuard
	counts    *tooldispatch.Ordinals
	questions map[string]struct{}
	decisions map[pulid.ID]struct{}
	shown     map[string][]string
	// scripts are the scripts the turn ran: which a later call may continue
	// and which read nothing.
	scripts turnscripts.Ledger
	// callIDs is every tool call id the conversation holds, so a provider
	// that reuses one is given a fresh id rather than a clash.
	callIDs map[string]struct{}
	result  *agentruntimeports.RunResult
	// delegates are the agents the turn may hand a task to, taken when it
	// opened, and delegations how many tasks it has handed out.
	delegates   []agentturn.RuntimeDelegate
	delegations int
	// directed is a task the person handed straight to another agent, which
	// the turn runs instead of asking its own model.
	directed *agentruntimeports.DirectedTask
	// command is the extension command the person opened the turn with, and
	// the turn is held to it.
	command agentextension.Command
	// opened are the marks the turn's own subject, attachments and memories
	// added when it opened, announced once the loop starts.
	opened []agent.TaintMark
	// external says the turn has read content from outside the
	// organization. From then on every write it asks for waits for a person.
	external bool
	cutOff   loopguard.CutOffState
	// world is what the turn watches for changes made elsewhere while it
	// works. Nil when nothing is watched.
	world *worldwatch.State
	// toolMemories are the carried memories about a tool that was only on
	// hand, by tool, counted as used when the turn calls it.
	toolMemories map[string][]pulid.ID
	// artifacts are what the turn put beside the conversation, in the order
	// it did.
	artifacts    []agentruntimeports.ShownArtifact
	providerKind aiprovider.Kind
	reask        string
}

// TurnEffects is everything a turn does outside itself.
type TurnEffects interface {
	// Complete asks the model for the next step, streaming the reply as it
	// arrives. Looped reports the reply fell into a repetition and was
	// stopped; it is only meaningful when the error is nil.
	Complete(t *Turn, req *llmports.ChatCompletionRequest) (modelstream.Reply, error)
	// Dispatch runs one tool call the loop has decided to make.
	Dispatch(t *Turn, call *toolcall.Call) toolcall.Outcome
	// DispatchReads runs reads the loop has decided to make, which may run
	// at the same time, and returns their outcomes in the order given.
	DispatchReads(t *Turn, calls []toolcall.Call) []toolcall.Outcome
	// Find answers find_tools and loads what it found into the turn.
	Find(t *Turn, arguments map[string]any) toolaccess.FindAnswer
	Emit(event kernelports.StreamEvent)
	// Observe hands a finished call to whatever shows it to the person, and
	// returns the outcome with what it showed folded into what the model
	// reads. A published document is kept here.
	Observe(t *Turn, call *llmports.ToolCall, outcome *toolcall.Outcome) toolcall.Outcome
	// NewCallID mints a tool call id, for a provider that gave none or reused
	// one. Minting is random, so workflow code has it recorded.
	NewCallID() string
	// Delegate hands tasks to other agents and runs each agent's turn to its
	// end, side by side where it can, reporting what each did in the order
	// given. The loop has already checked the calls.
	Delegate(t *Turn, calls []delegation.Call) []delegation.Run
	// Interject hands the loop, between steps, what the person said while
	// it worked and, when check is set, the changes made since the turn last
	// looked to the records it watches.
	Interject(t *Turn, check *worldwatch.Check) worldwatch.Interjections
	Now() int64
}

// Request is what the turn was asked. Effects read it; the loop does not
// change it.
func (t *Turn) Request() *agentruntimeports.RunRequest { return t.req }

// Definition is the agent the turn runs as.
func (t *Turn) Definition() *agentdefinition.Definition { return t.req.Definition }

// Holds reports whether the agent held a tool when the turn opened.
func (t *Turn) Holds(name string) bool {
	if slices.Contains(t.held, name) {
		return true
	}
	_, reached := t.beyond[name]

	return reached
}

func (t *Turn) beyondTier(name string) agent.AutonomyTier {
	if slices.Contains(t.held, name) {
		return ""
	}

	return t.beyond[name]
}

func (t *Turn) Mode() toolaccess.ProfileMode { return t.mode }

// Messages is what the model is shown, as the turn holds it now.
func (t *Turn) Messages() []llmports.Message { return t.messages }

// System is the turn's system prompt.
func (t *Turn) System() string { return t.system }

// Evidence is the figures of the tool results the replay shortened, which
// the grounding guard still counts as read.
func (t *Turn) Evidence() []string { return t.evidence }

// Result is what the turn has done so far.
func (t *Turn) Result() *agentruntimeports.RunResult { return t.result }

// Artifacts are what the turn put beside the conversation, in the order it
// did.
func (t *Turn) Artifacts() []agentruntimeports.ShownArtifact { return t.artifacts }

// Decisions are the proposals and plans the turn has asked the person to
// decide.
func (t *Turn) Decisions() map[pulid.ID]struct{} { return t.decisions }

// Delegates are the agents the turn may hand a task to.
func (t *Turn) Delegates() []agentturn.RuntimeDelegate { return t.delegates }

// ToolMemories are the carried memories about a tool the turn has not called
// yet, by tool.
func (t *Turn) ToolMemories() map[string][]pulid.ID { return t.toolMemories }

// TurnState is a Turn as data, for handing a turn built in one place to a loop
// running in another: built by an activity, which may read permissions and
// history, and driven by workflow code, which may not.
type TurnState struct {
	Budget int    `json:"budget"`
	System string `json:"system"`
	// SystemStable is how much of System leads every turn unchanged.
	SystemStable int                `json:"systemStable,omitempty"`
	Messages     []llmports.Message `json:"messages"`
	// Evidence is the figures of the tool results the replay shortened.
	Evidence []string                `json:"evidence,omitempty"`
	Tools    toolaccess.ToolSetState `json:"tools"`
	// Held is every tool the agent holds, as the turn opened.
	Held   []string                      `json:"held,omitempty"`
	Beyond map[string]agent.AutonomyTier `json:"beyond,omitempty"`
	Mode   toolaccess.ProfileMode        `json:"mode,omitempty"`
	// Reads are the held tools that only look something up.
	Reads     []string                    `json:"reads,omitempty"`
	Failures  map[string]string           `json:"failures,omitempty"`
	Ordinals  map[string]int              `json:"ordinals,omitempty"`
	Questions []string                    `json:"questions,omitempty"`
	CallIDs   []string                    `json:"callIds,omitempty"`
	Result    agentruntimeports.RunResult `json:"result"`
	// Delegates are the agents the turn may hand a task to, and Delegations
	// how many it has handed out.
	Delegates   []agentturn.RuntimeDelegate `json:"delegates,omitempty"`
	Delegations int                         `json:"delegations,omitempty"`
	// Directed is a task the person handed straight to another agent.
	Directed *agentruntimeports.DirectedTask `json:"directed,omitempty"`
	// Command is the extension command the turn is held to.
	Command agentextension.Command `json:"command,omitempty"`
	// TaintOpened are the marks the turn's opening added, still to be
	// announced. The turn's whole taint is Result.Taint.
	TaintOpened []agent.TaintMark `json:"taintOpened,omitempty"`
	// ExternalContent says the turn has read content from outside the
	// organization.
	ExternalContent bool `json:"externalContent,omitempty"`
	// World is what the turn watches for changes made elsewhere.
	World *worldwatch.State `json:"world,omitempty"`
	// ToolMemories are the carried memories about a tool the turn has not
	// called yet.
	ToolMemories map[string][]pulid.ID `json:"toolMemories,omitempty"`
	// Continuable are the scripts a later call may continue, by call id, and
	// Continuing each call that continues one, with the id it continues.
	Continuable map[string]codemode.Continuation `json:"continuable,omitempty"`
	Continuing  map[string]string                `json:"continuing,omitempty"`

	ProviderKind aiprovider.Kind `json:"providerKind,omitempty"`
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
		Tools:           t.tools.State(),
		Held:            slices.Clone(t.held),
		Beyond:          maps.Clone(t.beyond),
		Mode:            t.mode,
		Reads:           tooldispatch.ReadsState(t.reads),
		Failures:        maps.Clone(t.repeats.Failures),
		Ordinals:        maps.Clone(t.counts.Seen),
		Questions:       questions,
		CallIDs:         callIDs,
		Result:          *t.result,
		Delegates:       slices.Clone(t.delegates),
		Delegations:     t.delegations,
		Directed:        t.directed,
		Command:         t.command,
		ExternalContent: t.external,
		TaintOpened:     slices.Clone(t.opened),
		World:           t.world.Clone(),
		ToolMemories:    cloneToolMemories(t.toolMemories),
		Continuable:     maps.Clone(t.scripts.Continuable),
		Continuing:      maps.Clone(t.scripts.Continuing),
		ProviderKind:    t.providerKind,
	}
}

// ToolsState is the turn's current tool set as data, for answering find_tools
// somewhere that may check permissions.
func (t *Turn) ToolsState() toolaccess.ToolSetState { return t.tools.State() }

// LoadTools makes the named tools callable, as a find_tools answered elsewhere
// found them. A name the turn may not call is ignored.
func (t *Turn) LoadTools(names []string) {
	for _, name := range names {
		t.s.access.Load(t.tools, name)
	}
}

// RestoreTurn rebuilds a Turn from its state. The rebuilt tool set cannot check
// permissions, so whatever answers its find_tools must: see TurnEffects.Find.
func (s *Service) RestoreTurn(req *agentruntimeports.RunRequest, state *TurnState) *Turn {
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
		tools:        toolaccess.RestoreToolSet(&state.Tools),
		held:         state.Held,
		beyond:       state.Beyond,
		mode:         state.Mode,
		reads:        tooldispatch.RestoreReads(state.Reads),
		repeats:      &loopguard.RepeatGuard{Failures: failures, Reads: make(map[string]bool, 4)},
		counts:       &tooldispatch.Ordinals{Seen: seen},
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
		world:        state.World.Clone(),
		toolMemories: cloneToolMemories(state.ToolMemories),
		scripts: turnscripts.Ledger{
			Continuable: maps.Clone(state.Continuable),
			Continuing:  maps.Clone(state.Continuing),
		},
		providerKind: state.ProviderKind,
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

// OpenTurn builds a turn: the tool set the actor may use, what earlier attempts
// of the run already learned, and the prompt. It reads permissions and the
// ledger, so it runs where those can be read: in process, or in an activity.
func (s *Service) OpenTurn(ctx context.Context, req *agentruntimeports.RunRequest) *Turn {
	definition := req.Definition
	budget := definition.MaxToolCalls
	if budget <= 0 {
		budget = agentdefinition.DefaultMaxToolCalls
	}

	runtimeContext := req.Context
	if len(runtimeContext.Tools) == 0 {
		runtimeContext.Tools = s.access.ToolSummaries(definition)
	}

	// Another agent's steps on a task this one handed it are never replayed:
	// the model only ever saw its own call and the answer it got back. The
	// thread's reader leaves them out already; this holds for any caller.
	history := turnhistory.ModelHistory(req.History)

	kit := s.openTurnTools(ctx, req, &runtimeContext, history)
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
	repeats := loopguard.NewRepeatGuard()
	counts := tooldispatch.NewOrdinals()
	// A delegate's turn shares the ledger of the turn that delegated, and is
	// opened once per task, so there is no earlier attempt of it to learn
	// from; the ledger's lessons are the other turn's.
	if req.Delegation == nil {
		s.dispatcher.SeedFromLedger(ctx, req, repeats)
	}

	messages, shortened := turnhistory.ReplayHistoryWithFiles(history, req.Proposals, req.FileTexts)
	now := turnNow(&runtimeContext)
	input := &turnInput{
		req:     req,
		history: history,
		command: kit.command,
		context: &runtimeContext,
		now:     now,
	}
	messages = append(messages, input.message())
	taint, opened := turntaint.Open(req, &runtimeContext, now)

	prompt := agentprompt.BuildSystemPromptParts(definition, &runtimeContext)

	return &Turn{
		s:            s,
		req:          req,
		budget:       budget,
		system:       prompt.Stable + prompt.Volatile,
		systemStable: len(prompt.Stable),
		messages:     messages,
		evidence: replycheck.FiguresOf(
			append(shortened, maputils.SortedValues(req.FileTexts)...),
		),
		tools:  kit.set,
		held:   kit.held,
		beyond: kit.beyond,
		mode:   kit.mode,
		reads: s.dispatcher.ConcurrentReads(
			append(slices.Clone(kit.held), toolaccess.BeyondNames(kit.beyond)...),
		),
		repeats:   repeats,
		counts:    counts,
		questions: askuser.AskedQuestions(history),
		decisions: make(map[pulid.ID]struct{}, 1),
		callIDs:   turnhistory.UsedCallIDs(req.History),
		result: &agentruntimeports.RunResult{
			Messages: []conversation.Message{{
				Role:      conversation.RoleUser,
				Content:   req.Input,
				CreatedAt: now,
			}},
			Taint:         taint,
			Fingerprint:   s.access.Fingerprint(definition, preferredProvider(req, definition), ""),
			UsedMemoryIDs: usedMemories,
		},
		delegates:    kit.delegates,
		directed:     delegation.DirectedTask(req),
		command:      kit.command,
		opened:       opened,
		external:     req.AfterExternalContent,
		world:        s.watchWorld(ctx, req, now),
		toolMemories: toolMemories,
	}
}

type turnTools struct {
	set       *toolaccess.ToolSet
	held      []string
	beyond    map[string]agent.AutonomyTier
	mode      toolaccess.ProfileMode
	delegates []agentturn.RuntimeDelegate
	command   agentextension.Command
}

// openTurnTools builds the tool set the actor may use on the turn and
// describes it in the runtime context.
func (s *Service) openTurnTools(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	runtimeContext *agentturn.RuntimeContext,
	history []conversation.Message,
) turnTools {
	definition := req.Definition
	active := s.access.ActiveExtensions(ctx, req.Actor)
	granted := active.Grants()
	runtimeContext.Tools = toolaccess.WithSummaries(
		runtimeContext.Tools,
		s.access.Summarize(definition, granted),
	)

	job := s.access.ProfileFor(ctx, req, granted)
	held := job.Held
	if !s.codeMode.Offered(req.CodeMode, s.access.ScriptsOffered(ctx, req)) {
		held, job.Beyond = s.access.WithoutScripts(definition, held, job.Beyond)
	}
	var delegates []agentturn.RuntimeDelegate
	if req.MayDelegate() {
		delegates = slices.Clone(runtimeContext.Delegates)
	} else {
		runtimeContext.Delegates = nil
	}
	tools := s.access.NewToolSet(ctx, &toolaccess.ToolSetRequest{
		Definition: definition,
		Held:       held,
		Beyond:     job.Beyond,
		Actor:      req.Actor,
		Input:      req.Input,
		Need:       toolaccess.FrameNeed(req),
		Query:      toolaccess.TurnQuery(req),
		History:    history,
		Unattended: req.Unattended,
		Delegated:  req.Delegation != nil,
		Publishes:  req.KeepsDocuments(),
		Delegates:  delegates,
		Decisions:  runtimetools.PendingDecisions(req.Proposals),
		Extensions: &active,
		Attachments: len(runtimeContext.Attachments) > 0 ||
			conversation.CarriesAttachments(history),
		Command: req.Command,
	})
	s.access.NotePreselected(ctx, req, tools)
	var command agentextension.Command
	if tools.Pinned {
		command = req.Command
	}
	s.describeToolSet(runtimeContext, req, tools, job.Mode)

	return turnTools{
		set:       tools,
		held:      held,
		beyond:    job.Beyond,
		mode:      job.Mode,
		delegates: delegates,
		command:   command,
	}
}

func (s *Service) describeToolSet(
	rc *agentturn.RuntimeContext,
	req *agentruntimeports.RunRequest,
	tools *toolaccess.ToolSet,
	mode toolaccess.ProfileMode,
) {
	rc.ToolsDisclosed = tools.Disclosed
	rc.ReachesBeyond = len(tools.Beyond) > 0
	if tools.Searchable() {
		rc.ToolMap = s.access.ToolMap(slices.Concat(tools.Allowed, tools.Beyond))
	}
	rc.Artifacts = tools.Offers(runtimetools.PublishArtifactName)
	rc.Inline = req.Inline
	rc.LooksUpOnly = mode == toolaccess.ProfileFanOut
	rc.DecisionRequests = tools.Offers(runtimetools.RequestDecisionName)
	// The prompt describes the set the person may use, not the agent's whole
	// configuration: a tool named there and refused when called reads as
	// the system refusing rather than the person lacking the right.
	rc.Tools = usableSummaries(rc.Tools, tools)
}

// turnInput is what the person's message to the model is built from.
type turnInput struct {
	req     *agentruntimeports.RunRequest
	history []conversation.Message
	command agentextension.Command
	context *agentturn.RuntimeContext
	now     int64
}

func (in *turnInput) message() llmports.Message {
	req, rc := in.req, in.context
	input := req.Input
	if in.command != "" {
		input = toolaccess.CommandInput(in.command, input)
	}
	var attached agentmessage.RenderedAttachments
	if req.Delegation == nil {
		attached = agentmessage.RenderAttachments(
			rc.Attachments, agentmessage.AttachmentTextBudget,
		)
		input = attached.Block + input
		input = proposalledger.OutOfViewDecisions(in.history, req.Proposals) + input
		input = agentmessage.DescribeAnchors(rc.Anchors, in.now, rc.Timezone) + input
	}
	if req.Definition.HasContextProvider(agentdefinition.ContextClock) {
		input = clockLine(in.now, rc.Timezone) + input
	}

	return llmports.Message{
		Role:    llmports.RoleUser,
		Content: input,
		Files:   turnhistory.MessageFiles(input, rc.Attachments, &attached),
	}
}

func (t *Turn) localNow(fx TurnEffects) time.Time {
	return time.Unix(fx.Now(), 0).In(timeutils.LoadLocation(t.req.Context.Timezone))
}

// turnNow is the moment the turn's context was built, which is the clock the
// prompt already reads ("Today is"), so the question's clock line, its records'
// ages and its taint marks agree with it, and a replay with a fixed clock sends
// the same request every run.
func turnNow(rc *agentturn.RuntimeContext) int64 {
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
func (t *Turn) completionRequest() *llmports.ChatCompletionRequest {
	req := t.req
	definition := req.Definition

	completion := &llmports.ChatCompletionRequest{
		TenantInfo:          req.Actor.TenantInfo(),
		System:              t.system,
		SystemStable:        t.systemStable,
		Messages:            t.messages,
		Tools:               t.tools.Specs,
		DeferredTools:       t.tools.Deferred,
		PreferredProviderID: preferredProvider(req, definition),
		PinPreferred:        req.PinProvider && !req.PreferredProviderID.IsNil(),
		Attribution:         toolaccess.TurnAttribution(req),
		MaxTokens:           t.cutOff.Budget,
		Reask:               t.takeReask(),
	}
	turnhistory.FitWindow(completion, t.window())

	return completion
}

// localEffects runs every effect in process, as a turn always did before it
// could run durably.
type localEffects struct {
	s        *Service
	ctx      context.Context
	emit     kernelports.AssistantStreamEmitter
	observer agentruntimeports.ToolObserver
}

func (*localEffects) Now() int64 { return timeutils.NowUnix() }

func (fx *localEffects) Complete(
	_ *Turn,
	req *llmports.ChatCompletionRequest,
) (modelstream.Reply, error) {
	return fx.s.streamer.StreamCompletion(fx.ctx, req, fx.Emit)
}

// KnowsTool reports whether name is a registered tool, read or write.
func (s *Service) KnowsTool(name string) bool { return s.access.ToolNamed(name) != nil }

func (fx *localEffects) Dispatch(t *Turn, call *toolcall.Call) toolcall.Outcome {
	return fx.s.DispatchStep(fx.ctx, t.req, call)
}

// DispatchReads runs the reads one after another in process. Its context may
// carry a transaction, which concurrent queries must not share.
func (fx *localEffects) DispatchReads(
	t *Turn,
	calls []toolcall.Call,
) []toolcall.Outcome {
	outcomes := make([]toolcall.Outcome, 0, len(calls))
	for idx := range calls {
		outcomes = append(outcomes, fx.Dispatch(t, &calls[idx]))
	}

	return outcomes
}

func (fx *localEffects) Find(t *Turn, arguments map[string]any) toolaccess.FindAnswer {
	answer := fx.s.access.ResolveFind(fx.ctx, t.tools, t.req.Actor, arguments)
	fx.s.access.NoteFound(fx.ctx, t.req, t.tools, arguments, &answer)

	return answer
}

func (fx *localEffects) Emit(event kernelports.StreamEvent) { fx.emit(event) }

func (fx *localEffects) Observe(
	t *Turn,
	call *llmports.ToolCall,
	outcome *toolcall.Outcome,
) toolcall.Outcome {
	result := outcome.Result()
	observed := runtimetools.Observe(fx.observer, call, &result, t.earlier(call.Name))

	return observed.Outcome()
}

func (*localEffects) NewCallID() string { return NewCallID() }

func (fx *localEffects) Interject(t *Turn, check *worldwatch.Check) worldwatch.Interjections {
	if check == nil || fx.s.changes == nil || t.req.Actor == nil {
		return worldwatch.Interjections{}
	}
	found, err := fx.s.changes.Since(fx.ctx, &realtimeports.RecordChangesRequest{
		OrganizationID: t.req.Actor.OrganizationID,
		BusinessUnitID: t.req.Actor.BusinessUnitID,
		Cursor:         check.Cursor,
		Records:        check.Records,
	})
	if err != nil {
		fx.s.logger.Debug("could not read changes to the turn's records", zap.Error(err))
		return worldwatch.Interjections{}
	}

	return worldwatch.Interjections{World: found}
}

// Delegate declines in process. A turn is offered delegate_task only when it
// is driven as a workflow, where the delegate's turn runs as activities of its
// own; this answers a model that names the tool anyway.
func (*localEffects) Delegate(_ *Turn, calls []delegation.Call) []delegation.Run {
	runs := make([]delegation.Run, 0, len(calls))
	for idx := range calls {
		runs = append(runs, delegation.Run{
			Declined: "handing a task to " + calls[idx].Delegate.Name + " is not available " +
				"here. Do what you can with your own tools.",
		})
	}

	return runs
}

// ObserveCall hands a finished call to observe and folds what it showed the
// person into what the model reads. It is what a durable tool activity runs
// after the call, where the call's raw result exists.
func (s *Service) ObserveCall(
	observe agentruntimeports.ToolObserver,
	call *llmports.ToolCall,
	outcome *toolcall.Outcome,
) toolcall.Outcome {
	result := outcome.Result()
	observed := runtimetools.Observe(observe, call, &result, nil)

	return observed.Outcome()
}

func (s *Service) ObserveDispatch(
	observe agentruntimeports.ToolObserver,
	call *toolcall.Call,
	outcome *toolcall.Outcome,
) toolcall.Outcome {
	result := outcome.Result()
	observed := runtimetools.Observe(observe, &call.Call, &result, call.Earlier)

	return observed.Outcome()
}

// PublishStep keeps a document a publish call asked for, and says what the
// model reads about it. A durable turn runs it in an activity: the document is
// read again from the call, because what the loop parsed stays in workflow
// code.
func (s *Service) PublishStep(
	observe agentruntimeports.ToolObserver,
	call *llmports.ToolCall,
) toolcall.Outcome {
	var step toolcall.Result
	switch call.Name {
	case runtimetools.RequestDecisionName:
		step = runtimetools.DecisionStepOutcome(call.Arguments)
	case runtimetools.ShowBoardName:
		step = runtimetools.BoardStepOutcome(call.Arguments)
	default:
		step = runtimetools.PublishOutcome(call.Arguments)
	}
	observed := runtimetools.Observe(observe, call, &step, nil)

	return observed.Outcome()
}

// DispatchStep runs one tool call the loop decided to make, claimed in the
// run's ledger when req carries one. It is what a durable tool activity calls:
// the loop decides in workflow code, and the call runs here, where a database
// and the tools can be reached.
func (s *Service) DispatchStep(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	call *toolcall.Call,
) toolcall.Outcome {
	outcome := s.dispatcher.GuardedDispatch(ctx, &tooldispatch.GuardedDispatchParams{
		Req:            req,
		Call:           call.Call,
		CompletionText: call.CompletionText,
		ProposedSoFar:  call.ProposedSoFar,
		MoneySpent:     call.MoneySpent,
		Beyond:         call.Beyond,
		Ordinal:        call.Ordinal,
		Taint:          call.Taint,
		AfterExternal:  call.AfterExternalContent,
		Gate: tooldispatch.ScriptGate{
			Model:      call.Model,
			Provider:   call.Provider,
			ProviderID: call.ProviderID,
			Position:   call.ScriptPosition,
			Continues:  call.ScriptContinues,
			Reach:      call.ScriptReach,
		},
	})
	outcome.Summary = toolsummary.Outcome(&call.Call, &outcome)

	return outcome.Outcome()
}

// FindFor answers find_tools for a turn held as data, and names the tools it
// made callable so the turn can load the same ones. An empty search names what
// exists beyond the agent only when the actor could use it, which takes a
// permission check, so this runs where permissions can be read.
func (s *Service) FindFor(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	state *toolaccess.ToolSetState,
	arguments map[string]any,
) toolaccess.FoundTools {
	set := toolaccess.RestoreToolSet(state)
	set.Self = req.Definition.ID
	set.Usable = func(name string) bool {
		return len(s.access.PermittedTools(ctx, req.Actor, []string{name})) == 1
	}

	set.Pending = nil
	answer := s.access.ResolveFind(ctx, set, req.Actor, arguments)
	s.access.NoteFound(ctx, req, set, arguments, &answer)
	loaded := set.TakeLoaded()

	return toolaccess.FoundTools{
		Content: answer.Content,
		Loaded:  loaded,
		Found:   answer.Found,
		HandOff: answer.HandOff,
	}
}

func (t *Turn) earlier(name string) []string {
	if !loopguard.IsFetch(name) {
		return nil
	}

	return slices.Clone(t.shown[name])
}

func (t *Turn) noteShown(call *llmports.ToolCall) {
	if !loopguard.IsFetch(call.Name) || call.ID == "" {
		return
	}
	if t.shown == nil {
		t.shown = make(map[string][]string, 1)
	}
	t.shown[call.Name] = append(t.shown[call.Name], call.ID)
}
