package assistantports

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentstarter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	Page       *agentpagecontext.PageContext
	Surface    agentpagecontext.Surface
	TenantInfo pagination.TenantInfo
	// AttachmentDocumentIDs are files the person uploaded for this message.
	// Each must be a document they uploaded to this thread; anything else is
	// refused rather than read.
	AttachmentDocumentIDs []pulid.ID
	// Mentions are the records the person named from the composer.
	Mentions []agentpagecontext.EntityRef
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
	// DirectedAgentID hands Content to another agent the person may use, as
	// a task run from this conversation in place of its own agent's reply.
	// The conversation's agent need not list it: the person chose it.
	DirectedAgentID pulid.ID
	CodeMode        codemode.Override
}

// AskRequest is a quick question from anywhere in the application. It runs
// as an ordinary turn on a hidden thread bound to the organization's general
// assistant, so the answer can be kept as a conversation afterwards.
type AskRequest struct {
	Content    string
	Page       *agentpagecontext.PageContext
	Mentions   []agentpagecontext.EntityRef
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
	Proposals []assistantproposalports.AssistantProposal `json:"proposals"`
	// Artifacts is what the turn produced besides words, in the order it
	// produced them.
	Artifacts []assistantartifactports.AssistantArtifact `json:"artifacts"`
	// ProposalsUnrecorded says the turn proposed a write that could not be saved
	// for approval. The write still has not run, but there is nothing to approve,
	// so the client must not offer an approval the server cannot honor.
	ProposalsUnrecorded bool `json:"proposalsUnrecorded"`
}

// ListThreadMessagesRequest reads one page of a thread, newest first from
// the end or from just above BeforeSequence.
type ListThreadMessagesRequest struct {
	Thread conversationports.GetThreadRequest
	// Limit is the page size. Zero takes the default; more than the maximum
	// is clamped, not refused.
	Limit int
	// BeforeSequence, when set, pages upward: the messages numbered below it.
	BeforeSequence *int
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
	ID          pulid.ID               `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Template    agenttemplate.Template `json:"template"`
	Icon        string                 `json:"icon"`
	Accent      string                 `json:"accent"`
	SystemKey   string                 `json:"systemKey"`
	ToolNames   []string               `json:"toolNames"`
	Starters    []agentstarter.Starter `json:"starters"`
}

type PageThread struct {
	Thread *conversation.Thread `json:"thread"`
	Agent  PageAgent            `json:"agent"`
}

type PageAssistant interface {
	OpenPageThread(
		ctx context.Context,
		req *OpenPageThreadRequest,
		actor *permissionports.RequestActor,
	) (*PageThread, error)
}

type AssistantService interface {
	StartThread(
		ctx context.Context,
		req *StartThreadRequest,
		actor *permissionports.RequestActor,
	) (*conversation.Thread, error)
	ListThreads(
		ctx context.Context,
		req conversationports.ListThreadsRequest,
	) (*conversationports.ThreadPage, error)
	CountThreadShelves(
		ctx context.Context,
		req *conversationports.CountThreadShelvesRequest,
	) (*conversationports.ThreadShelfCounts, error)
	GetThread(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) (*conversation.Thread, error)
	MarkThreadRead(ctx context.Context, req conversationports.GetThreadRequest) error
	SearchMentions(
		ctx context.Context,
		actor permissionports.RequestActor,
		req MentionSearchRequest,
	) ([]MentionCandidate, error)
	SearchMentionPage(
		ctx context.Context,
		actor permissionports.RequestActor,
		req MentionPageRequest,
	) (*MentionPage, error)
	SearchDesk(
		ctx context.Context,
		actor permissionports.RequestActor,
		req DeskSearchRequest,
	) ([]DeskSearchResult, error)
	ThreadBudget(ctx context.Context, req conversationports.GetThreadRequest) (*ThreadBudget, error)
	CommandAvailability(
		ctx context.Context,
		req *CommandAvailabilityRequest,
		actor *permissionports.RequestActor,
	) (*CommandAvailability, error)
	RequestMore(ctx context.Context, req RequestMoreRequest) (*RequestMoreResult, error)
	UpdateThread(
		ctx context.Context,
		req *UpdateThreadRequest,
		actor *permissionports.RequestActor,
	) (*conversation.Thread, error)
	// SelectableProviders lists the models a person may pick for the assistant,
	// projected so no credential leaves the provider record.
	SelectableProviders(
		ctx context.Context,
		actor permissionports.RequestActor,
	) ([]AssistantProviderOption, error)
	ListMessages(
		ctx context.Context,
		req ListThreadMessagesRequest,
	) (*ThreadMessagesPage, error)
	DeleteThread(ctx context.Context, req conversationports.GetThreadRequest) error
	// StartAsk opens the hidden thread a quick question is answered on. The
	// answer is a turn like any other; the thread is listed only when the
	// person keeps it.
	StartAsk(
		ctx context.Context,
		req *AskRequest,
		actor *permissionports.RequestActor,
	) (*conversation.Thread, error)
}

type AssistantArtifactService interface {
	ListThreadArtifacts(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		opts assistantartifactports.ListArtifactsOptions,
	) (*assistantartifactports.AssistantArtifactPage, error)
	// ArtifactLineage reads every version of the lineage an artifact belongs
	// to, oldest first, and ArtifactBySlug the lineage a link names.
	ArtifactLineage(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
	) ([]assistantartifactports.AssistantArtifact, error)
	ArtifactBySlug(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		slug string,
	) ([]assistantartifactports.AssistantArtifact, error)
	SaveDocumentVersion(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		version assistantartifactports.SaveDocumentVersionRequest,
	) (*assistantartifactports.AssistantArtifact, error)
	RestoreDocumentVersion(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
	) (*assistantartifactports.AssistantArtifact, error)
	RunArtifactReport(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
	) (*assistantartifactports.AssistantArtifact, error)
	RewriteDocument(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		rewrite assistantartifactports.DocumentRewriteRequest,
	) (*assistantartifactports.DocumentRewriteSuggestion, error)
	// ExportArtifactCSV writes a table or report preview whole, read again
	// from its source rather than from the rows the pane holds.
	ExportArtifactCSV(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		sink io.Writer,
	) error
	ArtifactCSVName(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
	) (string, error)
	ArtifactRows(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		query *assistantartifactports.ArtifactRowsQuery,
	) (*assistantartifactports.ArtifactRows, error)
	// ExportDocument renders a document as a PDF or a Word file.
	ExportDocument(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		format string,
	) (*assistantartifactports.ArtifactFile, error)
	PinArtifact(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		artifactID pulid.ID,
		pinned bool,
	) (*assistantartifactports.AssistantArtifact, error)
}

type AssistantProposalService interface {
	ListThreadProposals(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) ([]assistantproposalports.AssistantProposal, error)
	// SaveProposalEdits keeps the values a person changed on one of the
	// conversation's pending proposals, for the approval to go with; an empty
	// set clears them.
	SaveProposalEdits(
		ctx context.Context,
		req conversationports.GetThreadRequest,
		proposalID pulid.ID,
		modifications map[string]any,
	) (*assistantproposalports.ProposalEdits, error)
	ListThreadPlans(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) ([]assistantproposalports.AssistantPlan, error)
}

type AssistantTranscriptService interface {
	// Transcript renders the whole conversation as a document a person can
	// read away from the panel and hand to someone else.
	Transcript(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) (*kernelports.TranscriptFile, error)
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

type CommandAvailabilityRequest struct {
	AgentID    pulid.ID
	TenantInfo pagination.TenantInfo
}

// CommandAvailability is the extension commands (/browser, /search) the
// person can send an agent, so the composer offers only those.
type CommandAvailability struct {
	Commands []agentextension.Command `json:"commands"`
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

const RequestMoreAccess = "access"

const RequestMoreAllowance = "allowance"

const RequestMoreBudget = "budget"

const RequestMoreDailyRuns = "daily_runs"

// RequestMoreRequest asks for what a person ran out of in a conversation.
type RequestMoreRequest struct {
	Thread conversationports.GetThreadRequest
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
