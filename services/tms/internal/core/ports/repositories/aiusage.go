package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// AIUsageSummaryRequest bounds a summary to a tenant and a window.
type AIUsageSummaryRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

// AIUsageTotals is what a set of records adds up to. Latency percentiles are
// over successful calls only: a failure that was refused in a millisecond is
// not a fast answer.
type AIUsageTotals struct {
	Calls           int
	Failed          int
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	// CostUSD is the sum over the records that carried a cost. PricedCalls
	// says how many did, so a total can be labelled as partial.
	CostUSD     string
	PricedCalls int
	LatencyP50  int64
	LatencyP95  int64
}

// AIUsageProviderTotals is the same, for one provider and model.
type AIUsageProviderTotals struct {
	ProviderID   pulid.ID
	ProviderName string
	Model        string
	AIUsageTotals
}

type AIUsageSummary struct {
	Totals         AIUsageTotals
	ByProvider     []AIUsageProviderTotals
	RecentFailures []AIUsageFailure
}

// AIUsageCostRequest asks what one agent's calls have cost since an instant.
type AIUsageCostRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	Since        int64
}

// AIUsageCost is the answer: the sum over priced calls, and how many calls
// carried no price so the sum can be labelled as partial.
type AIUsageCost struct {
	CostUSD       decimal.Decimal
	Calls         int
	UnpricedCalls int
}

// AIUsageFailuresRequest asks for the newest failed attempts in a window.
type AIUsageFailuresRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Limit      int
}

// AIUsageFailure is one failed attempt: which provider and model, what kind
// of failure, and the provider's own words for it.
type AIUsageFailure struct {
	ProviderID   pulid.ID
	ProviderName string
	Model        string
	Task         string
	ErrorClass   string
	Message      string
	At           int64
}

type AIUsageEvaluationCostRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	SuiteRunID pulid.ID
}

type AIUsageSurfaceCostRequest struct {
	TenantInfo pagination.TenantInfo
	Surface    aiusage.Surface
	Since      int64
}

type AIUsageRepository interface {
	SurfaceCost(ctx context.Context, req AIUsageSurfaceCostRequest) (*AIUsageCost, error)
	EvaluationCost(ctx context.Context, req AIUsageEvaluationCostRequest) (*AIUsageCost, error)
	Create(ctx context.Context, record *aiusage.AIUsageRecord) error
	Summary(ctx context.Context, req AIUsageSummaryRequest) (*AIUsageSummary, error)
	RecentFailures(ctx context.Context, req AIUsageFailuresRequest) ([]AIUsageFailure, error)
	CostByDefinition(ctx context.Context, req AIUsageCostRequest) (*AIUsageCost, error)
}
