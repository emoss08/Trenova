package assistantstreamports

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/shared/pulid"
)

// AssistantArtifactEvent announces an artifact as a turn produces it, so the
// pane can open it while the reply is still arriving.
type AssistantArtifactEvent struct {
	ID               pulid.ID                 `json:"id"`
	Kind             assistantartifact.Kind   `json:"kind"`
	Status           assistantartifact.Status `json:"status"`
	Title            string                   `json:"title"`
	SourceToolCallID string                   `json:"sourceToolCallId,omitempty"`
	// Path is where a navigation artifact moves the app. It rides on the
	// event because the app follows it the moment it arrives, before the
	// artifact itself has been fetched.
	Path  string          `json:"path,omitempty"`
	Draft *pagedraft.Edit `json:"draft,omitempty"`
}

// AssistantArtifactRemovedEvent names an artifact the turn withdrew.
type AssistantArtifactRemovedEvent struct {
	ID pulid.ID `json:"id"`
}

const AssistantEventAccepted = "accepted"

const AssistantEventRefused = "refused"

const AssistantEventDelta = "delta"

const AssistantEventReasoning = "reasoning"

const AssistantEventMessage = "message"

const AssistantEventToolStarted = "tool_started"

const AssistantEventToolFinished = "tool_finished"

const AssistantEventRetrying = "retrying"

// AssistantEventReplyReplaced replaces the reply that streamed with the
// one the runtime corrected and recorded. Unlike a restart the model was
// not asked again, so a reader swaps the text without showing a retry.
const AssistantEventReplyReplaced = "reply_replaced"

const AssistantEventArtifact = "artifact"

// AssistantEventArtifactRemoved withdraws an artifact the turn announced:
// a record card a later read of the same tool folded into one table. The
// server has deleted it, so a reader drops it rather than keeping it
// beside the table that replaced it.
const AssistantEventArtifactRemoved = "artifact_removed"

const AssistantEventDone = "done"

// AssistantEventThread names the conversation a quick question was
// answered on, before the turn begins, so the reader can keep it even
// when the answer fails partway.
const AssistantEventThread = "thread"

// AssistantEventTurn names the turn producing a reply, before the reply
// begins. A reader keeps it so a dropped connection can be rejoined
// rather than restarted.
const AssistantEventTurn = "turn"

// AssistantEventError ends a turn that could not finish. It was written
// as a literal in the two handlers that emit it for as long as the stream
// was the handler's own; once the events travel through a relay, the name
// has to be one thing both ends agree on.
const AssistantEventError = "error"

// AssistantEventDelegateStarted and AssistantEventDelegateFinished open
// and close a task the turn's agent handed another agent. Everything the
// other agent does in between carries the same delegateCallId.
const AssistantEventDelegateStarted = "delegate_started"

const AssistantEventDelegateFinished = "delegate_finished"

// AssistantEventDelegateDelta, AssistantEventDelegateReasoning,
// AssistantEventDelegateRetrying and AssistantEventDelegateReplyReplaced
// are delta, reasoning, retrying and reply_replaced from the other agent.
// They are named apart because a reader applies the plain ones to the
// reply it is showing: another agent's words appended to it, its restart
// discarding it, or its corrected reply put in its place.
const AssistantEventDelegateDelta = "delegate_delta"

const AssistantEventDelegateReasoning = "delegate_reasoning"

const AssistantEventDelegateRetrying = "delegate_retrying"

const AssistantEventDelegateReplyReplaced = "delegate_reply_replaced"

// AssistantEventRunTainted says the turn read content written outside
// the organization, once for each new place it came from.
const AssistantEventRunTainted = "run_tainted"

const AssistantEventReplyRegrounded = "reply_regrounded"

// AssistantEventContext says how full the conversation's context is now,
// sent as a turn is saved so the composer's meter moves with the reply.
const AssistantEventContext = "context"

// AssistantEventCompactionStarted, AssistantEventCompactionFinished and
// AssistantEventCompactionCancelled follow a compaction: from the turn
// that set one off on its own, which names the compaction's turn so the
// reader can follow it, and on the compaction's own stream. Finished and
// cancelled each end that stream.
const AssistantEventCompactionStarted = "compaction_started"

const AssistantEventCompactionFinished = "compaction_finished"

const AssistantEventCompactionCancelled = "compaction_cancelled"

// AssistantEventMemoryUsed names every memory the turn has used so far:
// those its prompt carried, then each one agent_list_memories read back. Each
// event repeats the whole list, so a reader keeps the latest.
const AssistantEventMemoryUsed = "memory_used"

// AssistantEventMemorySaved says the turn kept a memory, or offered one
// for the person to accept when they asked to be asked first.
const AssistantEventMemorySaved = "memory_saved"

// AssistantEventSteered says the turn read something the person said
// while it worked, at the step it read it.
const AssistantEventSteered = "steered"

// AssistantEventWorldChanged says records the turn was working with
// changed under it, and that it was told so.
const AssistantEventWorldChanged = "world_changed"

// AssistantEventNextTurn names the turn the conversation's queue started
// once this one was saved, so the reader can follow it.
const AssistantEventNextTurn = "next_turn"

type AssistantSteeredEvent struct {
	ID       pulid.ID                     `json:"id"`
	Content  string                       `json:"content"`
	Mentions []agentpagecontext.EntityRef `json:"mentions,omitempty"`
}

type AssistantWorldChangedEvent struct {
	Changes []realtimeports.WatchedRecordChange `json:"changes"`
}

type AssistantNextTurnEvent struct {
	TurnID   pulid.ID `json:"turnId"`
	ThreadID pulid.ID `json:"threadId"`
	QueuedID pulid.ID `json:"queuedId"`
	Input    string   `json:"input"`
}

// AssistantContextEvent is how full a conversation's context is.
type AssistantContextEvent struct {
	ThreadID       pulid.ID                   `json:"threadId"`
	Usage          *conversation.ContextUsage `json:"usage"`
	AutoCompactOff bool                       `json:"autoCompactOff"`
}

// AssistantCompactionEvent is where a compaction stands. Before and After are
// the context use either side, in tokens: After is the estimate until the
// compaction finishes.
type AssistantCompactionEvent struct {
	TurnID   pulid.ID `json:"turnId"`
	ThreadID pulid.ID `json:"threadId"`
	Auto     bool     `json:"auto"`
	Before   int      `json:"before"`
	After    int      `json:"after"`
	// Message is the summary, once it is saved; Usage the context after it.
	Message *conversation.Message      `json:"message,omitempty"`
	Usage   *conversation.ContextUsage `json:"usage,omitempty"`
	// AutoCompactOff is set when cancelling a compaction that started on its
	// own turned compacting on its own off.
	AutoCompactOff bool `json:"autoCompactOff,omitempty"`
}

// AssistantMemoryUsedEvent is the memories a turn has used, in the order it
// used them.
type AssistantMemoryUsedEvent struct {
	IDs []pulid.ID `json:"ids"`
}

// AssistantMemorySavedEvent is a memory the turn saved, or offered to save.
type AssistantMemorySavedEvent = agentmemoryports.SavedMemory

type RegroundAction string

const RegroundRewrite RegroundAction = "rewrite"

const RegroundNote RegroundAction = "note"

// RegroundStripIDs is a final reply that wrote internal record ids out,
// with them taken out; RegroundPointToTable one that reprinted a table
// the turn kept beside the conversation, with the reprint replaced by a
// sentence pointing to it.
const RegroundStripIDs RegroundAction = "strip_ids"

const RegroundPointToTable RegroundAction = "point_to_table"

const RegroundRelinkArtifacts RegroundAction = "relink_artifacts"

const RegroundPlaceArtifacts RegroundAction = "place_artifacts"

type AssistantReplyRegroundedEvent struct {
	Action  RegroundAction `json:"action"`
	Figures []string       `json:"figures,omitempty"`
	Fields  []string       `json:"fields,omitempty"`
	// ArtifactID is the table a reprint was pointed to.
	ArtifactID     pulid.ID `json:"artifactId,omitempty"`
	Reason         string   `json:"reason"`
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// DelegateScope tags what another agent did on a task the turn's agent
// handed it, so a reader can nest it under the call that handed it over.
// Both fields are empty on the turn's own events.
type DelegateScope struct {
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// Empty reports the turn's own scope.
func (s DelegateScope) Empty() bool { return s.DelegateCallID == "" }

// Tag returns an event of another agent's turn as the reader of the turn
// that delegated sees it. ok is false for an event the reader must not see
// as it is: a refusal of the other agent's answer is reported when the task
// finishes, since on its own it reads as a refusal of the reply being shown.
func (s DelegateScope) Tag(event kernelports.StreamEvent) (kernelports.StreamEvent, bool) {
	if s.Empty() {
		return event, true
	}

	switch data := event.Data.(type) {
	case AssistantDeltaEvent:
		return kernelports.StreamEvent{
			Event: AssistantEventDelegateDelta,
			Data:  AssistantDelegateTextEvent{DelegateScope: s, Text: data.Text},
		}, true
	case AssistantReasoningEvent:
		return kernelports.StreamEvent{
			Event: AssistantEventDelegateReasoning,
			Data:  AssistantDelegateTextEvent{DelegateScope: s, Text: data.Text},
		}, true
	case AssistantRetryingEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: AssistantEventDelegateRetrying, Data: data}, true
	case AssistantReplyReplacedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: AssistantEventDelegateReplyReplaced, Data: data}, true
	case AssistantMessageEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: event.Event, Data: data}, true
	case AssistantToolStartedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: event.Event, Data: data}, true
	case AssistantToolFinishedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: event.Event, Data: data}, true
	case AssistantRunTaintedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: event.Event, Data: data}, true
	case AssistantReplyRegroundedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return kernelports.StreamEvent{Event: event.Event, Data: data}, true
	case AssistantRefusedEvent:
		return kernelports.StreamEvent{}, false
	// What another agent remembered on a task is its own; the card under the
	// reply is for what the conversation's agent kept.
	case AssistantMemoryUsedEvent, AssistantMemorySavedEvent:
		return kernelports.StreamEvent{}, false
	default:
		return event, true
	}
}

// AssistantDelegateTextEvent is a piece of another agent's reply or thinking.
type AssistantDelegateTextEvent struct {
	DelegateScope
	Text string `json:"text"`
}

// DelegateStatus is how a task handed to another agent ended.
type DelegateStatus = conversation.DelegateStatus

const DelegateStatusCompleted = conversation.DelegateStatusCompleted

const DelegateStatusExhausted = conversation.DelegateStatusExhausted

const DelegateStatusRefused = conversation.DelegateStatusRefused

const DelegateStatusDeclined = conversation.DelegateStatusDeclined

const DelegateStatusFailed = conversation.DelegateStatusFailed

const DelegateStatusStopped = conversation.DelegateStatusStopped

// AssistantDelegateStartedEvent says the turn's agent handed a task to
// another agent.
type AssistantDelegateStartedEvent struct {
	DelegateCallID string   `json:"delegateCallId"`
	AgentID        pulid.ID `json:"agentId"`
	AgentName      string   `json:"agentName"`
	Icon           string   `json:"icon,omitempty"`
	Accent         string   `json:"accent,omitempty"`
	Task           string   `json:"task"`
}

// AssistantDelegateFinishedEvent says how a task handed to another agent
// ended and what it came to. The same account is what the delegating agent
// reads as its tool result, and what the call's result message keeps.
type AssistantDelegateFinishedEvent = conversation.DelegateReport

// DelegateWrite is one write another agent made or proposed on a task.
type DelegateWrite = conversation.DelegateWrite

// DelegateDocument is something another agent kept beside the conversation.
type DelegateDocument = conversation.DelegateDocument

// AssistantTurnEvent names the turn a reply is being produced by.
type AssistantTurnEvent struct {
	TurnID   pulid.ID `json:"turnId"`
	ThreadID pulid.ID `json:"threadId"`
}

// TerminalAssistantEvent reports an event that ends a turn.
//
// A relay reading a turn's events needs to know when to stop without
// understanding any of them, and a reader who rejoins needs to know whether
// what they are watching is still running. Both ask this.
func TerminalAssistantEvent(event string) bool {
	switch event {
	case AssistantEventDone, AssistantEventError:
		return true
	default:
		return false
	}
}

// AssistantRetryingEvent says the model died partway through its reply and
// the turn is starting over, on another model when one is configured. Text
// streamed before it is discarded; what follows is the whole reply.
type AssistantRetryingEvent struct {
	Attempt  int    `json:"attempt"`
	Provider string `json:"provider,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Kind is "restart" for a reply starting over on another provider and
	// "busy" for a provider being asked again after a wait; WaitSeconds is
	// the wait, for the reader.
	Kind        kernelports.RetryKind `json:"kind,omitempty"`
	WaitSeconds int                   `json:"waitSeconds,omitempty"`
	// MaxAttempts is how many times the provider is asked in all.
	MaxAttempts int `json:"maxAttempts,omitempty"`
	// AgentID and DelegateCallID are set when it is another agent's reply
	// starting over, on a task this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantReplyReplacedEvent is the whole reply a turn recorded, sent in
// place of the one that streamed when the runtime corrected it afterwards:
// internal ids taken out, a reprinted table pointed to, a code block left
// out. The model was not asked again, so it is not a retry; the reader
// withdraws the streamed reply and shows Text instead. Reason says what was
// corrected, for the trajectory; the reply_regrounded events before it say
// what each pass changed.
type AssistantReplyReplacedEvent struct {
	Text   string `json:"text"`
	Reason string `json:"reason"`
	// AgentID and DelegateCallID are set when it is another agent's reply,
	// on a task this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantRunTaintedEvent is one new place outside content reached the turn
// from. From here on a write that leaves the organization waits for a person.
type AssistantRunTaintedEvent struct {
	Mark  agent.TaintMark `json:"mark"`
	Marks int             `json:"marks"`
	// AgentID and DelegateCallID are set when it is another agent's turn
	// that read it, on a task this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantAcceptedEvent says the question passed the scope guard and a model is
// being asked. It carries what the guard decided so a client can show the
// person's message immediately, before anything is saved.
type AssistantAcceptedEvent struct {
	Content       string `json:"content"`
	ScopeStage    string `json:"scopeStage"`
	ScopeCategory string `json:"scopeCategory"`
}

// AssistantRefusedEvent is the guard declining either the question or, later,
// the answer. Text already streamed before a late refusal must be discarded.
type AssistantRefusedEvent struct {
	Message  string `json:"message"`
	Stage    string `json:"stage"`
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// AssistantDeltaEvent is a piece of the reply the model is composing.
type AssistantDeltaEvent struct {
	Text string `json:"text"`
}

// AssistantReasoningEvent is a piece of the model's thinking, streamed before
// the reply so a heavy model's silence has something to show for it.
type AssistantReasoningEvent struct {
	Text string `json:"text"`
}

// AssistantMessageEvent closes the assistant message being streamed. It is
// sent when the model asked for tools, so the text before the tool traffic is
// kept as its own message; the final answer is closed by done instead.
type AssistantMessageEvent struct {
	Content   string                        `json:"content"`
	ToolCalls []conversation.ToolCallRecord `json:"toolCalls"`
	Model     string                        `json:"model"`
	// ProviderID is the provider that answered; FallbackFrom the one asked
	// first, when that one didn't. Truncated says the reply stopped partway.
	ProviderID   pulid.ID                       `json:"providerId,omitempty"`
	FallbackFrom *conversation.ProviderFallback `json:"fallbackFrom,omitempty"`
	Truncated    bool                           `json:"truncated,omitempty"`
	// AgentID and DelegateCallID are set on another agent's message, on a
	// task this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantToolStartedEvent says a tool is running with these arguments.
type AssistantToolStartedEvent struct {
	CallID    string                    `json:"callId"`
	Name      string                    `json:"name"`
	Arguments map[string]any            `json:"arguments"`
	Effect    agenttooltrust.ToolEffect `json:"effect,omitempty"`
	// AgentID and DelegateCallID are set on another agent's call, on a task
	// this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantToolFinishedEvent carries what the tool returned. Proposed means the
// tool was a write and became a proposal rather than running.
type AssistantToolFinishedEvent struct {
	CallID   string                    `json:"callId"`
	Name     string                    `json:"name"`
	Failed   bool                      `json:"failed"`
	Proposed bool                      `json:"proposed"`
	Content  string                    `json:"content"`
	Effect   agenttooltrust.ToolEffect `json:"effect,omitempty"`
	Summary  string                    `json:"summary,omitempty"`
	Verdict  string                    `json:"verdict,omitempty"`
	// HandOffAgents are the person's other agents a find_tools answer named
	// as holding what this agent could not load, the same ids the saved
	// tool message keeps, so a reader can offer the hand-off while the turn
	// is still running.
	HandOffAgents []pulid.ID `json:"handOffAgents,omitempty"`
	// Records are the records the call's result described, for a reader to
	// show as what the answer was built from.
	Records []agentpagecontext.EntityRef `json:"records,omitempty"`
	// AgentID and DelegateCallID are set on another agent's call, on a task
	// this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}
