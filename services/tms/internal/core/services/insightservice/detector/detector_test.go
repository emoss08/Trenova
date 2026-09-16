package detector_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDetector struct {
	key      string
	category insight.Category
	findings []detector.Finding
	err      error
}

func (d stubDetector) Key() string                     { return d.key }
func (d stubDetector) Category() insight.Category      { return d.category }
func (d stubDetector) Resource() permission.Resource   { return permission.ResourceShipment }
func (d stubDetector) Operation() permission.Operation { return permission.OpRead }
func (d stubDetector) Detect(context.Context, detector.Params) ([]detector.Finding, error) {
	return d.findings, d.err
}

func validFinding() detector.Finding {
	return detector.Finding{
		DedupeKey: "ontime-decline:cus_1",
		Subject:   "Acme Foods",
		Headline:  "On-time delivery for Acme Foods fell 11 points",
		Severity:  insight.SeverityWarning,
		Metrics: []insight.Metric{
			detector.Percent(
				"onTimePercent",
				"On-time delivery",
				decimal.NewFromFloat(82.4),
				insight.DirectionLowerIsWorse,
			),
		},
	}
}

func TestParamsWindowDays_ReportsThePeriodTheNumbersDescribe(t *testing.T) {
	t.Parallel()

	params := detector.Params{WindowStart: 1_700_000_000, WindowEnd: 1_700_000_000 + 30*86400}

	assert.Equal(t, 30, params.WindowDays())
}

// Wording derived from a window that makes no sense would read "over the last -3
// days". Zero is the honest answer and callers can omit the phrase.
func TestParamsWindowDays_IsZeroForAnInvertedOrEmptyWindow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, detector.Params{WindowStart: 100, WindowEnd: 100}.WindowDays())
	assert.Equal(t, 0, detector.Params{WindowStart: 200, WindowEnd: 100}.WindowDays())
}

func TestRegistry_KeepsDetectorsInTheOrderGiven(t *testing.T) {
	t.Parallel()

	registry := detector.NewRegistry(
		stubDetector{key: "first"},
		stubDetector{key: "second"},
		stubDetector{key: "third"},
	)

	keys := make([]string, 0, 3)
	for _, d := range registry.All() {
		keys = append(keys, d.Key())
	}

	assert.Equal(t, []string{"first", "second", "third"}, keys)
}

// Two detectors sharing a key would fight over the same dedupe space, each
// superseding the other's findings on every refresh. The second is refused.
func TestRegistry_RefusesADuplicateKey(t *testing.T) {
	t.Parallel()

	registry := detector.NewRegistry(
		stubDetector{key: "ontime-decline", category: insight.CategoryServiceQuality},
		stubDetector{key: "ontime-decline", category: insight.CategoryCashFlow},
	)

	require.Len(t, registry.All(), 1)

	found, ok := registry.Get("ontime-decline")
	require.True(t, ok)
	assert.Equal(t, insight.CategoryServiceQuality, found.Category())
}

func TestRegistry_IgnoresANilDetector(t *testing.T) {
	t.Parallel()

	registry := detector.NewRegistry(stubDetector{key: "real"}, nil)

	assert.Len(t, registry.All(), 1)
}

func TestFindingValidate_AcceptsACompleteFinding(t *testing.T) {
	t.Parallel()

	require.NoError(t, validFinding().Validate())
}

func TestFindingValidate_RequiresAnIdentityAndWording(t *testing.T) {
	t.Parallel()

	noKey := validFinding()
	noKey.DedupeKey = "  "
	require.ErrorIs(t, noKey.Validate(), detector.ErrMalformedFinding)

	noHeadline := validFinding()
	noHeadline.Headline = ""
	require.ErrorIs(t, noHeadline.Validate(), detector.ErrMalformedFinding)
}

// A finding with no numbers is an opinion. There is nothing for a reader to
// check, and nothing to stop narration inventing a figure to fill the gap.
func TestFindingValidate_RefusesAFindingWithNoNumbers(t *testing.T) {
	t.Parallel()

	finding := validFinding()
	finding.Metrics = nil

	err := finding.Validate()

	require.ErrorIs(t, err, detector.ErrMalformedFinding)
	assert.Contains(t, err.Error(), "carries no metrics")
}

func TestFindingValidate_RefusesAnInvalidSeverity(t *testing.T) {
	t.Parallel()

	finding := validFinding()
	finding.Severity = insight.Severity("Screaming")

	require.ErrorIs(t, finding.Validate(), detector.ErrMalformedFinding)
}

func TestFindingValidate_RefusesAnUnlabelledOrUnreadableMetric(t *testing.T) {
	t.Parallel()

	unlabelled := validFinding()
	unlabelled.Metrics[0].Label = ""
	require.ErrorIs(t, unlabelled.Validate(), detector.ErrMalformedFinding)

	unreadable := validFinding()
	unreadable.Metrics[0].Unit = insight.Unit("Furlongs")
	require.ErrorIs(t, unreadable.Validate(), detector.ErrMalformedFinding)
}

func TestFindingValidate_RefusesALinkThatLeavesTheApplication(t *testing.T) {
	t.Parallel()

	finding := validFinding()
	finding.Links = []insight.Link{{Label: "Look", Path: "https://evil.example"}}

	err := finding.Validate()

	require.ErrorIs(t, err, detector.ErrMalformedFinding)
	assert.Contains(t, err.Error(), "links outside the application")
}

func TestFindingMetricValues_ExposesEachNumberByKey(t *testing.T) {
	t.Parallel()

	finding := validFinding()
	finding.Metrics = append(
		finding.Metrics,
		detector.Count("lateStops", "Late stops", 14, insight.DirectionHigherIsWorse),
	)

	values := finding.MetricValues()

	require.Len(t, values, 2)
	assert.True(t, values["lateStops"].Equal(decimal.NewFromInt(14)))
	assert.True(t, values["onTimePercent"].Equal(decimal.NewFromFloat(82.4)))
}

func TestDedupeKey_IsStableAndDistinguishesSubjects(t *testing.T) {
	t.Parallel()

	first := detector.DedupeKey("detention-dwell", "loc_1")
	again := detector.DedupeKey("detention-dwell", "loc_1")
	other := detector.DedupeKey("detention-dwell", "loc_2")

	assert.Equal(t, first, again)
	assert.NotEqual(t, first, other)
}

// A subject containing the separator must not be able to forge another
// subject's key, which would make one finding silently supersede the other.
func TestDedupeKey_DoesNotLetASubjectForgeAnotherKey(t *testing.T) {
	t.Parallel()

	forged := detector.DedupeKey("detention", "loc_1:extra")
	genuine := detector.DedupeKey("detention", "loc_1", "extra")

	assert.NotEqual(t, genuine, forged)
}

func TestDedupeKey_SkipsEmptyParts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "detention:loc_1", detector.DedupeKey("detention", "", "  ", "loc_1"))
}

// The link has to open the table already showing the rows the detector counted.
// Anything less and the reader is told a number and left to reconstruct the
// query themselves.
func TestFilteredLink_EncodesFiltersTheTableCanParse(t *testing.T) {
	t.Parallel()

	link := detector.FilteredLink(
		"Late stops",
		detector.RouteShipments,
		14,
		detector.FieldFilter{Field: "customerId", Operator: detector.OpEq, Value: "cus_1"},
	)

	assert.Equal(t, 14, link.Count)
	assert.True(t, link.IsSafe())

	path, query, found := strings.Cut(link.Path, "?")
	require.True(t, found)
	assert.Equal(t, detector.RouteShipments, path)

	values, err := url.ParseQuery(query)
	require.NoError(t, err)

	var filters []detector.FieldFilter
	require.NoError(t, sonic.UnmarshalString(values.Get("fieldFilters"), &filters))
	require.Len(t, filters, 1)
	assert.Equal(t, "customerId", filters[0].Field)
	assert.Equal(t, "eq", filters[0].Operator)
	assert.Equal(t, "cus_1", filters[0].Value)
}

func TestFilteredLink_LinksToThePlainTableWhenThereIsNothingToFilter(t *testing.T) {
	t.Parallel()

	link := detector.FilteredLink("All shipments", detector.RouteShipments, 0)

	assert.Equal(t, detector.RouteShipments, link.Path)
	assert.True(t, link.IsSafe())
}

func TestWithBaseline_AttachesTheComparisonAndItsMeaning(t *testing.T) {
	t.Parallel()

	metric := detector.WithBaseline(
		detector.Percent(
			"onTimePercent",
			"On-time delivery",
			decimal.NewFromFloat(82.4),
			insight.DirectionLowerIsWorse,
		),
		decimal.NewFromFloat(93.1),
		"prior 30 days",
	)

	require.NotNil(t, metric.Baseline)
	assert.True(t, metric.Baseline.Equal(decimal.NewFromFloat(93.1)))
	assert.Equal(t, "prior 30 days", metric.BaselineLabel)
}

func TestSeverityFor_GradesAMetricWhereMoreIsWorse(t *testing.T) {
	t.Parallel()

	warning := decimal.NewFromInt(10)
	critical := decimal.NewFromInt(25)

	assert.Equal(t, insight.SeverityInfo,
		detector.SeverityFor(decimal.NewFromInt(9), warning, critical))
	// The thresholds are inclusive, so landing exactly on one grades up.
	assert.Equal(t, insight.SeverityWarning,
		detector.SeverityFor(decimal.NewFromInt(10), warning, critical))
	assert.Equal(t, insight.SeverityCritical,
		detector.SeverityFor(decimal.NewFromInt(25), warning, critical))
}

func TestSeverityForDescending_GradesAMetricWhereLessIsWorse(t *testing.T) {
	t.Parallel()

	warning := decimal.NewFromInt(95)
	critical := decimal.NewFromInt(85)

	assert.Equal(t, insight.SeverityInfo,
		detector.SeverityForDescending(decimal.NewFromInt(96), warning, critical))
	assert.Equal(t, insight.SeverityWarning,
		detector.SeverityForDescending(decimal.NewFromInt(95), warning, critical))
	assert.Equal(t, insight.SeverityCritical,
		detector.SeverityForDescending(decimal.NewFromInt(85), warning, critical))
}
