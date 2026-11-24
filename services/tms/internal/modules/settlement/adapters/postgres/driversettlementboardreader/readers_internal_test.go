package driversettlementboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pacific = func() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}

	return loc
}()

func payDay() int64 {
	return time.Date(2026, time.November, 27, 0, 0, 0, 0, time.UTC).Unix()
}

func TestASettlementWithACriticalExceptionSaysSoAndWhatCarriesForward(t *testing.T) {
	t.Parallel()

	reading := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status:               driversettlement.StatusDraft,
		NetPayMinor:          0,
		CarryForwardOutMinor: -333_879,
		Currency:             "USD",
		PayDate:              payDay(),
		Block:                driversettlement.ReleaseBlockCriticalException,
		Exceptions: []driversettlement.Exception{
			{
				Code:     driversettlement.ExceptionCodeHighVariance,
				Severity: driversettlement.ExceptionSeverityWarning,
				Message:  "Net differs from last week",
			},
			{
				Code:     driversettlement.ExceptionCodeNegativeNet,
				Severity: driversettlement.ExceptionSeverityCritical,
				Message:  "Deductions exceed earnings; 3338.79 USD carries forward",
			},
		},
	}, pacific)

	assert.Equal(t, "0", reading.Value.Number.String(), "what a board adds up is the net")
	assert.Equal(t, "{0} carries forward", reading.Standing.Text)
	require.NotNil(t, reading.Standing.Value)
	assert.Equal(t, "-3338.79", reading.Standing.Value.Number.String())
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	require.NotNil(t, reading.Fact)
	assert.Equal(t, "{0}", reading.Fact.Text)
	require.NotNil(t, reading.Fact.Value)
	assert.Equal(t,
		"Has a critical exception: Deductions exceed earnings; 3338.79 USD carries forward",
		reading.Fact.Value.Text)
}

func TestASettlementWithNoPayProfileIsNotPriced(t *testing.T) {
	t.Parallel()

	reading := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status:   driversettlement.StatusDraft,
		Currency: "USD",
		PayDate:  payDay(),
		Block:    driversettlement.ReleaseBlockCriticalException,
		Exceptions: []driversettlement.Exception{{
			Code:     driversettlement.ExceptionCodeMissingPayProfile,
			Severity: driversettlement.ExceptionSeverityCritical,
			Message:  "No active pay profile",
		}},
	}, pacific)

	assert.Equal(t, "Not priced", reading.Standing.Text)
	assert.Nil(t, reading.Standing.Value, "an unpriced settlement does not stand at $0.00")
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
}

func TestASettlementWithADisputeOpenSaysWhatItIsAbout(t *testing.T) {
	t.Parallel()

	reading := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status:          driversettlement.StatusApproved,
		NetPayMinor:     141_250,
		Currency:        "USD",
		PayDate:         payDay(),
		Block:           driversettlement.ReleaseBlockDisputeOpen,
		DisputeCategory: driversettlement.DisputeCategoryIncorrectRate,
	}, pacific)

	assert.Equal(t, "1412.5", reading.Value.Number.String())
	assert.Equal(t, "{0}", reading.Standing.Text)
	require.NotNil(t, reading.Fact)
	assert.Equal(t, "Dispute open", reading.Fact.Text)
	assert.Equal(t, "about the rate", reading.Fact.Place)
	assert.NotContains(t, reading.Fact.Text, "held")
}

func TestAReadySettlementSaysTheDayItPays(t *testing.T) {
	t.Parallel()

	reading := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status:      driversettlement.StatusPendingApproval,
		NetPayMinor: 98_000,
		Currency:    "USD",
		PayDate:     payDay(),
	}, pacific)

	require.NotNil(t, reading.Fact)
	assert.Equal(t, "Pays {0}", reading.Fact.Text)
	assert.True(t, reading.Fact.Day)
	assert.Equal(t, 27, time.Unix(reading.Fact.At, 0).In(pacific).Day(),
		"a pay date kept at midnight UTC is still the 27th in the organization's zone")
	assert.Empty(t, reading.Tone)

	paid := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status: driversettlement.StatusPaid, NetPayMinor: 98_000, Currency: "USD",
		PayDate: payDay(),
	}, pacific)
	assert.Nil(t, paid.Fact, "a paid settlement does not say it pays")
}

func TestASettlementsStatusIsTheDeskCardsWordAndTone(t *testing.T) {
	t.Parallel()

	cases := map[driversettlement.Status]struct {
		words string
		tone  boardcatalog.Tone
	}{
		driversettlement.StatusDraft:           {"Draft", ""},
		driversettlement.StatusPendingApproval: {"Waiting on approval", boardcatalog.ToneWarn},
		driversettlement.StatusApproved:        {"Approved", boardcatalog.ToneInfo},
		driversettlement.StatusPosted:          {"Posted, not paid", boardcatalog.ToneInfo},
		driversettlement.StatusPaid:            {"Paid", boardcatalog.ToneOK},
		driversettlement.StatusVoided:          {"Voided", ""},
	}
	for status, want := range cases {
		words, tone := settlementStatus(context.Background(), status)
		assert.Equal(t, want.words, words, status)
		assert.Equal(t, want.tone, tone, status)
	}
}

func openBatch() *settlementreleaseports.Batch {
	return &settlementreleaseports.Batch{
		Name:                "Pay period ending Nov 21, 2026",
		Batches:             1,
		PayDate:             payDay(),
		Currency:            "USD",
		Currencies:          1,
		Settlements:         183,
		GrossMinor:          18_243_846,
		ReimbursementsMinor: 524_800,
		NetMinor:            16_246_106,
		Ready:               settlementreleaseports.Tally{Count: 174, NetMinor: 16_104_856},
		Blocked: map[driversettlement.ReleaseBlock]settlementreleaseports.Tally{
			driversettlement.ReleaseBlockCriticalException: {Count: 8},
			driversettlement.ReleaseBlockDisputeOpen:       {Count: 1, NetMinor: 141_250},
		},
	}
}

func TestTheBatchSplitsEarnedPayIntoReadyWhatCannotBeReleasedAndDeducted(t *testing.T) {
	t.Parallel()

	read := batchRead(context.Background(), openBatch(), pacific, 0)

	assert.Equal(t, "Pay period ending Nov 21, 2026", read.Title)
	assert.Equal(t, "183 settlements, paid Nov 27, 2026", read.Source)
	require.NotNil(t, read.Split)
	assert.Equal(t, "187686.46", read.Split.Total.Number.String(),
		"the whole is what was earned and reimbursed")
	labels := make([]string, 0, len(read.Split.Parts))
	var sum int64
	for _, part := range read.Split.Parts {
		labels = append(labels, part.Label)
		sum += part.Value.Number.Shift(2).IntPart()
		assert.NotContains(t, part.Label, "eld", "nothing is called held when nothing is")
	}
	assert.Equal(t, []string{
		"Ready to release", "Has a critical exception", "Dispute open", "Deducted",
	}, labels)
	assert.Equal(t, int64(18_768_646), sum, "the parts make up the whole")

	figures := read.Figures
	assert.Equal(t, "162461.06", figures[settlementboard.FigureBatchNet].Number.String())
	assert.Equal(t, "25225.4", figures[settlementboard.FigureBatchDeducted].Number.String())
	assert.Equal(t, "9", figures[settlementboard.FigureBlockedSettlements].Number.String())
	assert.Equal(t, "8", figures[settlementboard.FigureCriticalSettlements].Number.String())
	assert.Equal(t, "1", figures[settlementboard.FigureDisputeSettlements].Number.String())
	assert.Equal(t, "1412.5", figures[settlementboard.FigureBlockedNet].Number.String())
	assert.Equal(t, "Nov 27, 2026", figures[settlementboard.FigureBatchPayDate].Text)
	for _, metric := range settlementboard.Metrics {
		if metric.Name != settlementboard.MetricSettlementBatch {
			continue
		}
		for _, figure := range metric.Figures {
			assert.Contains(t, figures, figure.Name, "every declared figure is read")
		}
	}
}

func TestABatchInMoreThanOneCurrencyWithholdsItsMoneyAndDrawsNoSplit(t *testing.T) {
	t.Parallel()

	batch := openBatch()
	batch.Currencies = 2
	read := batchRead(context.Background(), batch, pacific, 0)

	assert.Nil(t, read.Split)
	assert.Nil(t, read.Figures[settlementboard.FigureBatchNet], "withheld, not added across currencies")
	assert.Equal(t, "183", read.Figures[settlementboard.FigureBatchSettlements].Number.String())
}

func TestNoBatchDrawsNothingAndReadsNoFigure(t *testing.T) {
	t.Parallel()

	read := batchRead(context.Background(), nil, pacific, 0)

	assert.Nil(t, read.Split)
	for name, value := range read.Figures {
		assert.Nil(t, value, name)
	}
}

func TestTheLastBatchesStackNetBelowDeductedWithTheNewestMarked(t *testing.T) {
	t.Parallel()

	week := func(start time.Time, gross, reimb, net int64) *settlementreleaseports.Week {
		return &settlementreleaseports.Week{
			PeriodStart: start.Unix(), Batches: 1, Currency: "USD", Currencies: 1,
			GrossMinor: gross, ReimbursementsMinor: reimb, NetMinor: net,
		}
	}
	day := func(d int) time.Time { return time.Date(2026, time.November, d, 0, 0, 0, 0, time.UTC) }
	read := weeksRead(context.Background(), []*settlementreleaseports.Week{
		week(day(1), 17_919_372, 563_000, 15_734_659),
		week(day(8), 20_579_767, 448_200, 17_861_289),
		week(day(15), 18_243_846, 524_800, 16_246_106),
	}, pacific, 0)

	require.NotNil(t, read.Chart)
	assert.Equal(t, boardcatalog.RoleStack, read.Chart.Series[0].Role)
	assert.Equal(t, "Net pay", read.Chart.Series[0].Name)
	assert.Equal(t, "Deducted", read.Chart.Series[1].Name)
	assert.Equal(t, []int{2}, read.Chart.Marked)
	assert.Equal(t, 15, time.Unix(read.Chart.X[2].At, 0).In(pacific).Day())
	assert.Equal(t, "162461.06", read.Chart.Series[0].Points[2].String())
	assert.Equal(t, "25225.4", read.Chart.Series[1].Points[2].String())
	assert.Equal(t, "210279.67",
		read.Figures[settlementboard.FigureLargestBatchEarned].Number.String())
	assert.Equal(t, "Week of Nov 8, 2026",
		read.Figures[settlementboard.FigureLargestBatchPeriod].Text)
	assert.Equal(t, "187686.46",
		read.Figures[settlementboard.FigureLatestBatchEarned].Number.String())
	assert.Equal(t, "3 pay batches", read.Source)
}

func TestADayInThisYearIsWordedWithoutTheYear(t *testing.T) {
	t.Parallel()

	payDate := time.Date(2026, time.November, 27, 0, 0, 0, 0, time.UTC).Unix()
	inYear := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC).Unix()
	nextYear := time.Date(2027, time.January, 5, 12, 0, 0, 0, time.UTC).Unix()

	assert.Equal(t, "Nov 27", dayText(context.Background(), payDate, pacific, inYear))
	assert.Equal(t, "Nov 27, 2026", dayText(context.Background(), payDate, pacific, nextYear))
}
