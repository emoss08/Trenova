package conversation

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// Message is one turn in a thread.
//
// Tool traffic is persisted alongside the prose rather than discarded after the
// loop, because "which records did the assistant actually read before saying
// that" is the question anyone reviewing an answer will ask. The scope columns
// record the guard's verdict for the same reason: a refusal is evidence about how
// the assistant is being used, and it is only useful if it survives the request.
type Message struct {
	bun.BaseModel `bun:"table:assistant_messages,alias:amsg" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	ThreadID pulid.ID `json:"threadId" bun:"thread_id,type:VARCHAR(100),notnull"`
	// Sequence orders messages within a thread without relying on a timestamp,
	// which can collide inside a single fast tool loop.
	Sequence int  `json:"sequence" bun:"sequence,type:INTEGER,notnull"`
	Role     Role `json:"role"     bun:"role,type:VARCHAR(50),notnull"`
	// Kind is Message for everything but the note that starts the turn after
	// a decision and the steps of another agent the turn handed work to; see
	// MessageKind.
	Kind MessageKind `json:"kind"     bun:"kind,type:VARCHAR(50),notnull,default:'Message'"`
	// AgentDefinitionID and DelegateCallID mark a step another agent took on
	// a task this conversation's agent handed it: which agent, and the
	// delegate_task call it answers. Both are empty on the conversation's own
	// messages. AgentName, AgentIcon and AgentAccent are the agent's name
	// and mark as the thread is served.
	AgentDefinitionID pulid.ID `json:"agentId,omitempty"        bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	DelegateCallID    string   `json:"delegateCallId,omitempty" bun:"delegate_call_id,type:VARCHAR(200),nullzero"`
	AgentName         string   `json:"agentName,omitempty"      bun:"-"`
	AgentIcon         string   `json:"agentIcon,omitempty"      bun:"-"`
	AgentAccent       string   `json:"agentAccent,omitempty"    bun:"-"`

	// DelegateReport is the account of the task a delegate_task call handed
	// out, kept on that call's result: how it ended, what the other agent
	// answered, made, left waiting and published. Nil on every other message
	// and on a call refused before anybody was asked.
	DelegateReport *DelegateReport `json:"delegateReport,omitempty" bun:"delegate_report,type:JSONB,nullzero"`

	// ScheduleID is the conversation schedule a Schedule message made. The
	// schedule may since have been deleted; the card then says so.
	ScheduleID pulid.ID `json:"scheduleId,omitempty" bun:"schedule_id,type:VARCHAR(100),nullzero"`
	// Handoff is what a hand-off carried, on the card it left in the
	// conversation handed off and on the brief that opens the new one. Nil
	// on every other message.
	Handoff *Handoff `json:"handoff,omitempty" bun:"handoff,type:JSONB,nullzero"`

	Content string `json:"content" bun:"content,type:TEXT,nullzero"`

	// Compaction says what a compaction summary stands in for. Nil on every
	// other message.
	Compaction *Compaction `json:"compaction,omitempty" bun:"compaction,type:JSONB,nullzero"`

	// ToolCalls is what an assistant turn asked for, stored as the normalized
	// shape rather than any one provider's wire format.
	ToolCalls []ToolCallRecord `json:"toolCalls"  bun:"tool_calls,type:jsonb,nullzero"`
	// ToolCallID and ToolName tie a Tool-role message to the call it answers.
	ToolCallID  string           `json:"toolCallId" bun:"tool_call_id,type:VARCHAR(200),nullzero"`
	ToolName    string           `json:"toolName"   bun:"tool_name,type:VARCHAR(200),nullzero"`
	ToolFailed  bool             `json:"toolFailed" bun:"tool_failed,type:BOOLEAN,notnull,default:false"`
	ToolVerdict string           `json:"toolVerdict,omitempty" bun:"tool_verdict,type:VARCHAR(50),nullzero"`
	ToolEffect  agent.ToolEffect `json:"effect,omitempty" bun:"-"`
	ToolSummary string           `json:"summary,omitempty" bun:"tool_summary,type:TEXT,nullzero"`
	FoundTools  []string         `json:"foundTools,omitempty" bun:"found_tools,type:JSONB,nullzero"`

	// ScopeStage, ScopeCategory and ScopeReason record the guard's verdict on a
	// user turn, or on an assistant turn the output guard refused.
	ScopeStage    string `json:"scopeStage"    bun:"scope_stage,type:VARCHAR(50),nullzero"`
	ScopeCategory string `json:"scopeCategory" bun:"scope_category,type:VARCHAR(50),nullzero"`
	ScopeReason   string `json:"scopeReason"   bun:"scope_reason,type:VARCHAR(50),nullzero"`
	Refused       bool   `json:"refused"       bun:"refused,type:BOOLEAN,notnull,default:false"`

	// PageContext is what the person was looking at when they sent a user
	// turn, kept so an answer can be reviewed against the page it was about.
	PageContext *agent.PageContext `json:"pageContext" bun:"page_context,type:JSONB,nullzero"`

	// Attachments are the files the person handed over with a user turn, and
	// Mentions the records they named. Both are kept on the turn so an
	// answer can be read against what it was given.
	Attachments []MessageAttachment `json:"attachments" bun:"attachments,type:JSONB,nullzero"`
	Mentions    []agent.EntityRef   `json:"mentions"    bun:"mentions,type:JSONB,nullzero"`

	// Reasoning is what the model thought before it answered, as much as the
	// provider lets through, plus whatever the provider needs handed back to
	// continue the same line of thought on the next call. Nil when the model
	// did not reason out loud or the provider was not asked to let it.
	Reasoning *ReasoningTrace `json:"reasoning" bun:"reasoning,type:JSONB,nullzero"`

	Model string `json:"model"        bun:"model,type:VARCHAR(200),nullzero"`
	// Truncated says the provider stopped partway through this reply, and
	// FallbackFrom names the provider asked first when another one answered.
	Truncated    bool              `json:"truncated,omitempty"    bun:"truncated,type:BOOLEAN,notnull,default:false"`
	FallbackFrom *ProviderFallback `json:"fallbackFrom,omitempty" bun:"fallback_from,type:JSONB,nullzero"`
	// Failure says why the reply did not finish, for a reply that is only a
	// closing note: the models that were asked, or that it was stopped.
	Failure      *ReplyFailure `json:"failure,omitempty" bun:"failure,type:JSONB,nullzero"`
	ProviderID   pulid.ID      `json:"providerId"   bun:"provider_id,type:VARCHAR(100),nullzero"`
	InputTokens  int           `json:"inputTokens"  bun:"input_tokens,type:INTEGER,notnull,default:0"`
	OutputTokens int           `json:"outputTokens" bun:"output_tokens,type:INTEGER,notnull,default:0"`
	// LatencyMs is how long the model took to answer this turn; CostUSD is
	// what it cost at the provider's price, nil where no price is configured.
	LatencyMs int64            `json:"latencyMs"    bun:"latency_ms,type:BIGINT,nullzero"`
	CostUSD   *decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),nullzero"`

	// UsedMemoryIDs are the memories the turn this reply ends used, and
	// SavedMemories what it kept or offered to keep; both only on a turn's
	// last reply. Memories is each of them as the reader may see it, filled
	// as the thread is served.
	UsedMemoryIDs []pulid.ID    `json:"usedMemoryIds,omitempty" bun:"used_memory_ids,type:JSONB,nullzero"`
	SavedMemories []SavedMemory `json:"savedMemories,omitempty" bun:"saved_memories,type:JSONB,nullzero"`
	Memories      []MemoryNote  `json:"memories,omitempty"      bun:"-"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Thread *Thread `json:"thread,omitempty" bun:"rel:belongs-to,join:thread_id=id"`
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
	ID          pulid.ID `json:"id"`
	Content     string   `json:"content"`
	Kind        string   `json:"kind"`
	Scope       string   `json:"scope"`
	RoleID      pulid.ID `json:"roleId,omitempty"`
	RoleName    string   `json:"roleName,omitempty"`
	Status      string   `json:"status"`
	Source      string   `json:"source"`
	SourceTitle string   `json:"sourceTitle,omitempty"`
	CreatedAt   int64    `json:"createdAt"`
	Version     int64    `json:"version"`
	Editable    bool     `json:"editable"`
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
	ProviderKind string `json:"providerKind,omitempty"`
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
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Arguments map[string]any   `json:"arguments"`
	Effect    agent.ToolEffect `json:"effect,omitempty"`
	// ProviderData and ProviderID keep what the provider attached to the
	// call and which provider that was, so a later turn on the same provider
	// can send it back. Gemini refuses a replayed call without its thought
	// signature; another provider would refuse the field itself.
	ProviderData map[string]any `json:"providerData,omitempty"`
	ProviderID   pulid.ID       `json:"providerId,omitempty"`
	// Why is the model's own account of the step, shown under "Why this
	// step?": what it looked at, why it chose this, and what it passed over.
	Why *StepRationale `json:"why,omitempty"`
}

// StepRationale is why the model took one step. It comes from the model with
// the call, in a short phrase each, and is shown as the model wrote it.
type StepRationale struct {
	Saw       string `json:"saw,omitempty"`
	Because   string `json:"because,omitempty"`
	InsteadOf string `json:"insteadOf,omitempty"`
}

func (m *Message) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("amsg_")
		}
		if m.CreatedAt == 0 {
			m.CreatedAt = timeutils.NowUnix()
		}
		if m.Kind == "" {
			m.Kind = MessageKindMessage
		}
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
