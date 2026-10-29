package aianalytics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type WorkUnit struct {
	keel.Table `keel:"ai_work_units"`

	ID             string   `json:"id"             keel:"id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Source            Source      `json:"source"            keel:"source"`
	AgentDefinitionID pulid.ID    `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	ProviderID        pulid.ID    `json:"providerId"        keel:"provider_id,nullzero"`
	Model             string      `json:"model"             keel:"model"`
	ModelKey          string      `json:"modelKey"          keel:"model_key"`
	Task              aitask.Task `json:"task"              keel:"task"`
	Outcome           Outcome     `json:"outcome"           keel:"outcome"`
	DecisionRunID     pulid.ID    `json:"decisionRunId"     keel:"decision_run_id,nullzero"`

	StartedAtMs  int64  `json:"startedAtMs"  keel:"started_at_ms"`
	FinishedAt   int64  `json:"finishedAt"   keel:"finished_at"`
	DurationMs   int64  `json:"durationMs"   keel:"duration_ms"`
	FirstTokenMs *int64 `json:"firstTokenMs" keel:"first_token_ms"`

	ModelCalls   int              `json:"modelCalls"   keel:"model_calls"`
	FailedCalls  int              `json:"failedCalls"  keel:"failed_calls"`
	Failovers    int              `json:"failovers"    keel:"failovers"`
	InputTokens  int64            `json:"inputTokens"  keel:"input_tokens"`
	OutputTokens int64            `json:"outputTokens" keel:"output_tokens"`
	CostUSD      *decimal.Decimal `json:"costUsd"      keel:"cost_usd"`

	CacheReadTokens  int64 `json:"cacheReadTokens"  keel:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cacheWriteTokens" keel:"cache_write_tokens"`

	// The advisor's share of the unit: its consultations, its prompt (fresh,
	// cached and cache-written together) and output tokens, and its cost,
	// which CostUSD already includes.
	AdvisorCalls        int              `json:"advisorCalls"        keel:"advisor_calls"`
	AdvisorPromptTokens int64            `json:"advisorPromptTokens" keel:"advisor_prompt_tokens"`
	AdvisorOutputTokens int64            `json:"advisorOutputTokens" keel:"advisor_output_tokens"`
	AdvisorCostUSD      *decimal.Decimal `json:"advisorCostUsd"      keel:"advisor_cost_usd"`

	ToolCalls         int `json:"toolCalls"         keel:"tool_calls"`
	ToolFailuresModel int `json:"toolFailuresModel" keel:"tool_failures_model"`
	ToolFailuresTool  int `json:"toolFailuresTool"  keel:"tool_failures_tool"`

	RefreshStamp int64 `json:"refreshStamp" keel:"refresh_stamp"`
}

type UnitTool struct {
	keel.Table `keel:"ai_work_unit_tools"`

	OrganizationID    pulid.ID    `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID    `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	UnitID            string      `json:"unitId"            keel:"unit_id,pk"`
	ToolName          string      `json:"toolName"          keel:"tool_name,pk"`
	Cause             Cause       `json:"cause"             keel:"cause,pk"`
	FinishedAt        int64       `json:"finishedAt"        keel:"finished_at"`
	ModelKey          string      `json:"modelKey"          keel:"model_key"`
	ProviderID        string      `json:"providerId"        keel:"provider_id"`
	AgentDefinitionID string      `json:"agentDefinitionId" keel:"agent_definition_id"`
	Task              aitask.Task `json:"task"              keel:"task"`
	Calls             int         `json:"calls"             keel:"calls"`
}

type ProviderHour struct {
	keel.Table `keel:"ai_provider_hourly_stats"`

	OrganizationID    pulid.ID      `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	Hour              int64         `json:"hour"              keel:"hour,pk"`
	ProviderID        string        `json:"providerId"        keel:"provider_id,pk"`
	Model             string        `json:"model"             keel:"model,pk"`
	ModelKey          string        `json:"modelKey"          keel:"model_key"`
	Task              aitask.Task   `json:"task"              keel:"task,pk"`
	AgentDefinitionID string        `json:"agentDefinitionId" keel:"agent_definition_id,pk"`
	ErrorClass        ProviderError `json:"errorClass"        keel:"error_class,pk"`
	Calls             int           `json:"calls"             keel:"calls"`
	FellToNext        int           `json:"fellToNext"        keel:"fell_to_next"`
	FirstTokenHist    []int32       `json:"firstTokenHist"    keel:"first_token_hist"`
}

type RefreshState struct {
	keel.Table `keel:"ai_analytics_refresh_state"`

	OrganizationID   pulid.ID `json:"organizationId"   keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	RefreshedThrough int64    `json:"refreshedThrough" keel:"refreshed_through"`
	RefreshedAt      int64    `json:"refreshedAt"      keel:"refreshed_at"`
}
