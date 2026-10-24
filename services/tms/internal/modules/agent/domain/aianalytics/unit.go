package aianalytics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type WorkUnit struct {
	bun.BaseModel `bun:"table:ai_work_units,alias:aiwu" json:"-"`
	keel.Table    `keel:"ai_work_units"`

	ID             string   `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	Source            Source          `json:"source"            bun:"source,type:VARCHAR(20),notnull" keel:"source"`
	AgentDefinitionID pulid.ID        `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero" keel:"agent_definition_id,nullzero"`
	ProviderID        pulid.ID        `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero" keel:"provider_id,nullzero"`
	Model             string          `json:"model"             bun:"model,type:VARCHAR(200),notnull" keel:"model"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''" keel:"model_key"`
	Task              aiprovider.Task `json:"task"              bun:"task,type:VARCHAR(100),notnull" keel:"task"`
	Outcome           Outcome         `json:"outcome"           bun:"outcome,type:VARCHAR(20),notnull" keel:"outcome"`
	DecisionRunID     pulid.ID        `json:"decisionRunId"     bun:"decision_run_id,type:VARCHAR(100),nullzero" keel:"decision_run_id,nullzero"`

	StartedAtMs  int64  `json:"startedAtMs"  bun:"started_at_ms,type:BIGINT,notnull" keel:"started_at_ms"`
	FinishedAt   int64  `json:"finishedAt"   bun:"finished_at,type:BIGINT,notnull" keel:"finished_at"`
	DurationMs   int64  `json:"durationMs"   bun:"duration_ms,type:BIGINT,notnull" keel:"duration_ms"`
	FirstTokenMs *int64 `json:"firstTokenMs" bun:"first_token_ms,type:BIGINT,nullzero" keel:"first_token_ms"`

	ModelCalls   int              `json:"modelCalls"   bun:"model_calls,type:INTEGER,notnull,default:0" keel:"model_calls"`
	FailedCalls  int              `json:"failedCalls"  bun:"failed_calls,type:INTEGER,notnull,default:0" keel:"failed_calls"`
	Failovers    int              `json:"failovers"    bun:"failovers,type:INTEGER,notnull,default:0" keel:"failovers"`
	InputTokens  int64            `json:"inputTokens"  bun:"input_tokens,type:BIGINT,notnull,default:0" keel:"input_tokens"`
	OutputTokens int64            `json:"outputTokens" bun:"output_tokens,type:BIGINT,notnull,default:0" keel:"output_tokens"`
	CostUSD      *decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),nullzero" keel:"cost_usd"`

	CacheReadTokens  int64 `json:"cacheReadTokens"  bun:"cache_read_tokens,type:BIGINT,notnull,default:0" keel:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cacheWriteTokens" bun:"cache_write_tokens,type:BIGINT,notnull,default:0" keel:"cache_write_tokens"`

	// The advisor's share of the unit: its consultations, its prompt (fresh,
	// cached and cache-written together) and output tokens, and its cost,
	// which CostUSD already includes.
	AdvisorCalls        int              `json:"advisorCalls"        bun:"advisor_calls,type:INTEGER,notnull,default:0" keel:"advisor_calls"`
	AdvisorPromptTokens int64            `json:"advisorPromptTokens" bun:"advisor_prompt_tokens,type:BIGINT,notnull,default:0" keel:"advisor_prompt_tokens"`
	AdvisorOutputTokens int64            `json:"advisorOutputTokens" bun:"advisor_output_tokens,type:BIGINT,notnull,default:0" keel:"advisor_output_tokens"`
	AdvisorCostUSD      *decimal.Decimal `json:"advisorCostUsd"      bun:"advisor_cost_usd,type:NUMERIC(14,6),nullzero" keel:"advisor_cost_usd"`

	ToolCalls         int `json:"toolCalls"         bun:"tool_calls,type:INTEGER,notnull,default:0" keel:"tool_calls"`
	ToolFailuresModel int `json:"toolFailuresModel" bun:"tool_failures_model,type:INTEGER,notnull,default:0" keel:"tool_failures_model"`
	ToolFailuresTool  int `json:"toolFailuresTool"  bun:"tool_failures_tool,type:INTEGER,notnull,default:0" keel:"tool_failures_tool"`

	RefreshStamp int64 `json:"refreshStamp" bun:"refresh_stamp,type:BIGINT,notnull" keel:"refresh_stamp"`
}

type UnitTool struct {
	bun.BaseModel `bun:"table:ai_work_unit_tools,alias:aiwt" json:"-"`
	keel.Table    `keel:"ai_work_unit_tools"`

	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	UnitID            string          `json:"unitId"            bun:"unit_id,pk,type:VARCHAR(100),notnull" keel:"unit_id,pk"`
	ToolName          string          `json:"toolName"          bun:"tool_name,pk,type:VARCHAR(200),notnull" keel:"tool_name,pk"`
	Cause             Cause           `json:"cause"             bun:"cause,pk,type:VARCHAR(30),notnull,default:''" keel:"cause,pk"`
	FinishedAt        int64           `json:"finishedAt"        bun:"finished_at,type:BIGINT,notnull" keel:"finished_at"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''" keel:"model_key"`
	ProviderID        string          `json:"providerId"        bun:"provider_id,type:VARCHAR(100),notnull,default:''" keel:"provider_id"`
	AgentDefinitionID string          `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull,default:''" keel:"agent_definition_id"`
	Task              aiprovider.Task `json:"task"              bun:"task,type:VARCHAR(100),notnull,default:''" keel:"task"`
	Calls             int             `json:"calls"             bun:"calls,type:INTEGER,notnull" keel:"calls"`
}

type ProviderHour struct {
	bun.BaseModel `bun:"table:ai_provider_hourly_stats,alias:aiph" json:"-"`
	keel.Table    `keel:"ai_provider_hourly_stats"`

	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	Hour              int64           `json:"hour"              bun:"hour,pk,type:BIGINT,notnull" keel:"hour,pk"`
	ProviderID        string          `json:"providerId"        bun:"provider_id,pk,type:VARCHAR(100),notnull,default:''" keel:"provider_id,pk"`
	Model             string          `json:"model"             bun:"model,pk,type:VARCHAR(200),notnull" keel:"model,pk"`
	ModelKey          string          `json:"modelKey"          bun:"model_key,type:VARCHAR(200),notnull,default:''" keel:"model_key"`
	Task              aiprovider.Task `json:"task"              bun:"task,pk,type:VARCHAR(100),notnull" keel:"task,pk"`
	AgentDefinitionID string          `json:"agentDefinitionId" bun:"agent_definition_id,pk,type:VARCHAR(100),notnull,default:''" keel:"agent_definition_id,pk"`
	ErrorClass        ProviderError   `json:"errorClass"        bun:"error_class,pk,type:VARCHAR(50),notnull,default:''" keel:"error_class,pk"`
	Calls             int             `json:"calls"             bun:"calls,type:INTEGER,notnull" keel:"calls"`
	FellToNext        int             `json:"fellToNext"        bun:"fell_to_next,type:INTEGER,notnull,default:0" keel:"fell_to_next"`
	FirstTokenHist    []int32         `json:"firstTokenHist"    bun:"first_token_hist,type:INTEGER[],array,notnull" keel:"first_token_hist"`
}

type RefreshState struct {
	bun.BaseModel `bun:"table:ai_analytics_refresh_state,alias:aiars" json:"-"`
	keel.Table    `keel:"ai_analytics_refresh_state"`

	OrganizationID   pulid.ID `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	RefreshedThrough int64    `json:"refreshedThrough" bun:"refreshed_through,type:BIGINT,notnull" keel:"refreshed_through"`
	RefreshedAt      int64    `json:"refreshedAt"      bun:"refreshed_at,type:BIGINT,notnull" keel:"refreshed_at"`
}
