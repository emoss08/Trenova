package services

import (
	"context"
	"io"

	"github.com/emoss08/trenova/pkg/toolschema"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// StartThreadRequest opens a conversation with a configured agent.
type StartThreadRequest struct {
	AgentDefinitionID pulid.ID
	Title             string
	TenantInfo        pagination.TenantInfo
	// Origin is where the conversation begins; empty means the panel.
	Origin conversation.ThreadOrigin
	// SubjectType and SubjectID name the record the conversation is about
	// when it was opened from one.
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
	// HandedFromThreadID is the conversation a hand-off started this one
	// from, and Taint the outside content that conversation had read, which
	// the summary carried over may repeat.
	HandedFromThreadID pulid.ID
	Taint              *agent.RunTaint
	TaintedAt          *int64
	// PinnedFacts are the facts the conversation opens keeping in mind.
	PinnedFacts []string
}

// UpdateThreadRequest changes what a person may change about their own
// conversation: its title, whether it is pinned, and whether a quick
// question is kept as a listed conversation.
type UpdateThreadRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Title      *string
	Pinned     *bool
	// PinnedFacts, when set, replaces what the agents keep in mind for the
	// whole conversation. An empty list unpins them all.
	PinnedFacts *[]string
	// Keep promotes an Ask thread to the Desk so it is listed.
	Keep bool
	// AutoCompact turns the conversation's compacting itself on or off.
	AutoCompact *bool
}

// SendMessageRequest is one turn from a person.
type SendMessageRequest struct {
	ThreadID pulid.ID
	Content  string
	// Page is what the person was looking at when they asked, if the client
	// sent it. It is validated and stored with the user turn.
	Page       *agent.PageContext
	Surface    agent.Surface
	TenantInfo pagination.TenantInfo
	// AttachmentDocumentIDs are files the person uploaded for this message.
	// Each must be a document they uploaded to this thread; anything else is
	// refused rather than read.
	AttachmentDocumentIDs []pulid.ID
	// Mentions are the records the person named from the composer.
	Mentions []agent.EntityRef
	// PreferredProviderID is the model the person picked in the composer. It is
	// validated against the providers this organization has assigned to the
	// assistant before it is stored, so a stale or foreign id is dropped rather
	// than carried; the router would ignore it anyway, but silently keeping an
	// unusable preference on the thread would show the wrong model in the
	// picker forever.
	PreferredProviderID pulid.ID
	// ProviderChosen says the request carried a choice at all, empty meaning
	// "the organization's order". A request that says nothing about the
	// model leaves the thread's saved choice as it is.
	ProviderChosen bool
	// FollowUpProposalID asks for the turn that follows a decision on one of
	// this thread's proposals. The request then carries no content: the input
	// is a note the service writes from the decision, so the agent reports
	// what happened rather than leaving an approval unanswered.
	FollowUpProposalID pulid.ID
	// FollowUpPlanID is FollowUpProposalID for a plan: the turn that follows
	// its decision says how far its steps got.
	FollowUpPlanID pulid.ID
	// ResumeWaitID asks for the turn that picks up work the agent parked on a
	// wait. Like a follow-up, the request carries no content: the wait's note
	// is the input.
	ResumeWaitID pulid.ID
}

// AskRequest is a quick question from anywhere in the application. It runs
// as an ordinary turn on a hidden thread bound to the organization's general
// assistant, so the answer can be kept as a conversation afterwards.
type AskRequest struct {
	Content    string
	Page       *agent.PageContext
	Surface    agent.Surface
	Mentions   []agent.EntityRef
	TenantInfo pagination.TenantInfo
}

// SendMessageResult is what the turn produced and saved.
type SendMessageResult struct {
	Thread   *conversation.Thread   `json:"thread"`
	Messages []conversation.Message `json:"messages"`
	// Reply is the assistant's answer, or the refusal explaining why there is
	// none.
	Reply string `json:"reply"`
	// Refused separates "the assistant declined" from "the assistant answered",
	// which a client renders differently.
	Refused bool `json:"refused"`
	// Proposals are writes the agent asked for. Nothing here has run: each one is
	// a pending record waiting on a person's decision.
	Proposals []AssistantProposal `json:"proposals"`
	// Artifacts is what the turn produced besides words, in the order it
	// produced them.
	Artifacts []AssistantArtifact `json:"artifacts"`
	// ProposalsUnrecorded says the turn proposed a write that could not be saved
	// for approval. The write still has not run, but there is nothing to approve,
	// so the client must not offer an approval the server cannot honor.
	ProposalsUnrecorded bool `json:"proposalsUnrecorded"`
}

// AssistantProposal is a write the agent proposed during a conversation. It is a
// persisted agent proposal, so it can be approved or rejected through the same
// decision endpoint as any other agent's proposal.
type AssistantProposal struct {
	ID              pulid.ID             `json:"id"`
	RunID           pulid.ID             `json:"runId"`
	ToolName        string               `json:"toolName"`
	Arguments       map[string]any       `json:"arguments"`
	Rationale       string               `json:"rationale"`
	AutonomyTier    agent.AutonomyTier   `json:"autonomyTier"`
	Status          agent.ProposalStatus `json:"status"`
	SourceMessageID pulid.ID             `json:"sourceMessageId"`
	// Confidence is the model's own estimate, 0 to 1, shown so an approver can
	// weigh the rationale.
	Confidence float64 `json:"confidence"`
	// ExecutedAt and ExecutionError report what happened after approval. An
	// accepted proposal with neither set was approved but has not run yet.
	ExecutedAt     *int64 `json:"executedAt"`
	ExecutionError string `json:"executionError"`
	// ExecutionResult is what the run made, and for a write over many
	// records each one that did not go through.
	ExecutionResult *agent.ToolExecutionResult `json:"executionResult,omitempty"`
	// ExpiresAt is when a pending proposal stops being decidable. Zero means
	// it was made before expiry existed.
	ExpiresAt int64 `json:"expiresAt"`
	// Hold is set while a shadow switch keeps the proposal from being decided,
	// so the client can say so instead of offering an approval the server
	// will refuse. Nil means it can be decided.
	Hold *ProposalHold `json:"hold"`
	// PlanID and PlanStep are set when the proposal is one step of a plan
	// decided as a whole; the client groups such proposals under the plan.
	PlanID   pulid.ID `json:"planId"`
	PlanStep int      `json:"planStep"`
	// SimulatedAt and Simulation report a write previewed instead of made,
	// because the agent was in simulation when it was cleared.
	SimulatedAt *int64                `json:"simulatedAt"`
	Simulation  *agent.ToolSimulation `json:"simulation"`
	// Fields are the tool's parameters as a person may edit them before
	// approving, derived from the tool's schema. Set only while the proposal
	// is pending, since a decided one can no longer be changed.
	Fields []toolschema.Field `json:"fields"`
	// Modifications are the values the approver changed before approving,
	// keyed by parameter. Nil when it was approved as proposed.
	Modifications map[string]any `json:"modifications"`
	// PendingModifications are the values a person changed and has not yet
	// approved with, keyed by parameter, kept so the edit survives a reload.
	// Nil once the proposal is decided, or when nothing was changed.
	PendingModifications map[string]any `json:"pendingModifications"`
	// AgentID and AgentName are the agent that proposed it: the
	// conversation's own, or another agent it handed a task to, whose
	// proposal it is and whose trust a decision on it teaches.
	AgentID   pulid.ID `json:"agentId,omitempty"`
	AgentName string   `json:"agentName,omitempty"`

	CreatedAt       int64    `json:"createdAt"`
	DecidedAt       *int64   `json:"decidedAt"`
	DecidedByUserID pulid.ID `json:"decidedByUserId"`
	DecisionNote    string   `json:"decisionNote"`
}

// ProposalEdits is what a pending proposal holds as changed once a person
// saved their edits: only the values that differ from what the agent proposed.
type ProposalEdits struct {
	ProposalID           pulid.ID       `json:"proposalId"`
	PendingModifications map[string]any `json:"pendingModifications"`
}

// AssistantPlan is several of a turn's proposals as one decision, as the
// client shows it: approve or reject all of them, in order.
type AssistantPlan struct {
	ID              pulid.ID         `json:"id"`
	RunID           pulid.ID         `json:"runId"`
	Title           string           `json:"title"`
	Summary         string           `json:"summary"`
	Status          agent.PlanStatus `json:"status"`
	StepCount       int              `json:"stepCount"`
	CompletedSteps  int              `json:"completedSteps"`
	FailedStep      *int             `json:"failedStep"`
	FailureError    string           `json:"failureError"`
	DecidedAt       *int64           `json:"decidedAt"`
	DecidedByUserID pulid.ID         `json:"decidedByUserId"`
	ExpiresAt       int64            `json:"expiresAt"`
	Hold            *ProposalHold    `json:"hold"`
	CreatedAt       int64            `json:"createdAt"`
	// AgentID and AgentName are the agent whose proposals the plan groups.
	AgentID   pulid.ID `json:"agentId,omitempty"`
	AgentName string   `json:"agentName,omitempty"`
}

// AssistantArtifact is what a turn produced besides words, as the Desk shows
// it beside the conversation.
type AssistantArtifact struct {
	ID         pulid.ID                 `json:"id"`
	ThreadID   pulid.ID                 `json:"threadId"`
	MessageID  pulid.ID                 `json:"messageId"`
	RunID      pulid.ID                 `json:"runId"`
	ProposalID pulid.ID                 `json:"proposalId"`
	PlanID     pulid.ID                 `json:"planId"`
	Kind       assistantartifact.Kind   `json:"kind"`
	Status     assistantartifact.Status `json:"status"`
	Title      string                   `json:"title"`
	Payload    map[string]any           `json:"payload"`
	// SourceToolCallID ties the artifact to the tool call that produced it,
	// so the transcript can point at it.
	SourceToolCallID string `json:"sourceToolCallId"`
	Pinned           bool   `json:"pinned"`
	// LineageID names the first artifact of the lineage this one is a later
	// version of, and LineageSeq its version number; empty and 1 for the first.
	LineageID  pulid.ID `json:"lineageId"`
	LineageSeq int      `json:"lineageSeq"`
	// Slug names the lineage in a link; the same for every version.
	Slug string `json:"slug"`
	// Turn is the question the person asked in the turn that made the
	// artifact, so an artifact from an earlier day says which turn it was.
	Turn      string `json:"turn,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ListArtifactsOptions is one page of a conversation's artifacts and what
// narrows it, all decided on the server so a long conversation pages rather
// than truncates.
type ListArtifactsOptions struct {
	Limit      int
	Cursor     string
	Query      string
	Family     assistantartifact.Family
	PinnedOnly bool
}

// AssistantArtifactPage is one page of lineages, every version of each, how
// many lineages match in all and how they split by family.
type AssistantArtifactPage struct {
	Results    []AssistantArtifact         `json:"results"`
	Total      int                         `json:"total"`
	NextCursor string                      `json:"nextCursor,omitempty"`
	Counts     repositories.ArtifactCounts `json:"counts"`
}

// DocumentRewriteMode is how a person asked a passage to change.
type DocumentRewriteMode string

const (
	DocumentRewriteShorter DocumentRewriteMode = "shorter"
	DocumentRewritePlainer DocumentRewriteMode = "plain"
	DocumentRewriteAsk     DocumentRewriteMode = "ask"
)

// DocumentRewriteRequest is a passage of a document to rewrite and how. The
// suggestion is not saved; accepting it saves a new version.
type DocumentRewriteRequest struct {
	Text   string              `json:"text"`
	Mode   DocumentRewriteMode `json:"mode"`
	Prompt string              `json:"prompt"`
}

type DocumentRewriteSuggestion struct {
	Text string `json:"text"`
}

// SaveDocumentVersionRequest is a person's edit of a document, kept as its
// next version.
type SaveDocumentVersionRequest struct {
	Body string `json:"body"`
	Note string `json:"note"`
}

// ArtifactFile is an artifact rendered as a file to download.
type ArtifactFile struct {
	FileName    string
	ContentType string
	Body        []byte
}

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

// ProposalHold names the switch holding a proposal and, when it is an agent's
// own, which agent.
type ProposalHold struct {
	Reason    string `json:"reason"`
	AgentName string `json:"agentName"`
}

// Names of the events a streamed turn emits, in the order a client should
// expect them: accepted or refused first, then any number of delta, message,
// tool_started and tool_finished, then done. A refusal can also arrive late,
// when the answer rather than the question was declined.
const (
	AssistantEventAccepted     = "accepted"
	AssistantEventRefused      = "refused"
	AssistantEventDelta        = "delta"
	AssistantEventReasoning    = "reasoning"
	AssistantEventMessage      = "message"
	AssistantEventToolStarted  = "tool_started"
	AssistantEventToolFinished = "tool_finished"
	AssistantEventRetrying     = "retrying"
	AssistantEventArtifact     = "artifact"
	// AssistantEventArtifactRemoved withdraws an artifact the turn announced:
	// a record card a later read of the same tool folded into one table. The
	// server has deleted it, so a reader drops it rather than keeping it
	// beside the table that replaced it.
	AssistantEventArtifactRemoved = "artifact_removed"
	AssistantEventDone            = "done"
	// AssistantEventThread names the conversation a quick question was
	// answered on, before the turn begins, so the reader can keep it even
	// when the answer fails partway.
	AssistantEventThread = "thread"
	// AssistantEventTurn names the turn producing a reply, before the reply
	// begins. A reader keeps it so a dropped connection can be rejoined
	// rather than restarted.
	AssistantEventTurn = "turn"
	// AssistantEventError ends a turn that could not finish. It was written
	// as a literal in the two handlers that emit it for as long as the stream
	// was the handler's own; once the events travel through a relay, the name
	// has to be one thing both ends agree on.
	AssistantEventError = "error"
	// AssistantEventDelegateStarted and AssistantEventDelegateFinished open
	// and close a task the turn's agent handed another agent. Everything the
	// other agent does in between carries the same delegateCallId.
	AssistantEventDelegateStarted  = "delegate_started"
	AssistantEventDelegateFinished = "delegate_finished"
	// AssistantEventDelegateDelta, AssistantEventDelegateReasoning and
	// AssistantEventDelegateRetrying are delta, reasoning and retrying from
	// the other agent. They are named apart because a reader applies the
	// plain ones to the reply it is showing: another agent's words appended
	// to it, or its restart discarding it.
	AssistantEventDelegateDelta     = "delegate_delta"
	AssistantEventDelegateReasoning = "delegate_reasoning"
	AssistantEventDelegateRetrying  = "delegate_retrying"
	// AssistantEventRunTainted says the turn read content written outside
	// the organization, once for each new place it came from.
	AssistantEventRunTainted      = "run_tainted"
	AssistantEventReplyRegrounded = "reply_regrounded"
	// AssistantEventContext says how full the conversation's context is now,
	// sent as a turn is saved so the composer's meter moves with the reply.
	AssistantEventContext = "context"
	// AssistantEventCompactionStarted, AssistantEventCompactionFinished and
	// AssistantEventCompactionCancelled follow a compaction: from the turn
	// that set one off on its own, which names the compaction's turn so the
	// reader can follow it, and on the compaction's own stream. Finished and
	// cancelled each end that stream.
	AssistantEventCompactionStarted   = "compaction_started"
	AssistantEventCompactionFinished  = "compaction_finished"
	AssistantEventCompactionCancelled = "compaction_cancelled"
	// AssistantEventMemoryUsed names every memory the turn has used so far:
	// those its prompt carried, then each one recall_memory read back. Each
	// event repeats the whole list, so a reader keeps the latest.
	AssistantEventMemoryUsed = "memory_used"
	// AssistantEventMemorySaved says the turn kept a memory, or offered one
	// for the person to accept when they asked to be asked first.
	AssistantEventMemorySaved = "memory_saved"
	// AssistantEventSteered says the turn read something the person said
	// while it worked, at the step it read it.
	AssistantEventSteered = "steered"
	// AssistantEventWorldChanged says records the turn was working with
	// changed under it, and that it was told so.
	AssistantEventWorldChanged = "world_changed"
	// AssistantEventNextTurn names the turn the conversation's queue started
	// once this one was saved, so the reader can follow it.
	AssistantEventNextTurn = "next_turn"
)

type AssistantSteeredEvent struct {
	ID       pulid.ID          `json:"id"`
	Content  string            `json:"content"`
	Mentions []agent.EntityRef `json:"mentions,omitempty"`
}

type AssistantWorldChangedEvent struct {
	Changes []WatchedRecordChange `json:"changes"`
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
type AssistantMemorySavedEvent = SavedMemory

type RegroundAction string

const (
	RegroundRewrite RegroundAction = "rewrite"
	RegroundNote    RegroundAction = "note"
	// RegroundStripIDs is a final reply that wrote internal record ids out,
	// with them taken out; RegroundPointToTable one that reprinted a table
	// the turn kept beside the conversation, with the reprint replaced by a
	// sentence pointing to it.
	RegroundStripIDs     RegroundAction = "strip_ids"
	RegroundPointToTable RegroundAction = "point_to_table"
)

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
func (s DelegateScope) Tag(event StreamEvent) (StreamEvent, bool) {
	if s.Empty() {
		return event, true
	}

	switch data := event.Data.(type) {
	case AssistantDeltaEvent:
		return StreamEvent{
			Event: AssistantEventDelegateDelta,
			Data:  AssistantDelegateTextEvent{DelegateScope: s, Text: data.Text},
		}, true
	case AssistantReasoningEvent:
		return StreamEvent{
			Event: AssistantEventDelegateReasoning,
			Data:  AssistantDelegateTextEvent{DelegateScope: s, Text: data.Text},
		}, true
	case AssistantRetryingEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: AssistantEventDelegateRetrying, Data: data}, true
	case AssistantMessageEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: event.Event, Data: data}, true
	case AssistantToolStartedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: event.Event, Data: data}, true
	case AssistantToolFinishedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: event.Event, Data: data}, true
	case AssistantRunTaintedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: event.Event, Data: data}, true
	case AssistantReplyRegroundedEvent:
		data.AgentID, data.DelegateCallID = s.AgentID, s.DelegateCallID
		return StreamEvent{Event: event.Event, Data: data}, true
	case AssistantRefusedEvent:
		return StreamEvent{}, false
	// What another agent remembered on a task is its own; the card under the
	// reply is for what the conversation's agent kept.
	case AssistantMemoryUsedEvent, AssistantMemorySavedEvent:
		return StreamEvent{}, false
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

const (
	DelegateStatusCompleted = conversation.DelegateStatusCompleted
	DelegateStatusExhausted = conversation.DelegateStatusExhausted
	DelegateStatusRefused   = conversation.DelegateStatusRefused
	DelegateStatusDeclined  = conversation.DelegateStatusDeclined
	DelegateStatusFailed    = conversation.DelegateStatusFailed
	DelegateStatusStopped   = conversation.DelegateStatusStopped
)

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
	Kind        RetryKind `json:"kind,omitempty"`
	WaitSeconds int       `json:"waitSeconds,omitempty"`
	// MaxAttempts is how many times the provider is asked in all.
	MaxAttempts int `json:"maxAttempts,omitempty"`
	// AgentID and DelegateCallID are set when it is another agent's reply
	// starting over, on a task this turn's agent handed it.
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
	CallID    string           `json:"callId"`
	Name      string           `json:"name"`
	Arguments map[string]any   `json:"arguments"`
	Effect    agent.ToolEffect `json:"effect,omitempty"`
	// Why is the model's reason for the step, when it gave one.
	Why *conversation.StepRationale `json:"why,omitempty"`
	// AgentID and DelegateCallID are set on another agent's call, on a task
	// this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantToolFinishedEvent carries what the tool returned. Proposed means the
// tool was a write and became a proposal rather than running.
type AssistantToolFinishedEvent struct {
	CallID   string           `json:"callId"`
	Name     string           `json:"name"`
	Failed   bool             `json:"failed"`
	Proposed bool             `json:"proposed"`
	Content  string           `json:"content"`
	Effect   agent.ToolEffect `json:"effect,omitempty"`
	Summary  string           `json:"summary,omitempty"`
	Verdict  string           `json:"verdict,omitempty"`
	// AgentID and DelegateCallID are set on another agent's call, on a task
	// this turn's agent handed it.
	AgentID        pulid.ID `json:"agentId,omitempty"`
	DelegateCallID string   `json:"delegateCallId,omitempty"`
}

// AssistantStreamEmitter receives the events of one turn as they happen. It is
// called from the turn's own goroutine, in order, and never after the turn
// returns.
type AssistantStreamEmitter func(event StreamEvent)

// ListThreadMessagesRequest reads one page of a thread, newest first from
// the end or from just above BeforeSequence.
type ListThreadMessagesRequest struct {
	Thread repositories.GetThreadRequest
	// Limit is the page size. Zero takes the default; more than the maximum
	// is clamped, not refused.
	Limit int
	// BeforeSequence, when set, pages upward: the messages numbered below it.
	BeforeSequence *int
}

// ThreadMessagesPage is one page of a thread in reading order, with what a
// client needs to ask for the page above it and to say how long the
// conversation has become.
type TranscriptFile struct {
	FileName string
	Body     string
}

type ThreadMessagesPage struct {
	Results []conversation.Message `json:"results"`
	// HasMore says there are messages above the first one here.
	HasMore bool `json:"hasMore"`
	// Total is the thread's whole length, not the page's.
	Total int `json:"total"`
	// Limit is how many messages a thread may hold before it must be
	// continued in a new one, so the client can say so before the wall.
	Limit int `json:"limit"`
}

type OpenPageThreadRequest struct {
	TenantInfo  pagination.TenantInfo
	Origin      conversation.ThreadOrigin
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
}

type PageAgent struct {
	ID          pulid.ID                  `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Template    agentdefinition.Template  `json:"template"`
	Icon        string                    `json:"icon"`
	Accent      string                    `json:"accent"`
	SystemKey   string                    `json:"systemKey"`
	ToolNames   []string                  `json:"toolNames"`
	Starters    []agentdefinition.Starter `json:"starters"`
}

type PageThread struct {
	Thread *conversation.Thread `json:"thread"`
	Agent  PageAgent            `json:"agent"`
}

type PageAssistant interface {
	OpenPageThread(
		ctx context.Context,
		req *OpenPageThreadRequest,
		actor *RequestActor,
	) (*PageThread, error)
}

type AssistantService interface {
	StartThread(
		ctx context.Context,
		req *StartThreadRequest,
		actor *RequestActor,
	) (*conversation.Thread, error)
	ListThreads(
		ctx context.Context,
		req repositories.ListThreadsRequest,
	) (*pagination.ListResult[*conversation.Thread], error)
	GetThread(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) (*conversation.Thread, error)
	MarkThreadRead(ctx context.Context, req repositories.GetThreadRequest) error
	SearchMentions(
		ctx context.Context,
		actor RequestActor,
		req MentionSearchRequest,
	) ([]MentionCandidate, error)
	SearchMentionPage(
		ctx context.Context,
		actor RequestActor,
		req MentionPageRequest,
	) (*MentionPage, error)
	SearchDesk(
		ctx context.Context,
		actor RequestActor,
		req DeskSearchRequest,
	) ([]DeskSearchResult, error)
	ThreadBudget(ctx context.Context, req repositories.GetThreadRequest) (*ThreadBudget, error)
	RequestMore(ctx context.Context, req RequestMoreRequest) (*RequestMoreResult, error)
	ListThreadProposals(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]AssistantProposal, error)
	// SaveProposalEdits keeps the values a person changed on one of the
	// conversation's pending proposals, for the approval to go with; an empty
	// set clears them.
	SaveProposalEdits(
		ctx context.Context,
		req repositories.GetThreadRequest,
		proposalID pulid.ID,
		modifications map[string]any,
	) (*ProposalEdits, error)
	ListThreadPlans(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]AssistantPlan, error)
	ListThreadArtifacts(
		ctx context.Context,
		req repositories.GetThreadRequest,
		opts ListArtifactsOptions,
	) (*AssistantArtifactPage, error)
	// ArtifactLineage reads every version of the lineage an artifact belongs
	// to, oldest first, and ArtifactBySlug the lineage a link names.
	ArtifactLineage(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
	) ([]AssistantArtifact, error)
	ArtifactBySlug(
		ctx context.Context,
		req repositories.GetThreadRequest,
		slug string,
	) ([]AssistantArtifact, error)
	SaveDocumentVersion(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
		version SaveDocumentVersionRequest,
	) (*AssistantArtifact, error)
	RestoreDocumentVersion(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
	) (*AssistantArtifact, error)
	RewriteDocument(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
		rewrite DocumentRewriteRequest,
	) (*DocumentRewriteSuggestion, error)
	// ExportArtifactCSV writes a table or report preview whole, read again
	// from its source rather than from the rows the pane holds.
	ExportArtifactCSV(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
		sink io.Writer,
	) error
	ArtifactCSVName(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
	) (string, error)
	// ExportDocument renders a document as a PDF or a Word file.
	ExportDocument(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
		format string,
	) (*ArtifactFile, error)
	PinArtifact(
		ctx context.Context,
		req repositories.GetThreadRequest,
		artifactID pulid.ID,
		pinned bool,
	) (*AssistantArtifact, error)
	UpdateThread(
		ctx context.Context,
		req *UpdateThreadRequest,
		actor *RequestActor,
	) (*conversation.Thread, error)
	// SelectableProviders lists the models a person may pick for the assistant,
	// projected so no credential leaves the provider record.
	SelectableProviders(
		ctx context.Context,
		actor RequestActor,
	) ([]AssistantProviderOption, error)
	ListMessages(
		ctx context.Context,
		req ListThreadMessagesRequest,
	) (*ThreadMessagesPage, error)
	// Transcript renders the whole conversation as a document a person can
	// read away from the panel and hand to someone else.
	Transcript(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) (*TranscriptFile, error)
	DeleteThread(ctx context.Context, req repositories.GetThreadRequest) error
	// StartAsk opens the hidden thread a quick question is answered on. The
	// answer is a turn like any other; the thread is listed only when the
	// person keeps it.
	StartAsk(
		ctx context.Context,
		req *AskRequest,
		actor *RequestActor,
	) (*conversation.Thread, error)
}

// AssistantProviderOption is one entry in the model picker: enough to render a
// choice, and none of the provider's credentials.
type AssistantProviderOption struct {
	ID pulid.ID `json:"id"`
	// Name is what the administrator called this endpoint.
	Name string `json:"name"`
	// Kind is the vendor, which the client renders as a mark.
	Kind string `json:"kind"`
	// Model is the identifier this endpoint is pinned to, and the label a
	// person actually recognises.
	Model string `json:"model"`
	// Trusted is shown because it decides whether this choice can serve work
	// that reaches financial records.
	Trusted     bool   `json:"trusted"`
	Vendor      string `json:"vendor"`
	Reasoning   string `json:"reasoning"`
	Unavailable bool   `json:"unavailable"`
}

type MentionSearchRequest struct {
	Query string
	Kind  string
}

// MentionPageRequest asks for one page of the records of one kind, for a
// list that scrolls through them rather than the @ search's first few.
type MentionPageRequest struct {
	Query  string
	Kind   string
	Offset int
	Limit  int
}

// MentionPage is a page of records and whether another follows it.
type MentionPage struct {
	Results []MentionCandidate `json:"results"`
	HasMore bool               `json:"hasMore"`
}

// ThreadBudget is where a conversation's agent stands against its monthly
// budget and daily run cap. LimitUSD is empty when the agent has no budget.
type ThreadBudget struct {
	AgentName     string  `json:"agentName"`
	SpentUSD      string  `json:"spentUsd"`
	LimitUSD      string  `json:"limitUsd"`
	Share         float64 `json:"share"`
	Near          bool    `json:"near"`
	MonthStart    int64   `json:"monthStart"`
	ResetsAt      int64   `json:"resetsAt"`
	RunsToday     int     `json:"runsToday"`
	DailyRunLimit int     `json:"dailyRunLimit"`
	// BudgetUsed and DailyUsed say a cap is spent, so the composer locks
	// before a question is sent rather than after it is refused.
	BudgetUsed bool `json:"budgetUsed"`
	DailyUsed  bool `json:"dailyUsed"`
	// DayResetsAt is when the daily run cap clears.
	DayResetsAt int64 `json:"dayResetsAt"`
	// DisabledBy and DisabledAt say who turned the agent off and when.
	DisabledBy string `json:"disabledBy,omitempty"`
	DisabledAt int64  `json:"disabledAt,omitempty"`
	// Person is the asker's own monthly allowance, nil when unlimited.
	Person *PersonAllowance `json:"person"`
}

// The things a person can ask AI Control for from a conversation.
const (
	RequestMoreAccess    = "access"
	RequestMoreAllowance = "allowance"
	RequestMoreBudget    = "budget"
	RequestMoreDailyRuns = "daily_runs"
)

// RequestMoreRequest asks for what a person ran out of in a conversation.
type RequestMoreRequest struct {
	Thread repositories.GetThreadRequest
	Kind   string
}

// RequestMoreResult says how many people were asked.
type RequestMoreResult struct {
	Sent int `json:"sent"`
}

// PersonAllowance is how many questions one person has asked this month
// against how many their organization allows.
type PersonAllowance struct {
	Used     int   `json:"used"`
	Limit    int   `json:"limit"`
	ResetsAt int64 `json:"resetsAt"`
}

// DeskSearchRequest searches a person's own Desk. Kind is chat, msg, art,
// dec or empty for all of them; an empty query lists what is recent.
type DeskSearchRequest struct {
	Query string
	Kind  string
}

// DeskSearchResult is one conversation, message, artifact or decision. For a
// message Title is the part of it around what matched.
type DeskSearchResult struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	ThreadID     pulid.ID `json:"threadId"`
	AgentID      pulid.ID `json:"agentId"`
	Title        string   `json:"title"`
	ThreadTitle  string   `json:"threadTitle"`
	ArtifactKind string   `json:"artifactKind"`
	Status       string   `json:"status"`
	At           int64    `json:"at"`
}

type MentionCandidate struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Subtitle string `json:"subtitle"`
}

// DecisionFollowUpRequest names a decision on a proposal or a plan an agent
// raised. RunID is the run that raised it, which says whether it came from a
// conversation.
type DecisionFollowUpRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	ProposalID pulid.ID
	PlanID     pulid.ID
}

// DecisionFollowUps has the agent report what came of a decision, in the
// conversation that raised it, wherever the decision was made: the card in
// the thread, the Desk's decisions, AI Control or a plan. It never fails the
// decision. A follow-up that cannot start because the conversation is busy
// is not lost: the turn in the way resumes it when it ends.
type DecisionFollowUps interface {
	FollowUp(ctx context.Context, req DecisionFollowUpRequest)
}

// ResumeFollowUpsRequest names a conversation whose turn just ended.
type ResumeFollowUpsRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadID   pulid.ID
}

// DecisionFollowUpResumer starts the follow-up a busy conversation could not
// take when its decision was made. A conversation answers one turn at a time,
// so a decision made while it was answering, the second of two cards approved
// in quick succession most often, used to be reported nowhere: no Decision
// entry, and a reply already under way that still read the proposal as
// waiting. It never fails the turn that ended.
type DecisionFollowUpResumer interface {
	ResumeFollowUps(ctx context.Context, req ResumeFollowUpsRequest)
}

type SettleQueueRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadID   pulid.ID
	UserID     pulid.ID
	Read       []pulid.ID
	Dispatch   bool
}

type QueuedTurn struct {
	TurnID   pulid.ID `json:"turnId"`
	QueuedID pulid.ID `json:"queuedId"`
	Input    string   `json:"input"`
}

type AssistantQueueSettler interface {
	SettleQueue(ctx context.Context, req *SettleQueueRequest) *QueuedTurn
}
