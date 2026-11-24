package billingboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var chicago = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}

	return loc
}()

func at(year int, month time.Month, day, hour int) int64 {
	return time.Date(year, month, day, hour, 0, 0, 0, chicago).Unix()
}

func TestAnExceptionReceiptReadsItsAmountAndTheProductsReasonAndNoMatch(t *testing.T) {
	t.Parallel()

	reading := receivedReading(context.Background(), &receiptRow{
		Status:          "Exception",
		AmountMinor:     945_000,
		ReceiptDate:     at(2026, time.November, 20, 9),
		ExceptionReason: "No unique customer payment match found for bank receipt",
	})

	assert.Equal(t, boardcatalog.ValueMoney, reading.Value.Kind)
	assert.Equal(t, "9450", reading.Value.Number.String())
	assert.Equal(t, "{0}", reading.Standing.Text)
	require.NotNil(t, reading.Fact)
	assert.Equal(t, "{0}", reading.Fact.Text)
	require.NotNil(t, reading.Fact.Value)
	assert.Equal(t, boardcatalog.ValueText, reading.Fact.Value.Kind)
	assert.Equal(t, "No unique customer payment match found for bank receipt", reading.Fact.Value.Text,
		"the reason is a value, so a payer's {0} in it is never read as the template")
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	assert.Nil(t, reading.Bounded, "no candidate is stored yet, so nothing is drawn against one")
}

func TestAMatchedReceiptSaysWhenItWasMatchedAndAnImportedOneWhenItCameIn(t *testing.T) {
	t.Parallel()

	matchedAt := at(2026, time.November, 21, 10)
	matched := receivedReading(context.Background(), &receiptRow{
		Status: "Matched", AmountMinor: 100, ReceiptDate: at(2026, time.November, 20, 9),
		MatchedAt: &matchedAt,
	})
	require.NotNil(t, matched.Fact)
	assert.Equal(t, "Matched {0}", matched.Fact.Text)
	assert.Equal(t, matchedAt, matched.Fact.At)
	assert.True(t, matched.Fact.Day)
	assert.Empty(t, matched.Tone)

	received := at(2026, time.November, 22, 9)
	imported := receivedReading(context.Background(), &receiptRow{
		Status: "Imported", AmountMinor: 100, ReceiptDate: received,
	})
	assert.Equal(t, "Received {0}", imported.Fact.Text)
	assert.Equal(t, received, imported.Fact.At)
}

func TestAReceiptIsNamedByItsReferenceAndItsStatusCarriesTheProductsTone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	day := at(2026, time.November, 20, 23)

	exception := receiptSummary(ctx, &receiptRow{
		Reference: " ACH 7731090 ", Status: "Exception", ReceiptDate: day,
	})
	assert.Equal(t, "ACH 7731090", exception.Label)
	assert.Empty(t, exception.Subtitle, "the day is the fact's, which the client never cuts")
	assert.Equal(t, "Exception", exception.Status)
	assert.Equal(t, boardcatalog.ToneBad, exception.StatusTone)

	matched := receiptSummary(ctx, &receiptRow{
		Status: "Matched", ReceiptDate: day, Customer: "Kessler Farms",
	})
	assert.Equal(t, "Receipt with no reference", matched.Label)
	assert.Equal(t, "Kessler Farms", matched.Subtitle)
	assert.Equal(t, boardcatalog.ToneOK, matched.StatusTone)

	imported := receiptSummary(ctx, &receiptRow{Status: "Imported", ReceiptDate: day})
	assert.Equal(t, "Imported", imported.Status)
	assert.Empty(t, imported.StatusTone)
	assert.Empty(t, imported.Subtitle)
}

func TestADayInThisYearIsWordedWithoutTheYear(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	day := at(2026, time.November, 20, 9)
	inYear := at(2026, time.December, 1, 9)
	nextYear := at(2027, time.January, 3, 9)

	assert.Equal(t, "Nov 20", dayText(ctx, day, chicago, inYear))
	assert.Equal(t, "Nov 20, 2026", dayText(ctx, day, chicago, nextYear))
}

func TestTheUnappliedReceiptsCountTotalAndOldestWaitInWholeDays(t *testing.T) {
	t.Parallel()

	oldest := at(2026, time.November, 2, 22)
	read := unappliedRead(context.Background(), &unappliedRow{
		Count: 51, AmountMinor: 27_469_991, Oldest: &oldest,
	}, at(2026, time.November, 23, 1), chicago)

	assert.Equal(t, "51", read.Figures["receiptsToApply"].Number.String())
	assert.Equal(t, "274699.91", read.Figures["amountToApply"].Number.String())
	require.NotNil(t, read.Figures["oldestToApply"])
	assert.Equal(t, boardcatalog.ValueDays, read.Figures["oldestToApply"].Kind)
	assert.Equal(t, "21", read.Figures["oldestToApply"].Number.String())
	assert.Equal(t, "51 bank receipts in Exception", read.Source)

	none := unappliedRead(context.Background(), &unappliedRow{}, at(2026, time.November, 23, 1), chicago)
	assert.Equal(t, "0", none.Figures["receiptsToApply"].Number.String())
	assert.Contains(t, none.Figures, "oldestToApply")
	assert.Nil(t, none.Figures["oldestToApply"], "nothing waits, so there is no oldest")
}

func TestTheComparedDateIsTheSameMomentAPeriodEarlierInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	now := at(2026, time.November, 23, 9)
	assert.Equal(t, at(2026, time.October, 23, 9), earlierThan(now, boardcatalog.PeriodLastMonth, chicago))
	assert.Equal(t, at(2026, time.November, 16, 9), earlierThan(now, boardcatalog.PeriodLastWeek, chicago))
	assert.Equal(t, at(2026, time.August, 23, 9), earlierThan(now, boardcatalog.PeriodLastQuarter, chicago))
	assert.Equal(t, at(2026, time.October, 23, 9), earlierThan(now, "", chicago))
}

func buckets(current, d1, d31, d61, d90 int64) customerledgerports.ARAgingBucketTotals {
	return customerledgerports.ARAgingBucketTotals{
		CurrentMinor: current, Days1To30Minor: d1, Days31To60Minor: d31,
		Days61To90Minor: d61, DaysOver90Minor: d90,
		TotalOpenMinor: current + d1 + d31 + d61 + d90,
	}
}

func TestAgingMarksThePastDueBucketThatGrewMostAndKeepsBothDates(t *testing.T) {
	t.Parallel()

	then, now := at(2026, time.October, 23, 9), at(2026, time.November, 23, 9)
	read := agingRead(context.Background(), &agingFacts{
		then: &customerledgerports.ARAgingTrendPoint{
			PeriodEnd: then, Buckets: buckets(39_870_000, 15_120_000, 4_165_000, 1_890_000, 1_120_000),
		},
		now: &customerledgerports.ARAgingTrendPoint{
			PeriodEnd: now, OpenInvoiceCount: 412,
			Buckets: buckets(41_230_000, 16_845_000, 7_492_000, 2_130_000, 984_000),
		},
		thenAt: then, nowAt: now, location: chicago,
	})

	assert.Equal(t, "Aging, Nov 23 against Oct 23", read.Title)
	assert.Equal(t, "412 open invoices, as of each date", read.Source)
	require.NotNil(t, read.Chart)
	assert.Equal(t, []int{2}, read.Chart.Marked, "31 to 60 grew most; current growing is billing")
	assert.True(t, read.Chart.X[2].Marked)
	assert.Equal(t, "31–60", read.Chart.X[2].Label)
	require.Len(t, read.Chart.Series, 2)
	assert.Equal(t, boardcatalog.RoleWas, read.Chart.Series[0].Role)
	assert.Equal(t, "Oct 23", read.Chart.Series[0].Name)
	assert.Equal(t, boardcatalog.RoleNow, read.Chart.Series[1].Role)
	assert.Equal(t, "74920", read.Chart.Series[1].Points[2].String())
	assert.Equal(t, "686810", read.Figures["openNow"].Number.String())
	assert.Equal(t, "621650", read.Figures["openThen"].Number.String())
	assert.Equal(t, "31 to 60 days past due", read.Figures["grewBucket"].Text)
	assert.Equal(t, "74920", read.Figures["grewBucketNow"].Number.String())
	assert.Equal(t, "41650", read.Figures["grewBucketThen"].Number.String())
	assert.Equal(t, "412", read.Figures["openInvoicesNow"].Number.String())
	assert.Equal(t, then, read.Changes["openNow"].WasAt)
	assert.Equal(t, boardcatalog.ToneWarn, read.Changes["grewBucketNow"].Tone)
}

func TestAgingMarksNothingWhenNoPastDueBucketGrew(t *testing.T) {
	t.Parallel()

	read := agingRead(context.Background(), &agingFacts{
		then:   &customerledgerports.ARAgingTrendPoint{Buckets: buckets(100, 500, 400, 0, 0)},
		now:    &customerledgerports.ARAgingTrendPoint{Buckets: buckets(900, 500, 100, 0, 0)},
		thenAt: 1, nowAt: 2,
	})

	assert.Empty(t, read.Chart.Marked)
	assert.Equal(t, "No past-due bucket grew", read.Figures["grewBucket"].Text)
	assert.Nil(t, read.Figures["grewBucketNow"])
	assert.NotContains(t, read.Changes, "grewBucketNow")
}

type fakeAnalytics struct {
	customerledgerports.ARAnalyticsRepository

	compared []customerledgerports.CompareARAgingRequest
	totals   map[int64]*customerledgerports.ARCollectionTotals
	points   []*customerledgerports.ARAgingTrendPoint
}

func (f *fakeAnalytics) CompareAging(
	_ context.Context,
	req customerledgerports.CompareARAgingRequest,
) ([]*customerledgerports.ARAgingTrendPoint, error) {
	f.compared = append(f.compared, req)

	return f.points, nil
}

func (f *fakeAnalytics) GetCollectionTotals(
	_ context.Context,
	req customerledgerports.GetARCollectionMetricsRequest,
) (*customerledgerports.ARCollectionTotals, error) {
	return f.totals[req.AsOfDate], nil
}

func TestAgingCompareReadsTheReportAtNowAndAMonthBefore(t *testing.T) {
	t.Parallel()

	now := at(2026, time.November, 23, 9)
	then := at(2026, time.October, 23, 9)
	analytics := &fakeAnalytics{points: []*customerledgerports.ARAgingTrendPoint{
		{PeriodEnd: then, Buckets: buckets(1, 0, 0, 0, 0)},
		{PeriodEnd: now, Buckets: buckets(2, 0, 0, 0, 0), OpenInvoiceCount: 3},
	}}
	reader := NewAgingCompare(AnalyticsParams{Analytics: analytics})

	read, err := reader.Read(context.Background(), &boardreaderports.MetricRequest{
		Now: now, Location: chicago, Period: boardcatalog.PeriodLastMonth,
	})
	require.NoError(t, err)

	require.Len(t, analytics.compared, 1)
	assert.Equal(t, now, analytics.compared[0].AsOfDate)
	assert.Equal(t, then, analytics.compared[0].EarlierDate)
	assert.Equal(t, "0.02", read.Figures["openNow"].Number.String())
	assert.Equal(t, "0.01", read.Figures["openThen"].Number.String())
}

func TestCollectingReadsDaysToPayAndTheCollectedShareWithTheirEarlierValues(t *testing.T) {
	t.Parallel()

	now := at(2026, time.November, 23, 9)
	then := at(2026, time.October, 23, 9)
	analytics := &fakeAnalytics{totals: map[int64]*customerledgerports.ARCollectionTotals{
		now: {
			AvgDaysToPay: 38.44, ApplicationCount: 120, CollectedMinor: 50_000_00,
			BeginningOpenMinor: 600_00, CreditSalesMinor: 400_00,
			EndingOpenMinor: 190_00, EndingCurrentMinor: 0,
		},
		then: {
			AvgDaysToPay: 34.9, ApplicationCount: 100, CollectedMinor: 40_000_00,
			BeginningOpenMinor: 600_00, CreditSalesMinor: 400_00,
			EndingOpenMinor: 140_00, EndingCurrentMinor: 0,
		},
	}}
	reader := NewCollecting(AnalyticsParams{Analytics: analytics})

	read, err := reader.Read(context.Background(), &boardreaderports.MetricRequest{
		Now: now, Location: chicago, Period: boardcatalog.PeriodLastMonth,
	})
	require.NoError(t, err)

	assert.Equal(t, "38.4", read.Figures["daysToPay"].Number.String())
	assert.Equal(t, "0.81", read.Figures["collectedShare"].Number.String())
	assert.Equal(t, "50000", read.Figures["collectedAmount"].Number.String())
	assert.Equal(t, "34.9", read.Changes["daysToPay"].Was.Number.String())
	assert.Equal(t, then, read.Changes["daysToPay"].WasAt)
	assert.Equal(t, boardcatalog.ToneWarn, read.Changes["daysToPay"].Tone, "slower to pay")
	assert.Equal(t, "0.86", read.Changes["collectedShare"].Was.Number.String())
	assert.Equal(t, boardcatalog.ToneWarn, read.Changes["collectedShare"].Tone, "less came in")
	assert.Equal(t, "120 payments applied in the 91 days to Nov 23", read.Source)
}

func TestCollectingWithholdsAFigureItHasNothingToComputeFrom(t *testing.T) {
	t.Parallel()

	read := collectingRead(context.Background(), &collectingFacts{
		now:  &customerledgerports.ARCollectionTotals{},
		then: &customerledgerports.ARCollectionTotals{},
	})

	assert.Contains(t, read.Figures, "daysToPay")
	assert.Nil(t, read.Figures["daysToPay"], "no payment applied, so no time to pay")
	assert.Nil(t, read.Figures["collectedShare"], "nothing was due")
	assert.NotContains(t, read.Changes, "daysToPay")
}
