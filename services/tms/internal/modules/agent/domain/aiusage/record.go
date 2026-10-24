package aiusage

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*AIUsageRecord)(nil)

// Surface is which part of the system made the call.
type Surface string

const (
	SurfaceChat       = Surface("Chat")
	SurfaceStructured = Surface("Structured")
	SurfaceBackground = Surface("Background")
	SurfaceEvaluation = Surface("Evaluation")
	SurfaceIndexing   = Surface("Indexing")
	SurfaceRetrieval  = Surface("Retrieval")
)

func (s Surface) IsEmbedding() bool {
	switch s {
	case SurfaceIndexing, SurfaceRetrieval:
		return true
	case SurfaceChat, SurfaceStructured, SurfaceBackground, SurfaceEvaluation:
		return false
	}

	return false
}

// AIUsageRecord is one attempt to have a model answer: which provider, how long it
// took, what it consumed and what that consumption cost. One row per attempt
// rather than per turn, because a turn that fell through two providers before
// a third answered was three calls, three latencies and three bills.
type AIUsageRecord struct {
	bun.BaseModel `bun:"table:ai_usage_records,alias:aiu" json:"-"`
	keel.Table    `keel:"ai_usage_records"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:aiu_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	ProviderID   pulid.ID        `json:"providerId"   bun:"provider_id,type:VARCHAR(100),nullzero" keel:"provider_id,nullzero"`
	ProviderKind aiprovider.Kind `json:"providerKind" bun:"provider_kind,type:VARCHAR(50),notnull" keel:"provider_kind"`
	Model        string          `json:"model"        bun:"model,type:VARCHAR(200),notnull" keel:"model"`
	Task         aiprovider.Task `json:"task"         bun:"task,type:VARCHAR(100),notnull" keel:"task"`
	Surface      Surface         `json:"surface"      bun:"surface,type:VARCHAR(50),notnull" keel:"surface"`

	// Attribution: who and what the call was for, as far as the caller said.
	UserID            pulid.ID `json:"userId"            bun:"user_id,type:VARCHAR(100),nullzero" keel:"user_id,nullzero"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero" keel:"agent_definition_id,nullzero"`
	ThreadID          pulid.ID `json:"threadId"          bun:"thread_id,type:VARCHAR(100),nullzero" keel:"thread_id,nullzero"`
	RunID             pulid.ID `json:"runId"             bun:"run_id,type:VARCHAR(100),nullzero" keel:"run_id,nullzero"`

	Feature     Feature     `json:"feature"     bun:"feature,type:VARCHAR(50),nullzero" keel:"feature,nullzero"`
	SubjectType SubjectType `json:"subjectType" bun:"subject_type,type:VARCHAR(50),nullzero" keel:"subject_type,nullzero"`
	SubjectID   string      `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),nullzero" keel:"subject_id,nullzero"`

	Succeeded  bool   `json:"succeeded"  bun:"succeeded,type:BOOLEAN,notnull" keel:"succeeded"`
	ErrorClass string `json:"errorClass" bun:"error_class,type:VARCHAR(50),nullzero" keel:"error_class,nullzero"`
	// ErrorMessage is the provider's own words for a failed attempt, cut to
	// fit. The class says what kind of failure it was; this says why, which
	// is what an administrator needs when a provider refuses every request.
	ErrorMessage string `json:"errorMessage" bun:"error_message,type:VARCHAR(500),nullzero" keel:"error_message,nullzero"`
	Streamed     bool   `json:"streamed"     bun:"streamed,type:BOOLEAN,notnull,default:false" keel:"streamed"`

	LatencyMs       int64 `json:"latencyMs"       bun:"latency_ms,type:BIGINT,notnull" keel:"latency_ms"`
	InputTokens     int   `json:"inputTokens"     bun:"input_tokens,type:INTEGER,notnull,default:0" keel:"input_tokens"`
	OutputTokens    int   `json:"outputTokens"    bun:"output_tokens,type:INTEGER,notnull,default:0" keel:"output_tokens"`
	ReasoningTokens int   `json:"reasoningTokens" bun:"reasoning_tokens,type:INTEGER,notnull,default:0" keel:"reasoning_tokens"`
	// CostUSD is nil when the provider carries no pricing. It is not zero:
	// a call that cost something unknown must not be summed as free.
	CostUSD *decimal.Decimal `json:"costUsd" bun:"cost_usd,type:NUMERIC(14,6),nullzero" keel:"cost_usd"`

	// Provenance: the span the attempt ran in, the run or turn it was for and
	// the task a delegate was working on, the agent's version, which try of
	// the call this was and whether it had fallen over to another provider.
	TraceID                string             `json:"traceId"                bun:"trace_id,type:VARCHAR(32),nullzero" keel:"trace_id,nullzero"`
	SpanID                 string             `json:"spanId"                 bun:"span_id,type:VARCHAR(16),nullzero" keel:"span_id,nullzero"`
	OwnerKind              agent.RunOwnerKind `json:"ownerKind"              bun:"owner_kind,type:VARCHAR(20),nullzero" keel:"owner_kind,nullzero"`
	OwnerID                pulid.ID           `json:"ownerId"                bun:"owner_id,type:VARCHAR(100),nullzero" keel:"owner_id,nullzero"`
	DelegateCallID         string             `json:"delegateCallId"         bun:"delegate_call_id,type:VARCHAR(200),nullzero" keel:"delegate_call_id,nullzero"`
	AgentDefinitionVersion *int64             `json:"agentDefinitionVersion" bun:"agent_definition_version,type:BIGINT,nullzero" keel:"agent_definition_version"`
	Attempt                int                `json:"attempt"                bun:"attempt,type:INTEGER,nullzero" keel:"attempt,nullzero"`
	Failover               bool               `json:"failover"               bun:"failover,type:BOOLEAN,notnull,default:false" keel:"failover"`
	CacheReadTokens        int                `json:"cacheReadTokens"        bun:"cache_read_tokens,type:INTEGER,notnull,default:0" keel:"cache_read_tokens"`
	CacheWriteTokens       int                `json:"cacheWriteTokens"       bun:"cache_write_tokens,type:INTEGER,notnull,default:0" keel:"cache_write_tokens"`
	// FirstTokenMs is how long the attempt took to write the first piece of
	// its reply, text or thinking: how long a person watched nothing. Nil for
	// an attempt that wrote nothing a person reads, a reply that was only tool
	// calls or one that was not streamed.
	FirstTokenMs *int64 `json:"firstTokenMs" bun:"first_token_ms,type:BIGINT,nullzero" keel:"first_token_ms"`
	RecordedAtMs *int64 `json:"recordedAtMs" bun:"recorded_at_ms,type:BIGINT,scanonly" keel:"recorded_at_ms,readonly"`

	AdvisorModel            string           `json:"advisorModel"            bun:"advisor_model,type:VARCHAR(200),nullzero" keel:"advisor_model,nullzero"`
	AdvisorCalls            int              `json:"advisorCalls"            bun:"advisor_calls,type:INTEGER,notnull,default:0" keel:"advisor_calls"`
	AdvisorInputTokens      int              `json:"advisorInputTokens"      bun:"advisor_input_tokens,type:INTEGER,notnull,default:0" keel:"advisor_input_tokens"`
	AdvisorOutputTokens     int              `json:"advisorOutputTokens"     bun:"advisor_output_tokens,type:INTEGER,notnull,default:0" keel:"advisor_output_tokens"`
	AdvisorCacheReadTokens  int              `json:"advisorCacheReadTokens"  bun:"advisor_cache_read_tokens,type:INTEGER,notnull,default:0" keel:"advisor_cache_read_tokens"`
	AdvisorCacheWriteTokens int              `json:"advisorCacheWriteTokens" bun:"advisor_cache_write_tokens,type:INTEGER,notnull,default:0" keel:"advisor_cache_write_tokens"`
	AdvisorCostUSD          *decimal.Decimal `json:"advisorCostUsd"          bun:"advisor_cost_usd,type:NUMERIC(14,6),nullzero" keel:"advisor_cost_usd"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
}

func (r *AIUsageRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("aiu_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}
