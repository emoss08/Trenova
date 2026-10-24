package aiaudit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

var (
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
	keel.Table `keel:"ai_audit_events"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:aiae_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Seq        int64   `json:"seq" keel:"seq"`
	SourceKey  string  `json:"sourceKey" keel:"source_key"`
	OccurredAt int64   `json:"occurredAt" keel:"occurred_at"`
	RecordedAt int64   `json:"recordedAt" keel:"recorded_at"`
	Kind       Kind    `json:"kind" keel:"kind"`
	Outcome    Outcome `json:"outcome" keel:"outcome"`

	PrincipalType      PrincipalType `json:"principalType" keel:"principal_type"`
	PrincipalID        string        `json:"principalId" keel:"principal_id,nullzero"`
	OnBehalfOfUserID   pulid.ID      `json:"onBehalfOfUserId" keel:"on_behalf_of_user_id,nullzero"`
	OnBehalfOfUserName string        `json:"onBehalfOfUserName" keel:"on_behalf_of_user_name,nullzero"`
	DecidedByUserID    pulid.ID      `json:"decidedByUserId" keel:"decided_by_user_id,nullzero"`
	DecidedByUserName  string        `json:"decidedByUserName" keel:"decided_by_user_name,nullzero"`

	AgentDefinitionID      pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentDefinitionVersion *int64   `json:"agentDefinitionVersion" keel:"agent_definition_version"`
	AgentName              string   `json:"agentName" keel:"agent_name,nullzero"`

	OwnerKind      string   `json:"ownerKind" keel:"owner_kind,nullzero"`
	OwnerID        pulid.ID `json:"ownerId" keel:"owner_id,nullzero"`
	RunID          pulid.ID `json:"runId" keel:"run_id,nullzero"`
	TurnID         pulid.ID `json:"turnId" keel:"turn_id,nullzero"`
	ThreadID       pulid.ID `json:"threadId" keel:"thread_id,nullzero"`
	ProposalID     pulid.ID `json:"proposalId" keel:"proposal_id,nullzero"`
	PlanID         pulid.ID `json:"planId" keel:"plan_id,nullzero"`
	DecisionID     pulid.ID `json:"decisionId" keel:"decision_id,nullzero"`
	StepKey        string   `json:"stepKey" keel:"step_key,nullzero"`
	CallID         string   `json:"callId" keel:"call_id,nullzero"`
	DelegateCallID string   `json:"delegateCallId" keel:"delegate_call_id,nullzero"`
	ParentOwnerID  pulid.ID `json:"parentOwnerId" keel:"parent_owner_id,nullzero"`
	TraceID        string   `json:"traceId" keel:"trace_id,nullzero"`
	SpanID         string   `json:"spanId" keel:"span_id,nullzero"`

	ProviderID       pulid.ID         `json:"providerId" keel:"provider_id,nullzero"`
	ProviderKind     string           `json:"providerKind" keel:"provider_kind,nullzero"`
	Model            string           `json:"model" keel:"model,nullzero"`
	Attempt          *int             `json:"attempt" keel:"attempt"`
	Failover         bool             `json:"failover" keel:"failover"`
	InputTokens      int              `json:"inputTokens" keel:"input_tokens"`
	OutputTokens     int              `json:"outputTokens" keel:"output_tokens"`
	ReasoningTokens  int              `json:"reasoningTokens" keel:"reasoning_tokens"`
	CacheReadTokens  int              `json:"cacheReadTokens" keel:"cache_read_tokens"`
	CacheWriteTokens int              `json:"cacheWriteTokens" keel:"cache_write_tokens"`
	CostUSD          *decimal.Decimal `json:"costUsd" keel:"cost_usd"`
	LatencyMs        *int64           `json:"latencyMs" keel:"latency_ms"`

	ToolName    string   `json:"toolName" keel:"tool_name,nullzero"`
	ToolEffect  string   `json:"toolEffect" keel:"tool_effect,nullzero"`
	EgressClass string   `json:"egressClass" keel:"egress_class,nullzero"`
	Tier        string   `json:"tier" keel:"tier,nullzero"`
	TierSource  string   `json:"tierSource" keel:"tier_source,nullzero"`
	HeldBy      []string `json:"heldBy" keel:"held_by"`
	Reason      string   `json:"reason" keel:"reason,nullzero"`

	// Arguments are what the call was made with, redacted when projected.
	Arguments           map[string]any       `json:"arguments" keel:"arguments,jsonb,jsonnumber,nullzero"`
	ArgumentSensitivity *ArgumentSensitivity `json:"argumentSensitivity" keel:"argument_sensitivity,jsonb"`
	RedactedPaths       []string             `json:"redactedPaths" keel:"redacted_paths"`
	ArgumentsTruncated  bool                 `json:"argumentsTruncated" keel:"arguments_truncated"`
	ResultSummary       string               `json:"resultSummary" keel:"result_summary,nullzero"`

	EntityType    string `json:"entityType" keel:"entity_type,nullzero"`
	EntityID      string `json:"entityId" keel:"entity_id,nullzero"`
	VersionBefore *int64 `json:"versionBefore" keel:"version_before"`
	VersionAfter  *int64 `json:"versionAfter" keel:"version_after"`
	WindowStart   int64  `json:"windowStart" keel:"window_start,nullzero"`
	WindowEnd     int64  `json:"windowEnd" keel:"window_end,nullzero"`

	Tainted         bool            `json:"tainted" keel:"tainted"`
	Taint           *agent.RunTaint `json:"taint" keel:"taint,jsonb"`
	ExternalContent bool            `json:"externalContent" keel:"external_content"`
	Simulated       bool            `json:"simulated" keel:"simulated"`
	Purpose         Purpose         `json:"purpose" keel:"purpose"`
	// Reconstructed marks a row whose provenance was worked out from what the
	// source rows held before their link columns were written, rather than
	// read from those columns.
	Reconstructed bool `json:"reconstructed" keel:"reconstructed"`

	PrevHash    string `json:"prevHash" keel:"prev_hash"`
	Hash        string `json:"hash" keel:"hash"`
	HashKeyID   string `json:"hashKeyId" keel:"hash_key_id,nullzero"`
	HashVersion int16  `json:"hashVersion" keel:"hash_version"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (e *AIAuditEvent) BeforeCreate(_ context.Context) error {
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

	return nil
}
