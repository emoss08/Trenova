package services

import (
	"context"
	"github.com/emoss08/trenova/pkg/toolschema"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
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
}

// UpdateThreadRequest changes what a person may change about their own
// conversation: its title, whether it is pinned, and whether a quick
// question is kept as a listed conversation.
type UpdateThreadRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Title      *string
	Pinned     *bool
	// Keep promotes an Ask thread to the Desk so it is listed.
	Keep bool
}

// SendMessageRequest is one turn from a person.
type SendMessageRequest struct {
	ThreadID pulid.ID
	Content  string
	// Page is what the person was looking at when they asked, if the client
	// sent it. It is validated and stored with the user turn.
	Page       *agent.PageContext
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
}

// AskRequest is a quick question from anywhere in the application. It runs
// as an ordinary turn on a hidden thread bound to the organization's general
// assistant, so the answer can be kept as a conversation afterwards.
type AskRequest struct {
	Content    string
	Page       *agent.PageContext
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
	// AgentID and AgentName are the agent that proposed it: the
	// conversation's own, or another agent it handed a task to, whose
	// proposal it is and whose trust a decision on it teaches.
	AgentID   pulid.ID `json:"agentId,omitempty"`
	AgentName string   `json:"agentName,omitempty"`
}

// AssistantPlan is several of a turn's proposals as one decision, as the
// client shows it: approve or reject all of them, in order.
type AssistantPlan struct {
	ID             pulid.ID         `json:"id"`
	RunID          pulid.ID         `json:"runId"`
	Title          string           `json:"title"`
	Summary        string           `json:"summary"`
	Status         agent.PlanStatus `json:"status"`
	StepCount      int              `json:"stepCount"`
	CompletedSteps int              `json:"completedSteps"`
	FailedStep     *int             `json:"failedStep"`
	FailureError   string           `json:"failureError"`
	DecidedAt      *int64           `json:"decidedAt"`
	ExpiresAt      int64            `json:"expiresAt"`
	Hold           *ProposalHold    `json:"hold"`
	CreatedAt      int64            `json:"createdAt"`
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
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
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
	Path string `json:"path,omitempty"`
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
	AssistantEventDone         = "done"
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
	AssistantEventRunTainted = "run_tainted"
)

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
	case AssistantRefusedEvent:
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
// ThreadTranscript is a conversation rendered as Markdown, with the name the
// file should be saved under.
type ThreadTranscript struct {
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
	ListThreadProposals(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]AssistantProposal, error)
	ListThreadPlans(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]AssistantPlan, error)
	ListThreadArtifacts(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]AssistantArtifact, error)
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
	) (*ThreadTranscript, error)
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
	Trusted bool `json:"trusted"`
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
