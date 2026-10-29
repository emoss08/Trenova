package aiusage

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

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
	keel.Table `keel:"ai_usage_records"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aiu_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ProviderID   pulid.ID        `json:"providerId"   keel:"provider_id,nullzero"`
	ProviderKind aiprovider.Kind `json:"providerKind" keel:"provider_kind"`
	Model        string          `json:"model"        keel:"model"`
	Task         aitask.Task     `json:"task"         keel:"task"`
	Surface      Surface         `json:"surface"      keel:"surface"`

	// Attribution: who and what the call was for, as far as the caller said.
	UserID            pulid.ID `json:"userId"            keel:"user_id,nullzero"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	ThreadID          pulid.ID `json:"threadId"          keel:"thread_id,nullzero"`
	RunID             pulid.ID `json:"runId"             keel:"run_id,nullzero"`

	Feature     Feature     `json:"feature"     keel:"feature,nullzero"`
	SubjectType SubjectType `json:"subjectType" keel:"subject_type,nullzero"`
	SubjectID   string      `json:"subjectId"   keel:"subject_id,nullzero"`

	Succeeded  bool   `json:"succeeded"  keel:"succeeded"`
	ErrorClass string `json:"errorClass" keel:"error_class,nullzero"`
	// ErrorMessage is the provider's own words for a failed attempt, cut to
	// fit. The class says what kind of failure it was; this says why, which
	// is what an administrator needs when a provider refuses every request.
	ErrorMessage string `json:"errorMessage" keel:"error_message,nullzero"`
	Streamed     bool   `json:"streamed"     keel:"streamed"`

	LatencyMs       int64 `json:"latencyMs"       keel:"latency_ms"`
	InputTokens     int   `json:"inputTokens"     keel:"input_tokens"`
	OutputTokens    int   `json:"outputTokens"    keel:"output_tokens"`
	ReasoningTokens int   `json:"reasoningTokens" keel:"reasoning_tokens"`
	// CostUSD is nil when the provider carries no pricing. It is not zero:
	// a call that cost something unknown must not be summed as free.
	CostUSD *decimal.Decimal `json:"costUsd" keel:"cost_usd"`

	// Provenance: the span the attempt ran in, the run or turn it was for and
	// the task a delegate was working on, the agent's version, which try of
	// the call this was and whether it had fallen over to another provider.
	TraceID                string             `json:"traceId"                keel:"trace_id,nullzero"`
	SpanID                 string             `json:"spanId"                 keel:"span_id,nullzero"`
	OwnerKind              agent.RunOwnerKind `json:"ownerKind"              keel:"owner_kind,nullzero"`
	OwnerID                pulid.ID           `json:"ownerId"                keel:"owner_id,nullzero"`
	DelegateCallID         string             `json:"delegateCallId"         keel:"delegate_call_id,nullzero"`
	AgentDefinitionVersion *int64             `json:"agentDefinitionVersion" keel:"agent_definition_version"`
	Attempt                int                `json:"attempt"                keel:"attempt,nullzero"`
	Failover               bool               `json:"failover"               keel:"failover"`
	CacheReadTokens        int                `json:"cacheReadTokens"        keel:"cache_read_tokens"`
	CacheWriteTokens       int                `json:"cacheWriteTokens"       keel:"cache_write_tokens"`
	// FirstTokenMs is how long the attempt took to write the first piece of
	// its reply, text or thinking: how long a person watched nothing. Nil for
	// an attempt that wrote nothing a person reads, a reply that was only tool
	// calls or one that was not streamed.
	FirstTokenMs *int64 `json:"firstTokenMs" keel:"first_token_ms"`
	RecordedAtMs *int64 `json:"recordedAtMs" keel:"recorded_at_ms,readonly"`

	AdvisorModel            string           `json:"advisorModel"            keel:"advisor_model,nullzero"`
	AdvisorCalls            int              `json:"advisorCalls"            keel:"advisor_calls"`
	AdvisorInputTokens      int              `json:"advisorInputTokens"      keel:"advisor_input_tokens"`
	AdvisorOutputTokens     int              `json:"advisorOutputTokens"     keel:"advisor_output_tokens"`
	AdvisorCacheReadTokens  int              `json:"advisorCacheReadTokens"  keel:"advisor_cache_read_tokens"`
	AdvisorCacheWriteTokens int              `json:"advisorCacheWriteTokens" keel:"advisor_cache_write_tokens"`
	AdvisorCostUSD          *decimal.Decimal `json:"advisorCostUsd"          keel:"advisor_cost_usd"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	// Batched is an attempt the provider answered through its batch API, at the
	// batch discount: CostUSD is already discounted.
	Batched bool `json:"batched" keel:"batched"`
}
