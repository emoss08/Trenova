package aiusage

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
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
)

// AIUsageRecord is one attempt to have a model answer: which provider, how long it
// took, what it consumed and what that consumption cost. One row per attempt
// rather than per turn, because a turn that fell through two providers before
// a third answered was three calls, three latencies and three bills.
type AIUsageRecord struct {
	bun.BaseModel `bun:"table:ai_usage_records,alias:aiu" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	ProviderID   pulid.ID        `json:"providerId"   bun:"provider_id,type:VARCHAR(100),nullzero"`
	ProviderKind aiprovider.Kind `json:"providerKind" bun:"provider_kind,type:VARCHAR(50),notnull"`
	Model        string          `json:"model"        bun:"model,type:VARCHAR(200),notnull"`
	Task         aiprovider.Task `json:"task"         bun:"task,type:VARCHAR(100),notnull"`
	Surface      Surface         `json:"surface"      bun:"surface,type:VARCHAR(50),notnull"`

	// Attribution: who and what the call was for, as far as the caller said.
	UserID            pulid.ID `json:"userId"            bun:"user_id,type:VARCHAR(100),nullzero"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	ThreadID          pulid.ID `json:"threadId"          bun:"thread_id,type:VARCHAR(100),nullzero"`
	RunID             pulid.ID `json:"runId"             bun:"run_id,type:VARCHAR(100),nullzero"`

	Succeeded  bool   `json:"succeeded"  bun:"succeeded,type:BOOLEAN,notnull"`
	ErrorClass string `json:"errorClass" bun:"error_class,type:VARCHAR(50),nullzero"`
	Streamed   bool   `json:"streamed"   bun:"streamed,type:BOOLEAN,notnull,default:false"`

	LatencyMs       int64 `json:"latencyMs"       bun:"latency_ms,type:BIGINT,notnull"`
	InputTokens     int   `json:"inputTokens"     bun:"input_tokens,type:INTEGER,notnull,default:0"`
	OutputTokens    int   `json:"outputTokens"    bun:"output_tokens,type:INTEGER,notnull,default:0"`
	ReasoningTokens int   `json:"reasoningTokens" bun:"reasoning_tokens,type:INTEGER,notnull,default:0"`
	// CostUSD is nil when the provider carries no pricing. It is not zero:
	// a call that cost something unknown must not be summed as free.
	CostUSD *decimal.Decimal `json:"costUsd" bun:"cost_usd,type:NUMERIC(14,6),nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
