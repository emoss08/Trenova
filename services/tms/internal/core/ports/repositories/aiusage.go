package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	Totals     AIUsageTotals
	ByProvider []AIUsageProviderTotals
}

type AIUsageRepository interface {
	Create(ctx context.Context, record *aiusage.AIUsageRecord) error
	Summary(ctx context.Context, req AIUsageSummaryRequest) (*AIUsageSummary, error)
}
