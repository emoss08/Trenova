package detectors_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detectors"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMetrics returns whatever a test sets, so a detector's thresholds can be
// exercised without a database. The whole point of splitting the queries behind
// a port is that these judgement calls become testable.
type fakeMetrics struct {
	onTime     []repositories.CustomerOnTimeRow
	unbilled   []repositories.UnbilledShipmentRow
	detention  []repositories.UnbilledDetentionRow
	emptyMiles []repositories.CustomerEmptyMilesRow
	credential []repositories.ExpiringCredentialRow
	err        error

	lastWindow      repositories.InsightWindowRequest
	lastUnbilledReq repositories.UnbilledShipmentsRequest
	lastCredential  repositories.ExpiringCredentialsRequest
}

func (m *fakeMetrics) CustomerOnTimeComparison(
	_ context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.CustomerOnTimeRow, error) {
	m.lastWindow = req

	return m.onTime, m.err
}

func (m *fakeMetrics) UnbilledDeliveredShipments(
	_ context.Context,
	req repositories.UnbilledShipmentsRequest,
) ([]repositories.UnbilledShipmentRow, error) {
	m.lastUnbilledReq = req

	return m.unbilled, m.err
}

func (m *fakeMetrics) UnbilledDetention(
	_ context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.UnbilledDetentionRow, error) {
	m.lastWindow = req

	return m.detention, m.err
}

func (m *fakeMetrics) CustomerEmptyMiles(
	_ context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.CustomerEmptyMilesRow, error) {
	m.lastWindow = req

	return m.emptyMiles, m.err
}

func (m *fakeMetrics) ExpiringCredentials(
	_ context.Context,
	req repositories.ExpiringCredentialsRequest,
) ([]repositories.ExpiringCredentialRow, error) {
	m.lastCredential = req

	return m.credential, m.err
}

const (
	windowEnd   = int64(1_800_000_000)
	windowStart = windowEnd - 30*86400
)

func testParams() detector.Params {
	return detector.Params{WindowStart: windowStart, WindowEnd: windowEnd, Timezone: "UTC"}
}

// Every finding a detector emits is stored and shown, so it has to satisfy the
// framework's own contract or it would be dropped at write time.
func requireValidFindings(t *testing.T, findings []detector.Finding) {
	t.Helper()

	for _, finding := range findings {
		require.NoError(t, finding.Validate(), finding.DedupeKey)
	}
}

func TestOnTimeDecline_ReportsACustomerWhoseServiceFell(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{onTime: []repositories.CustomerOnTimeRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Acme Foods",
		CurrentTotal:  100,
		CurrentOnTime: 82,
		PriorTotal:    100,
		PriorOnTime:   95,
	}}}

	findings, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	requireValidFindings(t, findings)

	require.Len(t, findings, 1)
	assert.Equal(t, "Acme Foods", findings[0].Subject)
	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)

	onTime, ok := findingMetric(findings[0], "onTimePercent")
	require.True(t, ok)
	assert.Equal(t, "82", onTime.Value.String())
	require.NotNil(t, onTime.Baseline)
	assert.Equal(t, "95", onTime.Baseline.String())
}

// A customer with four deliveries can go from 100% to 75% because one truck hit
// traffic. Putting that on a home screen teaches people to ignore the panel.
func TestOnTimeDecline_IgnoresACustomerWithTooLittleVolume(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{onTime: []repositories.CustomerOnTimeRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Tiny Shipper",
		CurrentTotal:  4,
		CurrentOnTime: 2,
		PriorTotal:    4,
		PriorOnTime:   4,
	}}}

	findings, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

// A customer who shipped nothing last month has not declined, they have
// started. Comparing against an empty period invents a 100-point drop.
func TestOnTimeDecline_IgnoresACustomerWithNoPriorPeriod(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{onTime: []repositories.CustomerOnTimeRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "New Customer",
		CurrentTotal:  40,
		CurrentOnTime: 20,
		PriorTotal:    0,
		PriorOnTime:   0,
	}}}

	findings, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestOnTimeDecline_IgnoresACustomerWhoImproved(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{onTime: []repositories.CustomerOnTimeRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Improving Co",
		CurrentTotal:  100,
		CurrentOnTime: 97,
		PriorTotal:    100,
		PriorOnTime:   88,
	}}}

	findings, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestOnTimeDecline_GradesASmallerSlipAsWarning(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{onTime: []repositories.CustomerOnTimeRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Acme Foods",
		CurrentTotal:  100,
		CurrentOnTime: 89,
		PriorTotal:    100,
		PriorOnTime:   95,
	}}}

	findings, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	require.Len(t, findings, 1)
	assert.Equal(t, insight.SeverityWarning, findings[0].Severity)
}

func TestOnTimeDecline_SurfacesARepositoryFailureRatherThanReportingNoProblems(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{err: errors.New("query failed")}

	_, err := detectors.NewOnTimeDecline(metrics).Detect(t.Context(), testParams())

	require.Error(t, err)
}

func TestUnbilledAging_ReportsRevenueSittingStill(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{unbilled: []repositories.UnbilledShipmentRow{{
		CustomerID:     pulid.MustNew("cus_"),
		CustomerName:   "Borden Freight",
		ShipmentCount:  14,
		TotalAmount:    "31240.00",
		OldestDelivery: windowEnd - 21*86400,
	}}}

	findings, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	requireValidFindings(t, findings)

	require.Len(t, findings, 1)
	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)

	age, ok := findingMetric(findings[0], "oldestAgeDays")
	require.True(t, ok)
	assert.Equal(t, "21", age.Value.String())
}

// Freight delivered this morning is not late to bill, so the query is asked for
// shipments older than the cutoff rather than everything unbilled.
func TestUnbilledAging_AsksOnlyForShipmentsOldEnoughToBeLate(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{}

	_, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Less(t, metrics.lastUnbilledReq.DeliveredBefore, windowEnd)
}

// One stuck shipment is a task. A pattern is an insight.
func TestUnbilledAging_IgnoresASingleStuckShipment(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{unbilled: []repositories.UnbilledShipmentRow{{
		CustomerID:     pulid.MustNew("cus_"),
		CustomerName:   "Borden Freight",
		ShipmentCount:  1,
		TotalAmount:    "90000.00",
		OldestDelivery: windowEnd - 40*86400,
	}}}

	findings, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestUnbilledAging_IgnoresAnImmaterialAmount(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{unbilled: []repositories.UnbilledShipmentRow{{
		CustomerID:     pulid.MustNew("cus_"),
		CustomerName:   "Borden Freight",
		ShipmentCount:  6,
		TotalAmount:    "410.00",
		OldestDelivery: windowEnd - 9*86400,
	}}}

	findings, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

// A sum that will not parse must produce no card. Reporting zero would read as
// "no money here", which is the opposite of the truth.
func TestUnbilledAging_SaysNothingRatherThanReportingZeroForAnUnreadableAmount(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{unbilled: []repositories.UnbilledShipmentRow{{
		CustomerID:     pulid.MustNew("cus_"),
		CustomerName:   "Borden Freight",
		ShipmentCount:  9,
		TotalAmount:    "not a number",
		OldestDelivery: windowEnd - 9*86400,
	}}}

	findings, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

// A delivery dated in the future is a clock problem. "-3 days old" on a card
// puts that confusion in front of a person instead of absorbing it.
func TestUnbilledAging_NeverReportsANegativeAge(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{unbilled: []repositories.UnbilledShipmentRow{{
		CustomerID:     pulid.MustNew("cus_"),
		CustomerName:   "Borden Freight",
		ShipmentCount:  9,
		TotalAmount:    "31240.00",
		OldestDelivery: windowEnd + 5*86400,
	}}}

	findings, err := detectors.NewUnbilledAging(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	require.Len(t, findings, 1)

	age, ok := findingMetric(findings[0], "oldestAgeDays")
	require.True(t, ok)
	assert.False(t, age.Value.IsNegative())
}

func TestUnbilledDetention_ReportsMoneyLeftOnTheTable(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{detention: []repositories.UnbilledDetentionRow{{
		LocationID:        pulid.MustNew("loc_"),
		LocationName:      "Dallas Distribution Center",
		OccurrenceCount:   19,
		UnbilledAmount:    "9840.00",
		TotalDwellMinutes: 5400,
	}}}

	findings, err := detectors.NewUnbilledDetention(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	requireValidFindings(t, findings)

	require.Len(t, findings, 1)
	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)

	dwell, ok := findingMetric(findings[0], "dwellHours")
	require.True(t, ok)
	assert.Equal(t, "90", dwell.Value.String())
}

// One bad afternoon at a consignee must not become a standing card about that
// location.
func TestUnbilledDetention_IgnoresAnIsolatedOccurrence(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{detention: []repositories.UnbilledDetentionRow{{
		LocationID:        pulid.MustNew("loc_"),
		LocationName:      "Dallas Distribution Center",
		OccurrenceCount:   1,
		UnbilledAmount:    "9840.00",
		TotalDwellMinutes: 600,
	}}}

	findings, err := detectors.NewUnbilledDetention(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestEmptyMiles_ReportsACustomerRunningEmptierThanTheBook(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{emptyMiles: []repositories.CustomerEmptyMilesRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Northwind Retail",
		EmptyMiles:    "28000",
		TotalMiles:    "100000",
		MoveCount:     140,
		OrgEmptyMiles: "120000",
		OrgTotalMiles: "1000000",
	}}}

	findings, err := detectors.NewEmptyMiles(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	requireValidFindings(t, findings)

	require.Len(t, findings, 1)
	// 28% against a 12% book is a 16-point spread: bad, but short of the
	// threshold where a card should be shouting.
	assert.Equal(t, insight.SeverityWarning, findings[0].Severity)

	percent, ok := findingMetric(findings[0], "emptyMilePercent")
	require.True(t, ok)
	assert.Equal(t, "28", percent.Value.String())
	require.NotNil(t, percent.Baseline)
	assert.Equal(t, "12", percent.Baseline.String())
}

func TestEmptyMiles_EscalatesAnExtremeSpread(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{emptyMiles: []repositories.CustomerEmptyMilesRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Northwind Retail",
		EmptyMiles:    "35000",
		TotalMiles:    "100000",
		MoveCount:     140,
		OrgEmptyMiles: "120000",
		OrgTotalMiles: "1000000",
	}}}

	findings, err := detectors.NewEmptyMiles(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	require.Len(t, findings, 1)
	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)
}

// A dedicated fleet lives at 8% empty and a brokered operation at 25%, and
// neither is a problem in itself. Only the spread against this operation's own
// book is news.
func TestEmptyMiles_IgnoresAHighRatioThatMatchesTheFleet(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{emptyMiles: []repositories.CustomerEmptyMilesRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Spot Market Co",
		EmptyMiles:    "26000",
		TotalMiles:    "100000",
		MoveCount:     140,
		OrgEmptyMiles: "250000",
		OrgTotalMiles: "1000000",
	}}}

	findings, err := detectors.NewEmptyMiles(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestEmptyMiles_IgnoresACustomerWithTooFewMoves(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{emptyMiles: []repositories.CustomerEmptyMilesRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Occasional Shipper",
		EmptyMiles:    "600",
		TotalMiles:    "1000",
		MoveCount:     2,
		OrgEmptyMiles: "120000",
		OrgTotalMiles: "1000000",
	}}}

	findings, err := detectors.NewEmptyMiles(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

// A zero denominator must produce no finding rather than a confident 0%.
func TestEmptyMiles_SaysNothingWhenThereAreNoMilesToDivideBy(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{emptyMiles: []repositories.CustomerEmptyMilesRow{{
		CustomerID:    pulid.MustNew("cus_"),
		CustomerName:  "Northwind Retail",
		EmptyMiles:    "0",
		TotalMiles:    "0",
		MoveCount:     40,
		OrgEmptyMiles: "120000",
		OrgTotalMiles: "1000000",
	}}}

	findings, err := detectors.NewEmptyMiles(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

func TestCredentialExpiry_ReportsAClusterComingDue(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{credential: []repositories.ExpiringCredentialRow{{
		CredentialTypeID:   pulid.MustNew("wct_"),
		CredentialTypeName: "DOT Medical Card",
		IsRequired:         true,
		WorkerCount:        11,
		EarliestExpiry:     windowEnd + 6*86400,
	}}}

	findings, err := detectors.NewCredentialExpiry(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	requireValidFindings(t, findings)

	require.Len(t, findings, 1)
	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)

	days, ok := findingMetric(findings[0], "daysToFirstExpiry")
	require.True(t, ok)
	assert.Equal(t, "6", days.Value.String())
}

// A driver whose medical card expired yesterday cannot legally run today. That
// must not be averaged into something calm.
func TestCredentialExpiry_EscalatesWhenARequiredCredentialHasAlreadyLapsed(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{credential: []repositories.ExpiringCredentialRow{{
		CredentialTypeID:   pulid.MustNew("wct_"),
		CredentialTypeName: "DOT Medical Card",
		IsRequired:         true,
		WorkerCount:        2,
		AlreadyExpired:     1,
		EarliestExpiry:     windowEnd - 2*86400,
	}}}

	findings, err := detectors.NewCredentialExpiry(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	require.Len(t, findings, 1)

	assert.Equal(t, insight.SeverityCritical, findings[0].Severity)
	assert.Contains(t, findings[0].Headline, "already expired")

	expired, ok := findingMetric(findings[0], "alreadyExpired")
	require.True(t, ok)
	assert.Equal(t, "1", expired.Value.String())
}

// Somebody has to act before the date on a required credential, and an Info
// card is the one nobody reads.
func TestCredentialExpiry_NeverGradesARequiredCredentialAsMerelyInformational(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{credential: []repositories.ExpiringCredentialRow{{
		CredentialTypeID:   pulid.MustNew("wct_"),
		CredentialTypeName: "Hazmat Endorsement",
		IsRequired:         true,
		WorkerCount:        1,
		EarliestExpiry:     windowEnd + 40*86400,
	}}}

	findings, err := detectors.NewCredentialExpiry(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)
	require.Len(t, findings, 1)

	assert.Equal(t, insight.SeverityWarning, findings[0].Severity)
}

// One optional credential expiring is a calendar entry, not a home-screen card.
func TestCredentialExpiry_IgnoresALoneOptionalCredential(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{credential: []repositories.ExpiringCredentialRow{{
		CredentialTypeID:   pulid.MustNew("wct_"),
		CredentialTypeName: "Forklift Certification",
		IsRequired:         false,
		WorkerCount:        1,
		EarliestExpiry:     windowEnd + 20*86400,
	}}}

	findings, err := detectors.NewCredentialExpiry(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Empty(t, findings)
}

// The horizon has to look forward from now, not over the reporting window, or a
// credential expiring next month would never be seen.
func TestCredentialExpiry_LooksForwardFromTheEndOfTheWindow(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{}

	_, err := detectors.NewCredentialExpiry(metrics).Detect(t.Context(), testParams())
	require.NoError(t, err)

	assert.Equal(t, windowEnd, metrics.lastCredential.From)
	assert.Greater(t, metrics.lastCredential.Through, windowEnd)
}

// Two detectors sharing a dedupe key would supersede each other's findings on
// every refresh, so the keys must be distinct across the whole registry.
func TestDetectorKeys_AreDistinct(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetrics{}
	registry := detector.NewRegistry(
		detectors.NewOnTimeDecline(metrics),
		detectors.NewUnbilledAging(metrics),
		detectors.NewUnbilledDetention(metrics),
		detectors.NewEmptyMiles(metrics),
		detectors.NewCredentialExpiry(metrics),
	)

	assert.Len(t, registry.All(), 5)
}

func findingMetric(finding detector.Finding, key string) (insight.Metric, bool) {
	for _, metric := range finding.Metrics {
		if metric.Key == key {
			return metric, true
		}
	}

	return insight.Metric{}, false
}
