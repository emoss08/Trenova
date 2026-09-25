package aiaudit

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*AIAuditEvent)(nil)
	_ pagination.CursorEntity        = (*AIAuditEvent)(nil)
	_ domaintypes.PostgresSearchable = (*AIAuditEvent)(nil)
)

const (
	EventIDPrefix           = "aiae_"
	MaxSourceKeyLength      = 250
	MaxNameLength           = 255
	MaxAgentNameLength      = 100
	MaxResultSummaryLength  = 500
	MaxReasonLength         = 4000
	MaxEntityTypeLength     = 100
	MaxEntityIDLength       = 100
	MaxToolNameLength       = 200
	MaxModelLength          = 200
	MaxCallIDLength         = 200
	MaxStepKeyLength        = 120
	MaxArgumentsBytes       = 16 * 1024
	ConfidentialPlaceholder = "[confidential]"
	WithheldPlaceholder     = "[withheld]"
	TruncatedPlaceholder    = "[truncated]"
)

// ArgumentSensitivity is how sensitive each recorded argument was when it was
// projected, so a reader is shown only what their role reaches on the
// resource the tool acts on. Paths use "." between keys and "[]" for every
// element of a list.
type ArgumentSensitivity struct {
	Resource permission.Resource                    `json:"resource,omitempty"`
	Levels   map[string]permission.FieldSensitivity `json:"levels,omitempty"`
}

// AIAuditEvent is one thing an agent did, decided or had decided for it,
// written once and never changed. Each tenant's rows form a hash chain in seq
// order, so a changed or removed row is found by walking it.
type AIAuditEvent struct {
	bun.BaseModel `bun:"table:ai_audit_events,alias:aiae" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	Seq        int64   `json:"seq"        bun:"seq,type:BIGINT,notnull"`
	SourceKey  string  `json:"sourceKey"  bun:"source_key,type:VARCHAR(250),notnull"`
	OccurredAt int64   `json:"occurredAt" bun:"occurred_at,type:BIGINT,notnull"`
	RecordedAt int64   `json:"recordedAt" bun:"recorded_at,type:BIGINT,notnull"`
	Kind       Kind    `json:"kind"       bun:"kind,type:VARCHAR(30),notnull"`
	Outcome    Outcome `json:"outcome"    bun:"outcome,type:VARCHAR(30),notnull"`

	PrincipalType      PrincipalType `json:"principalType"      bun:"principal_type,type:VARCHAR(20),notnull"`
	PrincipalID        string        `json:"principalId"        bun:"principal_id,type:VARCHAR(100),nullzero"`
	OnBehalfOfUserID   pulid.ID      `json:"onBehalfOfUserId"   bun:"on_behalf_of_user_id,type:VARCHAR(100),nullzero"`
	OnBehalfOfUserName string        `json:"onBehalfOfUserName" bun:"on_behalf_of_user_name,type:VARCHAR(255),nullzero"`
	DecidedByUserID    pulid.ID      `json:"decidedByUserId"    bun:"decided_by_user_id,type:VARCHAR(100),nullzero"`
	DecidedByUserName  string        `json:"decidedByUserName"  bun:"decided_by_user_name,type:VARCHAR(255),nullzero"`

	AgentDefinitionID      pulid.ID `json:"agentDefinitionId"      bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	AgentDefinitionVersion *int64   `json:"agentDefinitionVersion" bun:"agent_definition_version,type:BIGINT,nullzero"`
	AgentName              string   `json:"agentName"              bun:"agent_name,type:VARCHAR(100),nullzero"`

	OwnerKind      string   `json:"ownerKind"      bun:"owner_kind,type:VARCHAR(20),nullzero"`
	OwnerID        pulid.ID `json:"ownerId"        bun:"owner_id,type:VARCHAR(100),nullzero"`
	RunID          pulid.ID `json:"runId"          bun:"run_id,type:VARCHAR(100),nullzero"`
	TurnID         pulid.ID `json:"turnId"         bun:"turn_id,type:VARCHAR(100),nullzero"`
	ThreadID       pulid.ID `json:"threadId"       bun:"thread_id,type:VARCHAR(100),nullzero"`
	ProposalID     pulid.ID `json:"proposalId"     bun:"proposal_id,type:VARCHAR(100),nullzero"`
	PlanID         pulid.ID `json:"planId"         bun:"plan_id,type:VARCHAR(100),nullzero"`
	DecisionID     pulid.ID `json:"decisionId"     bun:"decision_id,type:VARCHAR(100),nullzero"`
	StepKey        string   `json:"stepKey"        bun:"step_key,type:VARCHAR(120),nullzero"`
	CallID         string   `json:"callId"         bun:"call_id,type:VARCHAR(200),nullzero"`
	DelegateCallID string   `json:"delegateCallId" bun:"delegate_call_id,type:VARCHAR(200),nullzero"`
	ParentOwnerID  pulid.ID `json:"parentOwnerId"  bun:"parent_owner_id,type:VARCHAR(100),nullzero"`
	TraceID        string   `json:"traceId"        bun:"trace_id,type:VARCHAR(32),nullzero"`
	SpanID         string   `json:"spanId"         bun:"span_id,type:VARCHAR(16),nullzero"`

	ProviderID       pulid.ID         `json:"providerId"       bun:"provider_id,type:VARCHAR(100),nullzero"`
	ProviderKind     string           `json:"providerKind"     bun:"provider_kind,type:VARCHAR(50),nullzero"`
	Model            string           `json:"model"            bun:"model,type:VARCHAR(200),nullzero"`
	Attempt          *int             `json:"attempt"          bun:"attempt,type:INTEGER,nullzero"`
	Failover         bool             `json:"failover"         bun:"failover,type:BOOLEAN,notnull,default:false"`
	InputTokens      int              `json:"inputTokens"      bun:"input_tokens,type:INTEGER,notnull,default:0"`
	OutputTokens     int              `json:"outputTokens"     bun:"output_tokens,type:INTEGER,notnull,default:0"`
	ReasoningTokens  int              `json:"reasoningTokens"  bun:"reasoning_tokens,type:INTEGER,notnull,default:0"`
	CacheReadTokens  int              `json:"cacheReadTokens"  bun:"cache_read_tokens,type:INTEGER,notnull,default:0"`
	CacheWriteTokens int              `json:"cacheWriteTokens" bun:"cache_write_tokens,type:INTEGER,notnull,default:0"`
	CostUSD          *decimal.Decimal `json:"costUsd"          bun:"cost_usd,type:NUMERIC(14,6),nullzero"`
	LatencyMs        *int64           `json:"latencyMs"        bun:"latency_ms,type:BIGINT,nullzero"`

	ToolName    string   `json:"toolName"    bun:"tool_name,type:VARCHAR(200),nullzero"`
	ToolEffect  string   `json:"toolEffect"  bun:"tool_effect,type:VARCHAR(20),nullzero"`
	EgressClass string   `json:"egressClass" bun:"egress_class,type:VARCHAR(30),nullzero"`
	Tier        string   `json:"tier"        bun:"tier,type:VARCHAR(30),nullzero"`
	TierSource  string   `json:"tierSource"  bun:"tier_source,type:VARCHAR(30),nullzero"`
	HeldBy      []string `json:"heldBy"      bun:"held_by,type:TEXT[],array,notnull,default:'{}'"`
	Reason      string   `json:"reason"      bun:"reason,type:TEXT,nullzero"`

	// Arguments are what the call was made with, redacted when projected.
	Arguments           map[string]any       `json:"arguments"           bun:"arguments,type:JSONB,nullzero,json_use_number"`
	ArgumentSensitivity *ArgumentSensitivity `json:"argumentSensitivity" bun:"argument_sensitivity,type:JSONB,nullzero"`
	RedactedPaths       []string             `json:"redactedPaths"       bun:"redacted_paths,type:TEXT[],array,notnull,default:'{}'"`
	ArgumentsTruncated  bool                 `json:"argumentsTruncated"  bun:"arguments_truncated,type:BOOLEAN,notnull,default:false"`
	ResultSummary       string               `json:"resultSummary"       bun:"result_summary,type:VARCHAR(500),nullzero"`

	EntityType    string `json:"entityType"    bun:"entity_type,type:VARCHAR(100),nullzero"`
	EntityID      string `json:"entityId"      bun:"entity_id,type:VARCHAR(100),nullzero"`
	VersionBefore *int64 `json:"versionBefore" bun:"version_before,type:BIGINT,nullzero"`
	VersionAfter  *int64 `json:"versionAfter"  bun:"version_after,type:BIGINT,nullzero"`
	WindowStart   int64  `json:"windowStart"   bun:"window_start,type:BIGINT,nullzero"`
	WindowEnd     int64  `json:"windowEnd"     bun:"window_end,type:BIGINT,nullzero"`

	Tainted         bool            `json:"tainted"         bun:"tainted,type:BOOLEAN,notnull,default:false"`
	Taint           *agent.RunTaint `json:"taint"           bun:"taint,type:JSONB,nullzero"`
	ExternalContent bool            `json:"externalContent" bun:"external_content,type:BOOLEAN,notnull,default:false"`
	Simulated       bool            `json:"simulated"       bun:"simulated,type:BOOLEAN,notnull,default:false"`
	Purpose         Purpose         `json:"purpose"         bun:"purpose,type:VARCHAR(20),notnull,default:'Live'"`
	// Reconstructed marks a row whose provenance was worked out from what the
	// source rows held before their link columns were written, rather than
	// read from those columns.
	Reconstructed bool `json:"reconstructed"   bun:"reconstructed,type:BOOLEAN,notnull,default:false"`

	PrevHash    string `json:"prevHash"    bun:"prev_hash,type:VARCHAR(64),notnull"`
	Hash        string `json:"hash"        bun:"hash,type:VARCHAR(64),notnull"`
	HashKeyID   string `json:"hashKeyId"   bun:"hash_key_id,type:VARCHAR(40),nullzero"`
	HashVersion int16  `json:"hashVersion" bun:"hash_version,type:SMALLINT,notnull"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (e *AIAuditEvent) GetID() pulid.ID {
	return e.ID
}

func (e *AIAuditEvent) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *AIAuditEvent) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *AIAuditEvent) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (e *AIAuditEvent) GetTableName() string {
	return "ai_audit_events"
}

func (e *AIAuditEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aiae",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "tool_name", Type: domaintypes.FieldTypeText},
			{Name: "agent_name", Type: domaintypes.FieldTypeText},
			{Name: "entity_id", Type: domaintypes.FieldTypeText},
			{Name: "on_behalf_of_user_name", Type: domaintypes.FieldTypeText},
			{Name: "decided_by_user_name", Type: domaintypes.FieldTypeText},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
			{Name: "outcome", Type: domaintypes.FieldTypeEnum},
		},
	}
}

// ActingUserID is the person the event's action was taken for or by, which
// is how an audit entry written for the same write is matched to it.
func (e *AIAuditEvent) ActingUserID() pulid.ID {
	if e.DecidedByUserID.IsNotNil() {
		return e.DecidedByUserID
	}

	return e.OnBehalfOfUserID
}

func (e *AIAuditEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew(EventIDPrefix)
		}
		if e.RecordedAt == 0 {
			e.RecordedAt = timeutils.NowUnix()
		}
		if e.CreatedAt == 0 {
			e.CreatedAt = e.RecordedAt
		}
		if e.HeldBy == nil {
			e.HeldBy = []string{}
		}
		if e.RedactedPaths == nil {
			e.RedactedPaths = []string{}
		}
		if e.Purpose == "" {
			e.Purpose = PurposeLive
		}
	}

	return nil
}
