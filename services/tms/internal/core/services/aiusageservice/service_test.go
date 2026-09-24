package aiusageservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUsage struct {
	repositories.AIUsageRepository

	summary     *repositories.AIUsageSummary
	failures    []repositories.AIUsageFailure
	summarised  repositories.AIUsageSummaryRequest
	failuresFor repositories.AIUsageFailuresRequest
}

func (f *fakeUsage) Summary(
	_ context.Context,
	req repositories.AIUsageSummaryRequest,
) (*repositories.AIUsageSummary, error) {
	f.summarised = req

	return f.summary, nil
}

func (f *fakeUsage) RecentFailures(
	_ context.Context,
	req repositories.AIUsageFailuresRequest,
) ([]repositories.AIUsageFailure, error) {
	f.failuresFor = req

	return f.failures, nil
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestSummary_BreaksUsageDownByFeature(t *testing.T) {
	t.Parallel()

	usage := &fakeUsage{summary: &repositories.AIUsageSummary{
		Totals: repositories.AIUsageTotals{Calls: 12, Failed: 1, CostUSD: "0.42", PricedCalls: 10},
		ByFeature: []repositories.AIUsageFeatureTotals{
			{
				Feature: aiusage.FeatureDocumentIntelligenceExtract,
				AIUsageTotals: repositories.AIUsageTotals{
					Calls:        7,
					Failed:       1,
					InputTokens:  9000,
					OutputTokens: 800,
					CostUSD:      "0.40",
					PricedCalls:  7,
					LatencyP50:   1200,
					LatencyP95:   4100,
				},
			},
			{
				AIUsageTotals: repositories.AIUsageTotals{
					Calls:       5,
					CostUSD:     "0.02",
					PricedCalls: 3,
				},
			},
		},
	}}
	svc := New(Params{Usage: usage})
	tenantInfo := tenant()
	since := timeutils.NowUnix() - 24*60*60

	summary, err := svc.Summary(t.Context(), tenantInfo, since)
	require.NoError(t, err)

	assert.Equal(t, tenantInfo, usage.summarised.TenantInfo)
	assert.Equal(t, since, usage.summarised.Since)
	require.Len(t, summary.ByFeature, 2)

	extract := summary.ByFeature[0]
	require.NotNil(t, extract.Feature)
	assert.Equal(t, aiusage.FeatureDocumentIntelligenceExtract, *extract.Feature)
	assert.Equal(t, 7, extract.Calls)
	assert.Equal(t, 1, extract.Failed)
	assert.Equal(t, int64(9000), extract.InputTokens)
	assert.Equal(t, int64(800), extract.OutputTokens)
	assert.Equal(t, "0.40", extract.CostUSD)
	assert.Equal(t, 7, extract.PricedCalls)
	assert.Equal(t, int64(1200), extract.LatencyP50Ms)
	assert.Equal(t, int64(4100), extract.LatencyP95Ms)

	unattributed := summary.ByFeature[1]
	assert.Nil(t, unattributed.Feature, "calls that named no feature are reported as such")
	assert.Equal(t, 5, unattributed.Calls)
	assert.Equal(t, 3, unattributed.PricedCalls)
}

func TestSummary_AnEmptyWindowHasAnEmptyBreakdown(t *testing.T) {
	t.Parallel()

	svc := New(Params{Usage: &fakeUsage{summary: &repositories.AIUsageSummary{}}})

	summary, err := svc.Summary(t.Context(), tenant(), 0)
	require.NoError(t, err)

	assert.NotNil(t, summary.ByFeature, "the list is empty, never null")
	assert.Empty(t, summary.ByFeature)
}
