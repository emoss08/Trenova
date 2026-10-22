package aianalytics

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type WorkUnit struct {
	bun.BaseModel `bun:"table:ai_work_units,alias:aiwu" json:"-"`

	ID             string   `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Source            Source          `json:"source"            bun:"source,type:VARCHAR(20),notnull"`
	AgentDefinitionID pulid.ID        `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	ProviderID        pulid.ID        `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero"`
	Model             string          `json:"model"             bun:"model,type:VARCHAR(200),notnull"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''"`
	Task              aiprovider.Task `json:"task"              bun:"task,type:VARCHAR(100),notnull"`
	Outcome           Outcome         `json:"outcome"           bun:"outcome,type:VARCHAR(20),notnull"`
	DecisionRunID     pulid.ID        `json:"decisionRunId"     bun:"decision_run_id,type:VARCHAR(100),nullzero"`

	StartedAtMs  int64  `json:"startedAtMs"  bun:"started_at_ms,type:BIGINT,notnull"`
	FinishedAt   int64  `json:"finishedAt"   bun:"finished_at,type:BIGINT,notnull"`
	DurationMs   int64  `json:"durationMs"   bun:"duration_ms,type:BIGINT,notnull"`
	FirstTokenMs *int64 `json:"firstTokenMs" bun:"first_token_ms,type:BIGINT,nullzero"`

	ModelCalls   int              `json:"modelCalls"   bun:"model_calls,type:INTEGER,notnull,default:0"`
	FailedCalls  int              `json:"failedCalls"  bun:"failed_calls,type:INTEGER,notnull,default:0"`
	Failovers    int              `json:"failovers"    bun:"failovers,type:INTEGER,notnull,default:0"`
	InputTokens  int64            `json:"inputTokens"  bun:"input_tokens,type:BIGINT,notnull,default:0"`
	OutputTokens int64            `json:"outputTokens" bun:"output_tokens,type:BIGINT,notnull,default:0"`
	CostUSD      *decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),nullzero"`

	CacheReadTokens  int64 `json:"cacheReadTokens"  bun:"cache_read_tokens,type:BIGINT,notnull,default:0"`
	CacheWriteTokens int64 `json:"cacheWriteTokens" bun:"cache_write_tokens,type:BIGINT,notnull,default:0"`

	// The advisor's share of the unit: its consultations, its prompt (fresh,
	// cached and cache-written together) and output tokens, and its cost,
	// which CostUSD already includes.
	AdvisorCalls        int              `json:"advisorCalls"        bun:"advisor_calls,type:INTEGER,notnull,default:0"`
	AdvisorPromptTokens int64            `json:"advisorPromptTokens" bun:"advisor_prompt_tokens,type:BIGINT,notnull,default:0"`
	AdvisorOutputTokens int64            `json:"advisorOutputTokens" bun:"advisor_output_tokens,type:BIGINT,notnull,default:0"`
	AdvisorCostUSD      *decimal.Decimal `json:"advisorCostUsd"      bun:"advisor_cost_usd,type:NUMERIC(14,6),nullzero"`

	ToolCalls         int `json:"toolCalls"         bun:"tool_calls,type:INTEGER,notnull,default:0"`
	ToolFailuresModel int `json:"toolFailuresModel" bun:"tool_failures_model,type:INTEGER,notnull,default:0"`
	ToolFailuresTool  int `json:"toolFailuresTool"  bun:"tool_failures_tool,type:INTEGER,notnull,default:0"`

	RefreshStamp int64 `json:"refreshStamp" bun:"refresh_stamp,type:BIGINT,notnull"`
}

type UnitTool struct {
	bun.BaseModel `bun:"table:ai_work_unit_tools,alias:aiwt" json:"-"`

	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	UnitID            string          `json:"unitId"            bun:"unit_id,pk,type:VARCHAR(100),notnull"`
	ToolName          string          `json:"toolName"          bun:"tool_name,pk,type:VARCHAR(200),notnull"`
	Cause             Cause           `json:"cause"             bun:"cause,pk,type:VARCHAR(30),notnull,default:''"`
	FinishedAt        int64           `json:"finishedAt"        bun:"finished_at,type:BIGINT,notnull"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''"`
	ProviderID        string          `json:"providerId"        bun:"provider_id,type:VARCHAR(100),notnull,default:''"`
	AgentDefinitionID string          `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull,default:''"`
	Task              aiprovider.Task `json:"task"              bun:"task,type:VARCHAR(100),notnull,default:''"`
	Calls             int             `json:"calls"             bun:"calls,type:INTEGER,notnull"`
}

type ProviderHour struct {
	bun.BaseModel `bun:"table:ai_provider_hourly_stats,alias:aiph" json:"-"`

	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Hour              int64           `json:"hour"              bun:"hour,pk,type:BIGINT,notnull"`
	ProviderID        string          `json:"providerId"        bun:"provider_id,pk,type:VARCHAR(100),notnull,default:''"`
	Model             string          `json:"model"             bun:"model,pk,type:VARCHAR(200),notnull"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''"`
	Task              aiprovider.Task `json:"task"              bun:"task,pk,type:VARCHAR(100),notnull"`
	AgentDefinitionID string          `json:"agentDefinitionId" bun:"agent_definition_id,pk,type:VARCHAR(100),notnull,default:''"`
	ErrorClass        ProviderError   `json:"errorClass"        bun:"error_class,pk,type:VARCHAR(50),notnull,default:''"`
	Calls             int             `json:"calls"             bun:"calls,type:INTEGER,notnull"`
	FellToNext        int             `json:"fellToNext"        bun:"fell_to_next,type:INTEGER,notnull,default:0"`
	FirstTokenHist    []int32         `json:"firstTokenHist"    bun:"first_token_hist,type:INTEGER[],array,notnull"`
}

type RefreshState struct {
	bun.BaseModel `bun:"table:ai_analytics_refresh_state,alias:aiars" json:"-"`

	OrganizationID   pulid.ID `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	RefreshedThrough int64    `json:"refreshedThrough" bun:"refreshed_through,type:BIGINT,notnull"`
	RefreshedAt      int64    `json:"refreshedAt"      bun:"refreshed_at,type:BIGINT,notnull"`
}
