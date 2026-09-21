package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInsights struct {
	browsed  serviceports.BrowseInsightsRequest
	detailed serviceports.GetInsightDetailRequest
	items    []*insight.Insight
	detail   *serviceports.InsightDetail
}

func (f *fakeInsights) List(
	_ context.Context,
	req serviceports.BrowseInsightsRequest,
) (*pagination.ListResult[*insight.Insight], error) {
	f.browsed = req

	return &pagination.ListResult[*insight.Insight]{Items: f.items, Total: len(f.items)}, nil
}

func (f *fakeInsights) GetDetail(
	_ context.Context,
	req serviceports.GetInsightDetailRequest,
) (*serviceports.InsightDetail, error) {
	f.detailed = req

	return f.detail, nil
}

func testInsight() *insight.Insight {
	return &insight.Insight{
		ID:             pulid.MustNew("inst_"),
		Category:       insight.CategoryServiceQuality,
		Severity:       insight.SeverityWarning,
		Status:         insight.StatusActive,
		Subject:        "Acme Foods",
		Headline:       "On-time delivery for Acme Foods fell to 71%",
		Recommendation: "Review the lane's transit times.",
		Metrics: []insight.Metric{{
			Key:   "ontime",
			Label: "On-time rate",
			Value: decimal.NewFromInt(71),
			Unit:  insight.UnitPercent,
		}},
		Links:       []insight.Link{{Label: "Late shipments", Path: "/shipments?late=1", Count: 9}},
		WindowStart: 1_797_000_000,
		WindowEnd:   1_799_592_000,
		DetectedAt:  1_799_592_000,
		StaleAt:     1_700_000_000,
	}
}

// The default is what a person sees on the insights page: active findings,
// filtered to what this actor may see. The actor rides on the request so an
// agent is checked as an agent.
func TestListInsights_DefaultsToActiveAndCarriesTheActor(t *testing.T) {
	t.Parallel()

	insights := &fakeInsights{items: []*insight.Insight{testInsight()}}
	tool := newListInsightsTool(insights)
	params := testParams(map[string]any{})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, []insight.Status{insight.StatusActive}, insights.browsed.Statuses)
	assert.Same(t, params.Actor, insights.browsed.Actor)
	assert.Equal(t, params.OrganizationID, insights.browsed.TenantInfo.OrgID)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Equal(t, 1, outcome.Count)
	assert.Contains(t, outcome.SearchedFor, "status Active")

	rows, ok := outcome.Items.([]insightRow)
	require.True(t, ok)
	assert.Equal(t, "Acme Foods", rows[0].Subject)
	assert.Equal(t, "On-time rate", rows[0].Metrics[0].Label)
	assert.Equal(t, 9, rows[0].Links[0].Count)
	assert.True(t, rows[0].Stale, "a finding past its stale-at is marked so")
}

func TestListInsights_NarrowsByCategorySeverityAndStatus(t *testing.T) {
	t.Parallel()

	insights := &fakeInsights{}
	tool := newListInsightsTool(insights)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"category": "CashFlow",
		"severity": "Critical",
		"status":   "Dismissed",
		"limit":    5,
	}))
	require.NoError(t, err)

	assert.Equal(t, []insight.Category{insight.CategoryCashFlow}, insights.browsed.Categories)
	assert.Equal(t, []insight.Severity{insight.SeverityCritical}, insights.browsed.Severities)
	assert.Equal(t, []insight.Status{insight.StatusDismissed}, insights.browsed.Statuses)
	assert.Equal(t, 5, insights.browsed.Limit)

	outcome := result.(searchOutcome)
	assert.Contains(t, outcome.SearchedFor, "category CashFlow")
	assert.Contains(t, outcome.Note, "No insights matched")
}

func TestListInsights_RefusesAnUnknownCategory(t *testing.T) {
	t.Parallel()

	_, err := newListInsightsTool(&fakeInsights{}).Query(
		t.Context(),
		testParams(map[string]any{"category": "Weather"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ServiceQuality")
}

func TestGetInsight_ReturnsTheRuleAndTheTrend(t *testing.T) {
	t.Parallel()

	current := testInsight()
	earlier := testInsight()
	earlier.Status = insight.StatusSuperseded
	earlier.DetectedAt = current.DetectedAt - 86_400
	earlier.Metrics[0].Value = decimal.NewFromInt(78)

	insights := &fakeInsights{detail: &serviceports.InsightDetail{
		Insight: current,
		History: []*insight.Insight{earlier},
		Explanation: serviceports.InsightExplanation{
			Measures:  "on-time deliveries over the window",
			Threshold: "below 80%",
			Excludes:  "cancelled shipments",
		},
	}}
	tool := newGetInsightTool(insights)
	params := testParams(map[string]any{"insightId": current.ID.String()})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, current.ID, insights.detailed.ID)
	assert.Same(t, params.Actor, insights.detailed.Actor)

	row, ok := result.(insightDetailRow)
	require.True(t, ok)
	assert.Equal(t, current.Headline, row.Headline)
	assert.Equal(t, "below 80%", row.Rule.Threshold)
	require.Len(t, row.History, 1)
	assert.Equal(t, "Superseded", row.History[0].Status)
	assert.True(t, decimal.NewFromInt(78).Equal(row.History[0].Metrics[0].Value))
}

func TestGetInsight_RequiresAnID(t *testing.T) {
	t.Parallel()

	_, err := newGetInsightTool(&fakeInsights{}).Query(t.Context(), testParams(map[string]any{}))
	require.Error(t, err)
}
