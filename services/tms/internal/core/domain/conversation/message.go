package conversation

import (
	"context"
	"github.com/shopspring/decimal"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	// a decision; see MessageKind.
	Kind MessageKind `json:"kind"     bun:"kind,type:VARCHAR(50),notnull,default:'Message'"`

	Content string `json:"content" bun:"content,type:TEXT,nullzero"`

	// ToolCalls is what an assistant turn asked for, stored as the normalized
	// shape rather than any one provider's wire format.
	ToolCalls []ToolCallRecord `json:"toolCalls"  bun:"tool_calls,type:jsonb,nullzero"`
	// ToolCallID and ToolName tie a Tool-role message to the call it answers.
	ToolCallID string `json:"toolCallId" bun:"tool_call_id,type:VARCHAR(200),nullzero"`
	ToolName   string `json:"toolName"   bun:"tool_name,type:VARCHAR(200),nullzero"`
	ToolFailed bool   `json:"toolFailed" bun:"tool_failed,type:BOOLEAN,notnull,default:false"`

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

	Model        string   `json:"model"        bun:"model,type:VARCHAR(200),nullzero"`
	ProviderID   pulid.ID `json:"providerId"   bun:"provider_id,type:VARCHAR(100),nullzero"`
	InputTokens  int      `json:"inputTokens"  bun:"input_tokens,type:INTEGER,notnull,default:0"`
	OutputTokens int      `json:"outputTokens" bun:"output_tokens,type:INTEGER,notnull,default:0"`
	// LatencyMs is how long the model took to answer this turn; CostUSD is
	// what it cost at the provider's price, nil where no price is configured.
	LatencyMs int64            `json:"latencyMs"    bun:"latency_ms,type:BIGINT,nullzero"`
	CostUSD   *decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Thread *Thread `json:"thread,omitempty" bun:"rel:belongs-to,join:thread_id=id"`
}

// MessageAttachment is one file on a user turn: the document it became, and
// enough about it to draw a chip without reading the document back.
type MessageAttachment struct {
	DocumentID  pulid.ID `json:"documentId"`
	FileName    string   `json:"fileName"`
	ContentType string   `json:"contentType,omitempty"`
	FileSize    int64    `json:"fileSize,omitempty"`
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
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	// ProviderData and ProviderID keep what the provider attached to the
	// call and which provider that was, so a later turn on the same provider
	// can send it back. Gemini refuses a replayed call without its thought
	// signature; another provider would refuse the field itself.
	ProviderData map[string]any `json:"providerData,omitempty"`
	ProviderID   pulid.ID       `json:"providerId,omitempty"`
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
