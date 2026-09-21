package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
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
}

// SendMessageRequest is one turn from a person.
type SendMessageRequest struct {
	ThreadID pulid.ID
	Content  string
	// Page is what the person was looking at when they asked, if the client
	// sent it. It is validated and stored with the user turn.
	Page       *agent.PageContext
	TenantInfo pagination.TenantInfo
	// PreferredProviderID is the model the person picked in the composer. It is
	// validated against the providers this organization has assigned to the
	// assistant before it is stored, so a stale or foreign id is dropped rather
	// than carried; the router would ignore it anyway, but silently keeping an
	// unusable preference on the thread would show the wrong model in the
	// picker forever.
	PreferredProviderID pulid.ID
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
	AssistantEventDone         = "done"
)

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
}

// AssistantToolStartedEvent says a tool is running with these arguments.
type AssistantToolStartedEvent struct {
	CallID    string         `json:"callId"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// AssistantToolFinishedEvent carries what the tool returned. Proposed means the
// tool was a write and became a proposal rather than running.
type AssistantToolFinishedEvent struct {
	CallID   string `json:"callId"`
	Name     string `json:"name"`
	Failed   bool   `json:"failed"`
	Proposed bool   `json:"proposed"`
	Content  string `json:"content"`
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
	// SendMessage runs a guarded turn and persists it.
	SendMessage(
		ctx context.Context,
		req *SendMessageRequest,
		actor *RequestActor,
	) (*SendMessageResult, error)
	// SendMessageStream is SendMessage reported live: the guard's verdict, the
	// reply as it is written, and each tool as it runs, before the saved result.
	SendMessageStream(
		ctx context.Context,
		req *SendMessageRequest,
		actor *RequestActor,
		emit AssistantStreamEmitter,
	) (*SendMessageResult, error)
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
