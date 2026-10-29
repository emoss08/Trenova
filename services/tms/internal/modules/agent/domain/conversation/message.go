package conversation

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// Message is one turn in a thread.
//
// Tool traffic is persisted alongside the prose rather than discarded after the
// loop, because "which records did the assistant actually read before saying
// that" is the question anyone reviewing an answer will ask. The scope columns
// record the guard's verdict for the same reason: a refusal is evidence about how
// the assistant is being used, and it is only useful if it survives the request.
type Message struct {
	keel.Table `keel:"assistant_messages"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:amsg_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ThreadID pulid.ID `json:"threadId" keel:"thread_id"`
	// Sequence orders messages within a thread without relying on a timestamp,
	// which can collide inside a single fast tool loop.
	Sequence int  `json:"sequence" keel:"sequence"`
	Role     Role `json:"role"     keel:"role"`
	// TurnID is the turn that saved the message, which is where a chapter
	// that turn opened starts. Empty on cards written outside a turn.
	TurnID pulid.ID `json:"turnId,omitempty" keel:"turn_id,nullzero"`
	// Kind is Message for everything but the note that starts the turn after
	// a decision and the steps of another agent the turn handed work to; see
	// MessageKind.
	Kind MessageKind `json:"kind" keel:"kind"`
	// AgentDefinitionID and DelegateCallID mark a step another agent took on
	// a task this conversation's agent handed it: which agent, and the
	// delegate_task call it answers. Both are empty on the conversation's own
	// messages. AgentName, AgentIcon and AgentAccent are the agent's name
	// and mark as the thread is served.
	AgentDefinitionID pulid.ID `json:"agentId,omitempty"        keel:"agent_definition_id,nullzero"`
	DelegateCallID    string   `json:"delegateCallId,omitempty" keel:"delegate_call_id,nullzero"`
	AgentName         string   `json:"agentName,omitempty"`
	AgentIcon         string   `json:"agentIcon,omitempty"`
	AgentAccent       string   `json:"agentAccent,omitempty"`

	// DelegateReport is the account of the task a delegate_task call handed
	// out, kept on that call's result: how it ended, what the other agent
	// answered, made, left waiting and published. Nil on every other message
	// and on a call refused before anybody was asked.
	DelegateReport *DelegateReport `json:"delegateReport,omitempty" keel:"delegate_report,jsonb"`

	// ScheduleID is the conversation schedule a Schedule message made. The
	// schedule may since have been deleted; the card then says so.
	ScheduleID pulid.ID `json:"scheduleId,omitempty" keel:"schedule_id,nullzero"`
	// ProposalID is the agent proposal whose approval made a Schedule card.
	// Its words are the request the agent wrote out and the person approved,
	// not words they typed, so it is drawn as the approved proposal. Empty on
	// a card the person typed and on every other message.
	ProposalID pulid.ID `json:"proposalId,omitempty" keel:"proposal_id,nullzero"`
	// Handoff is what a hand-off carried, on the card it left in the
	// conversation handed off and on the brief that opens the new one. Nil
	// on every other message.
	Handoff *Handoff `json:"handoff,omitempty" keel:"handoff,jsonb"`

	Content string `json:"content" keel:"content,nullzero"`

	// WorldChanges are the records a WorldChange notice reported changed.
	// Nil on every other message.
	WorldChanges []WorldChange `json:"worldChanges,omitempty" keel:"world_changes,jsonb,nullzero"`

	// Compaction says what a compaction summary stands in for. Nil on every
	// other message.
	Compaction *Compaction `json:"compaction,omitempty" keel:"compaction,jsonb"`

	// ToolCalls is what an assistant turn asked for, stored as the normalized
	// shape rather than any one provider's wire format.
	ToolCalls []ToolCallRecord `json:"toolCalls" keel:"tool_calls,jsonb,nullzero"`
	// ToolCallID and ToolName tie a Tool-role message to the call it answers.
	ToolCallID  string                    `json:"toolCallId"            keel:"tool_call_id,nullzero"`
	ToolName    string                    `json:"toolName"              keel:"tool_name,nullzero"`
	ToolFailed  bool                      `json:"toolFailed"            keel:"tool_failed"`
	ToolVerdict string                    `json:"toolVerdict,omitempty" keel:"tool_verdict,nullzero"`
	ToolEffect  agenttooltrust.ToolEffect `json:"effect,omitempty"`
	ToolSummary string                    `json:"summary,omitempty"     keel:"tool_summary,nullzero"`
	FoundTools  []string                  `json:"foundTools,omitempty"  keel:"found_tools,jsonb,nullzero"`
	// HandOffAgents are the person's other agents a find_tools answer named
	// as holding what this agent could not call, set only on that call's
	// result, so the Desk's hand-off menu offers them first.
	HandOffAgents []pulid.ID `json:"handOffAgents,omitempty" keel:"hand_off_agents,jsonb,nullzero"`

	// ScopeStage, ScopeCategory and ScopeReason record the guard's verdict on a
	// user turn, or on an assistant turn the output guard refused.
	ScopeStage    string `json:"scopeStage"    keel:"scope_stage,nullzero"`
	ScopeCategory string `json:"scopeCategory" keel:"scope_category,nullzero"`
	ScopeReason   string `json:"scopeReason"   keel:"scope_reason,nullzero"`
	Refused       bool   `json:"refused"       keel:"refused"`

	// PageContext is what the person was looking at when they sent a user
	// turn, kept so an answer can be reviewed against the page it was about.
	PageContext *agentpagecontext.PageContext `json:"pageContext" keel:"page_context,jsonb"`

	// Attachments are the files the person handed over with a user turn, and
	// Mentions the records they named. Both are kept on the turn so an
	// answer can be read against what it was given.
	Attachments []MessageAttachment          `json:"attachments" keel:"attachments,jsonb,nullzero"`
	Mentions    []agentpagecontext.EntityRef `json:"mentions"    keel:"mentions,jsonb,nullzero"`

	// Reasoning is what the model thought before it answered, as much as the
	// provider lets through, plus whatever the provider needs handed back to
	// continue the same line of thought on the next call. Nil when the model
	// did not reason out loud or the provider was not asked to let it.
	Reasoning *ReasoningTrace `json:"reasoning" keel:"reasoning,jsonb"`

	Model string `json:"model" keel:"model,nullzero"`
	// Truncated says the provider stopped partway through this reply, and
	// FallbackFrom names the provider asked first when another one answered.
	Truncated    bool              `json:"truncated,omitempty"    keel:"truncated"`
	FallbackFrom *ProviderFallback `json:"fallbackFrom,omitempty" keel:"fallback_from,jsonb"`
	// Failure says why the reply did not finish, for a reply that is only a
	// closing note: the models that were asked, or that it was stopped.
	Failure      *ReplyFailure `json:"failure,omitempty" keel:"failure,jsonb"`
	ProviderID   pulid.ID      `json:"providerId"        keel:"provider_id,nullzero"`
	InputTokens  int           `json:"inputTokens"       keel:"input_tokens"`
	OutputTokens int           `json:"outputTokens"      keel:"output_tokens"`
	// ProviderMessageID is the provider's own id for this reply, which the
	// next call to the same provider names when it asks the Claude API why
	// its prompt cache missed.
	ProviderMessageID string `json:"providerMessageId,omitempty" keel:"provider_message_id,nullzero"`
	// LatencyMs is how long the model took to answer this turn; CostUSD is
	// what it cost at the provider's price, nil where no price is configured.
	LatencyMs int64            `json:"latencyMs" keel:"latency_ms,nullzero"`
	CostUSD   *decimal.Decimal `json:"costUsd"   keel:"cost_usd"`

	// UsedMemoryIDs are the memories the turn this reply ends used, and
	// SavedMemories what it kept or offered to keep; both only on a turn's
	// last reply. Memories is each of them as the reader may see it, filled
	// as the thread is served.
	UsedMemoryIDs []pulid.ID    `json:"usedMemoryIds,omitempty" keel:"used_memory_ids,jsonb,nullzero"`
	SavedMemories []SavedMemory `json:"savedMemories,omitempty" keel:"saved_memories,jsonb,nullzero"`
	Memories      []MemoryNote  `json:"memories,omitempty"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	Thread *Thread `json:"thread,omitempty" keel:"belongs:thread_id"`
}

// WorldChange is a record that changed elsewhere while a reply was being
// written, as the reply was told.
type WorldChange struct {
	RecordID    string   `json:"recordId"`
	Resource    string   `json:"resource"`
	Label       string   `json:"label,omitempty"`
	Action      string   `json:"action"`
	Fields      []string `json:"fields,omitempty"`
	ActorType   string   `json:"actorType,omitempty"`
	ActorUserID string   `json:"actorUserId,omitempty"`
	At          int64    `json:"at"`
}

// SavedMemory is a memory a turn kept through the remember tool, by the call
// that kept it. Pending is a memory offered rather than kept: the person asked
// to be asked first, and it waits for them to accept it.
type SavedMemory struct {
	ID      pulid.ID `json:"id"`
	CallID  string   `json:"callId"`
	Pending bool     `json:"pending"`
}

func MergeSavedMemories(current, added []SavedMemory) []SavedMemory {
	merged := make([]SavedMemory, 0, len(current)+len(added))
	merged = append(merged, current...)
	for _, memory := range added {
		if memory.ID.IsNil() || slices.ContainsFunc(merged, func(kept SavedMemory) bool {
			return kept.ID == memory.ID
		}) {
			continue
		}
		merged = append(merged, memory)
	}

	return merged
}

// MemoryNote is a memory a reply used or saved, as its reader sees it: what
// it says, who it is kept for, where it came from, and whether the reader may
// change it.
type MemoryNote struct {
	ID          pulid.ID        `json:"id"`
	Content     string          `json:"content"`
	Kind        string          `json:"kind"`
	Scope       string          `json:"scope"`
	RoleID      pulid.ID        `json:"roleId,omitempty"`
	RoleName    string          `json:"roleName,omitempty"`
	Status      string          `json:"status"`
	Source      string          `json:"source"`
	SourceTitle string          `json:"sourceTitle,omitempty"`
	CreatedAt   int64           `json:"createdAt"`
	Version     int64           `json:"version"`
	Editable    bool            `json:"editable"`
	Reason      string          `json:"reason,omitempty"`
	Replaces    *MemoryNoteLink `json:"replaces,omitempty"`
	ReplacedBy  *MemoryNoteLink `json:"replacedBy,omitempty"`
}

type MemoryNoteLink struct {
	ID      pulid.ID `json:"id"`
	Content string   `json:"content"`
	Status  string   `json:"status"`
}

// MessageAttachment is one file on a user turn: the document it became, and
// enough about it to draw a chip without reading the document back.
type MessageAttachment struct {
	DocumentID  pulid.ID `json:"documentId"`
	FileName    string   `json:"fileName"`
	ContentType string   `json:"contentType,omitempty"`
	FileSize    int64    `json:"fileSize,omitempty"`
	// PoorlyRead marks a file whose reading finished but could make out
	// little of it, such as a blurred photo, so the Desk can ask for a
	// clearer copy.
	PoorlyRead bool `json:"poorlyRead,omitempty"`
}

func CarriesAttachments(messages []Message) bool {
	for idx := range messages {
		if messages[idx].Role == RoleUser && len(messages[idx].Attachments) > 0 {
			return true
		}
	}

	return false
}

// ReasoningTrace is a model's thinking, kept in two parts.
//
// Text is what a person can read: the full chain for a provider that streams
// it, a summary for one that only summarises. The rest is opaque and belongs
// to the provider: Anthropic signs each thinking block and refuses a tool
// result that arrives without the signed block it followed; OpenAI's Responses
// API ties a reasoning item to the function calls it produced and rejects the
// calls replayed without it. Those are stored so the conversation can continue
// where it left off, not so anyone can read them.
type ReasoningTrace struct {
	Text string `json:"text"`
	// Signature is Anthropic's signature over the thinking block, or the
	// Responses API's reasoning item id.
	Signature string `json:"signature,omitempty"`
	// Encrypted is the Responses API's encrypted reasoning content.
	Encrypted string `json:"encrypted,omitempty"`
	// Redacted holds Anthropic's redacted_thinking blocks, replayed verbatim.
	Redacted []string `json:"redacted,omitempty"`
	// ProviderKind names the protocol that produced the trace. A signature
	// only means something to the provider that signed it, and a thread can
	// change model between turns, so an adapter replays only its own.
	ProviderKind string          `json:"providerKind,omitempty"`
	Blocks       []ProviderBlock `json:"blocks,omitempty"`
}

// ReplayableBy reports whether an adapter of the given kind may send this
// trace back. A trace from before kinds were recorded is replayed as before.
func (t *ReasoningTrace) ReplayableBy(kind string) bool {
	if t == nil {
		return false
	}

	return t.ProviderKind == "" || t.ProviderKind == kind
}

// Readable reports whether there is anything a person could be shown.
func (r *ReasoningTrace) Readable() bool {
	return r != nil && r.Text != ""
}

// ToolCallRecord is a persisted tool request.
type ToolCallRecord struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name"`
	Arguments map[string]any            `json:"arguments"`
	Effect    agenttooltrust.ToolEffect `json:"effect,omitempty"`
	// ProviderData and ProviderID keep what the provider attached to the
	// call and which provider that was, so a later turn on the same provider
	// can send it back. Gemini refuses a replayed call without its thought
	// signature; another provider would refuse the field itself.
	ProviderData map[string]any `json:"providerData,omitempty"`
	ProviderID   pulid.ID       `json:"providerId,omitempty"`
}

func (m *Message) BeforeCreate(ctx context.Context) error {
	if m.Kind == "" {
		m.Kind = MessageKindMessage
	}

	return nil
}

func StampUnstamped(messages []Message, now int64) {
	next := now
	for idx := len(messages) - 1; idx >= 0; idx-- {
		switch stamp := messages[idx].CreatedAt; {
		case stamp <= 0:
			messages[idx].CreatedAt = next
		case stamp > next:
			messages[idx].CreatedAt = next
		default:
			next = stamp
		}
	}
}

// Delegated reports a step another agent took on a task this conversation's
// agent handed it.
func (m *Message) Delegated() bool {
	return m.Kind == MessageKindDelegated
}

func (m *Message) GetID() pulid.ID { return m.ID }

func (m *Message) GetTableName() string { return "assistant_messages" }

func (m *Message) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "amsg",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "content", Type: domaintypes.FieldTypeText},
			{Name: "role", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (m *Message) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.ThreadID, validation.Required.Error("Thread is required")),
		validation.Field(&m.Role,
			validation.Required.Error("Role is required"),
			domainvalidation.ValidEnum[Role]("Role is invalid"),
		),
		validation.Field(&m.Sequence,
			validation.Min(0).Error("Sequence cannot be negative"),
		),
		validation.Field(&m.InputTokens,
			validation.Min(0).Error("Input tokens cannot be negative"),
		),
		validation.Field(&m.OutputTokens,
			validation.Min(0).Error("Output tokens cannot be negative"),
		),
	))
}

// ProviderFallback is the provider a reply was asked of first, and why it did
// not give it.
// ReplyFailure is why a reply did not finish.
type ReplyFailure struct {
	// Kind is no_model when every model asked failed, interrupted when the
	// reply broke off partway, stopped when the person stopped it, and
	// before_start for any other failure before a word arrived.
	Kind string `json:"kind"`
	// Providers are the models asked, and any the organization has that were
	// not given the task, each with what happened.
	Providers []FailedProvider `json:"providers,omitempty"`
}

// FailedProvider is one model a failed reply was asked of.
type FailedProvider struct {
	Name   string `json:"name"`
	Model  string `json:"model"`
	Vendor string `json:"vendor"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// The kinds of ReplyFailure.
const (
	ReplyFailureNoModel     = "no_model"
	ReplyFailureInterrupted = "interrupted"
	ReplyFailureStopped     = "stopped"
	ReplyFailureBeforeStart = "before_start"
)

type ProviderFallback struct {
	ProviderID pulid.ID `json:"providerId"`
	Name       string   `json:"name"`
	Model      string   `json:"model"`
	Status     string   `json:"status"`
}
