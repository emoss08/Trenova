package billingboardreader

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exceptionReceipt(amount int64) *receiptRow {
	return &receiptRow{
		Status:          "Exception",
		Reference:       "SEED-INV-0003",
		AmountMinor:     amount,
		ReceiptDate:     at(2026, time.November, 20, 9),
		ExceptionReason: "No unique customer payment match found for bank receipt",
	}
}

func filled(words *boardreading.Words) string {
	if words == nil {
		return ""
	}
	value := ""
	if words.Value != nil {
		switch words.Value.Kind {
		case boardcatalog.ValueText:
			value = words.Value.Text
		default:
			value = "$" + words.Value.Number.StringFixed(2)
		}
	}

	return strings.ReplaceAll(words.Text, "{0}", value)
}

func TestAgainstCandidate_ToTheCentOnlyWhenTheReferenceNamesTheInvoice(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(253_125), &bestRow{
		Kind: "Invoice", Standing: "Exact", OpenMinor: 253_125, InvoiceNumber: "SEED-INV-0006",
	})

	assert.Equal(t, "Matches SEED-INV-0006 to the cent", filled(reading.Fact))
	assert.Equal(t, "$2531.25", filled(&reading.Standing), "the row stands the receipt's own amount")
	assert.Equal(t, boardcatalog.ToneOK, reading.Tone)
	assert.Nil(t, reading.Bounded, "an exact match has no bar")
}

func TestAgainstCandidate_ShortLeavesAGapBeforeTheOpenAmount(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(415_500), &bestRow{
		Kind: "Invoice", Standing: "Short", OpenMinor: 427_500, DifferenceMinor: -12_000,
		InvoiceNumber: "SEED-INV-0003",
	})

	assert.Equal(t, "$120.00 short of SEED-INV-0003", filled(reading.Fact))
	assert.NotContains(t, filled(reading.Fact), "to the cent")
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	require.NotNil(t, reading.Bounded)
	assert.Equal(t, "415500", reading.Bounded.Value.String())
	assert.Equal(t, "427500", reading.Bounded.Bound.String(),
		"the bound is the open amount, so the bar stops short of the mark")
}

func TestAgainstCandidate_OverRunsPastTheOpenAmount(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(387_010), &bestRow{
		Kind: "Invoice", Standing: "Over", OpenMinor: 382_500, DifferenceMinor: 4_510,
		InvoiceNumber: "SEED-INV-0008",
	})

	assert.Equal(t, "$45.10 over SEED-INV-0008", filled(reading.Fact))
	require.NotNil(t, reading.Bounded)
	assert.True(t, reading.Bounded.Value.GreaterThan(reading.Bounded.Bound))
}

func TestAgainstCandidate_OneCentEitherSideIsNotToTheCent(t *testing.T) {
	t.Parallel()

	short := againstReading(context.Background(), exceptionReceipt(253_124), &bestRow{
		Kind: "Invoice", Standing: "Short", OpenMinor: 253_125, DifferenceMinor: -1,
		InvoiceNumber: "SEED-INV-0006",
	})
	over := againstReading(context.Background(), exceptionReceipt(253_126), &bestRow{
		Kind: "Invoice", Standing: "Over", OpenMinor: 253_125, DifferenceMinor: 1,
		InvoiceNumber: "SEED-INV-0006",
	})

	assert.Equal(t, "$0.01 short of SEED-INV-0006", filled(short.Fact))
	assert.Equal(t, "$0.01 over SEED-INV-0006", filled(over.Fact))
}

func TestAgainstCandidate_TheSameAmountNamedElsewhereIsNotAMatch(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(101_250), &bestRow{
		Kind: "Invoice", Standing: "SameAmount", OpenMinor: 101_250, InvoiceNumber: "SEED-INV-0010",
	})

	assert.Equal(t, "Same amount as SEED-INV-0010", filled(reading.Fact))
	assert.Nil(t, reading.Bounded)
}

func TestAgainstCandidate_APaymentIsNamedByItsCustomer(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(10_000), &bestRow{
		Kind: "CustomerPayment", Standing: "Short", OpenMinor: 10_050, DifferenceMinor: -50,
		Customer: "Premier Farms",
	})

	assert.Equal(t, "$0.50 short of Premier Farms's payment", filled(reading.Fact))
}

func TestAgainstCandidate_NoCandidateReadsAsTheReceiptsOwnException(t *testing.T) {
	t.Parallel()

	reading := againstReading(context.Background(), exceptionReceipt(838_303), nil)

	assert.Equal(t, "No unique customer payment match found for bank receipt", filled(reading.Fact))
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	assert.Nil(t, reading.Bounded)
}

func TestAgainstCandidate_AMatchedReceiptIgnoresAnyLeftoverCandidate(t *testing.T) {
	t.Parallel()

	matchedAt := at(2026, time.November, 21, 10)
	row := exceptionReceipt(10_000)
	row.Status = "Matched"
	row.MatchedAt = &matchedAt
	reading := againstReading(context.Background(), row, &bestRow{
		Kind: "Invoice", Standing: "Short", OpenMinor: 20_000, DifferenceMinor: -10_000,
		InvoiceNumber: "SEED-INV-0001",
	})

	require.NotNil(t, reading.Fact)
	assert.Equal(t, "Matched {0}", reading.Fact.Text)
	assert.Nil(t, reading.Bounded)
}

func TestUnappliedReceipts_CountsEachReceiptByItsBestShownCandidate(t *testing.T) {
	t.Parallel()

	read := unappliedRead(context.Background(), &unappliedRow{Count: 10, AmountMinor: 1}, 0, nil)
	addStandings(read, 10, []standingCount{
		{Standing: "Exact", Count: 2},
		{Standing: "Short", Count: 3},
		{Standing: "Over", Count: 1},
	})

	count := func(name string) int64 {
		value := read.Figures[name]
		require.NotNil(t, value, name)

		return value.Number.IntPart()
	}
	assert.Equal(t, int64(2), count(billingboard.FigureReceiptsExact))
	assert.Equal(t, int64(0), count(billingboard.FigureReceiptsSameAmount))
	assert.Equal(t, int64(3), count(billingboard.FigureReceiptsShort))
	assert.Equal(t, int64(1), count(billingboard.FigureReceiptsOver))
	assert.Equal(t, int64(4), count(billingboard.FigureReceiptsWithoutCandidate),
		"a receipt whose only candidates the reader may not see counts as having none")
}
