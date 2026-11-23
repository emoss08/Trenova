package aiusage

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const DailyRetentionDays = 25 * 31

// AI usage by day: model calls added up by the organization's day, by the person they were made
// for, the feature that made them, the surface and the model, with their tokens and estimated
// cost. Calls whose provider carries no price are counted apart, so an unpriced call never reads
// as free.
type DailyUsage struct {
	keel.Table `keel:"ai_usage_daily,report=Desk,label=AI Usage by Day,plural=AI Usage by Day"`

	ID             pulid.ID `json:"id"             keel:"id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// The day in the organization's time zone
	UsageDate int64    `json:"usageDate"    keel:"usage_date,label=Day"`
	UserID    pulid.ID `json:"userId"       keel:"user_id,nullzero"`
	// The product feature that made the calls; empty for calls that named none
	Feature Feature `json:"feature"      keel:"feature,nullzero"`
	// Which part of the system made the calls
	Surface Surface `json:"surface"      keel:"surface"`
	// The model that answered, as the provider named it
	Model        string          `json:"model"        keel:"model"`
	ProviderKind aiprovider.Kind `json:"providerKind" keel:"provider_kind,noreport"`

	// Model calls, every attempt included
	Calls int64 `json:"calls"            keel:"calls,format=count"`
	// Calls that failed
	FailedCalls int64 `json:"failedCalls"      keel:"failed_calls,format=count"`
	// Calls whose provider carries no price; their cost is not in the estimate
	UnpricedCalls int64 `json:"unpricedCalls"    keel:"unpriced_calls,format=count,label=Calls Without a Price"`
	// Input tokens the provider's cache did not serve
	InputTokens int64 `json:"inputTokens"      keel:"input_tokens,format=count,label=Uncached Input Tokens"`
	// Input tokens served from the provider's prompt cache
	CacheReadTokens int64 `json:"cacheReadTokens"  keel:"cache_read_tokens,format=count"`
	// Input tokens written to the provider's prompt cache
	CacheWriteTokens int64 `json:"cacheWriteTokens" keel:"cache_write_tokens,format=count"`
	// Tokens the model wrote, reasoning included
	OutputTokens int64 `json:"outputTokens"     keel:"output_tokens,format=count"`
	// Of the output, the tokens the model spent reasoning
	ReasoningTokens int64 `json:"reasoningTokens"  keel:"reasoning_tokens,format=count"`
	// Tokens an advisor model read and wrote for these calls
	AdvisorTokens int64 `json:"advisorTokens"    keel:"advisor_tokens,format=count"`
	// Every token billed: uncached input, cache reads, cache writes, output and the advisor's
	TotalTokens int64 `json:"totalTokens"      keel:"total_tokens,format=count"`
	// Estimated cost in US dollars of the calls that carried a price, the advisor's included
	CostUSD decimal.Decimal `json:"costUsd"          keel:"cost_usd,format=money,label=Estimated Cost"`

	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,noreport"`

	// The person the calls were made for
	User *tenant.Person `json:"-" keel:"belongs:user_id,label=Person"`
}
