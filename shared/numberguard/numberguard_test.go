package numberguard_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func values(numbers ...float64) []decimal.Decimal {
	out := make([]decimal.Decimal, 0, len(numbers))
	for _, number := range numbers {
		out = append(out, decimal.NewFromFloat(number))
	}

	return out
}

func TestCheckNumbers_AcceptsProseThatCitesWhatWasComputed(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"On-time delivery for Acme Foods was 82.4%, down from 93.1%.",
		values(82.4, 93.1),
	)

	assert.True(t, check.OK)
	assert.Empty(t, check.Unsupported)
}

// This is the failure the guard exists for. The model was given $12,400 and
// wrote $48,000; that sentence gets repeated in a meeting.
func TestCheckNumbers_RejectsAFigureNobodyComputed(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"Detention at the Dallas yard is costing you $48,000 a month.",
		values(12400),
	)

	require.False(t, check.OK)
	assert.Contains(t, check.Unsupported, "48,000")
}

func TestCheckNumbers_AcceptsProseWithNoFiguresAtAll(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"Service to this customer has slipped materially since last month.",
		values(82.4),
	)

	assert.True(t, check.OK)
}

// A model writing for a person rounds. Rejecting "about 82%" for a computed
// 82.4 would leave every card wearing the detector's stiffer wording and buy
// nothing.
func TestCheckNumbers_AllowsRoundingOfAComputedFigure(t *testing.T) {
	t.Parallel()

	for _, prose := range []string{
		"On-time delivery is about 82%.",
		"On-time delivery is 82.4%.",
		"Detention cost roughly $12,400 over the window.",
	} {
		assert.True(t, numberguard.CheckNumbers(prose, values(82.4, 12437.19)).OK, prose)
	}
}

func TestCheckNumbers_ReadsThousandsAndMillionsShorthand(t *testing.T) {
	t.Parallel()

	assert.True(t, numberguard.CheckNumbers("That is $12.4k of exposure.", values(12400)).OK)
	assert.True(t, numberguard.CheckNumbers("Roughly 1.2M miles run empty.", values(1_200_000)).OK)
}

// "12.4" against a computed 12,400 is the same claim written in thousands, and
// a detector reporting dollars should not force the prose to spell out digits.
func TestCheckNumbers_ReadsABareFigureInThousands(t *testing.T) {
	t.Parallel()

	assert.True(t, numberguard.CheckNumbers("Exposure reached 12.4 thousand.", values(12400)).OK)
}

// Small numbers are sentence furniture. Demanding a detector compute a 3 to
// permit "the top 3 customers" would reject perfectly honest prose.
func TestCheckNumbers_TreatsSmallNumbersAsOrdinaryProse(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"The top 3 customers account for most of it, across 2 lanes and 11 stops.",
		values(82.4),
	)

	assert.True(t, check.OK)
}

// The ceiling must not become a hole big enough to hide a real claim in. A
// percentage is the obvious case: "on-time fell to 40%" is a serious statement.
func TestCheckNumbers_StillCatchesAFabricationAboveTheSmallNumberCeiling(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers("On-time delivery collapsed to 40%.", values(82.4))

	require.False(t, check.OK)
	assert.Contains(t, check.Unsupported, "40")
}

func TestCheckNumbers_ReportsEveryUnsupportedFigureNotJustTheFirst(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"Detention cost $48,000 across 310 stops.",
		values(12400),
	)

	require.False(t, check.OK)
	assert.Len(t, check.Unsupported, 2)
}

func TestCheckNumbers_HandlesTrailingPunctuation(t *testing.T) {
	t.Parallel()

	assert.True(t, numberguard.CheckNumbers("Exposure reached 12,400.", values(12400)).OK)
	assert.True(t, numberguard.CheckNumbers("Was 93.1, now 82.4.", values(93.1, 82.4)).OK)
}

func TestSupportedValues_CoversMetricsBaselinesAndExtras(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedValues(
		values(82.4),
		values(93.1),
		decimal.NewFromInt(30),
	)

	assert.True(t, numberguard.CheckNumbers("82.4% over 30 days, down from 93.1%.", supported).OK)
}

// "Fell 11 points" is the sentence worth reading, and it is arithmetic on two
// computed numbers rather than an invention.
func TestSupportedValues_AdmitsTheChangeBetweenAValueAndItsBaseline(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedValues(values(82.1), values(93.1))

	assert.True(t, numberguard.CheckNumbers("On-time fell 11 points this month.", supported).OK)
}

func TestFormatForPrompt_RendersNumbersTheWayTheGuardReadsThem(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "30", numberguard.FormatForPrompt(decimal.NewFromInt(30)))
	assert.Equal(t, "82.4", numberguard.FormatForPrompt(decimal.NewFromFloat(82.4)))
	assert.Equal(t, "12437.19", numberguard.FormatForPrompt(decimal.NewFromFloat(12437.19)))
	assert.Equal(t, "12437.19", numberguard.FormatForPrompt(decimal.NewFromFloat(12437.1856)))
}

// Whatever the prompt shows, the guard must accept back. If these two ever
// disagree, every narration is rejected and nothing says why.
func TestFormatForPrompt_RoundTripsThroughTheGuard(t *testing.T) {
	t.Parallel()

	for _, value := range values(82.4, 12437.19, 1_200_000, 0.5, 93) {
		prose := "The figure is " + numberguard.FormatForPrompt(value) + " exactly."

		assert.True(
			t,
			numberguard.CheckNumbers(prose, []decimal.Decimal{value}).OK,
			prose,
		)
	}
}

func TestSupportedFromText_CollectsEveryFigureOnce(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedFromText(
		`{"totalCharge": 12400.50, "stops": 3}`,
		"",
		"Invoice total is $12,400.50 across 1.2k miles",
	)

	require.Len(t, supported, 3)
	assert.True(t, supported[0].Equal(decimal.NewFromFloat(12400.50)))
	assert.True(t, supported[1].Equal(decimal.NewFromInt(3)))
	assert.True(t, supported[2].Equal(decimal.NewFromInt(1200)))
}

func TestSupportedFromText_BacksCheckNumbers(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedFromText(`{"revenue": 48210}`, "Show me revenue")

	assert.True(t, numberguard.CheckNumbers("Revenue was $48,210 this week.", supported).OK)
	check := numberguard.CheckNumbers("Revenue was $95,000 this week.", supported)
	assert.False(t, check.OK)
	assert.Equal(t, []string{"95,000"}, check.Unsupported)
}

func TestSupportedFromText_EmptyInputSupportsNothing(t *testing.T) {
	t.Parallel()

	assert.Empty(t, numberguard.SupportedFromText())
	assert.Empty(t, numberguard.SupportedFromText("", "no figures here"))
}

func TestOnlyCountsHoldsEveryNumberToTheCounts(t *testing.T) {
	t.Parallel()

	counts := []int{2, 4, 14}

	if !numberguard.OnlyCounts("14 agents are on and 4 proposals wait.", counts) {
		t.Fatal("counts that are all known were refused")
	}
	if numberguard.OnlyCounts("9 proposals wait.", counts) {
		t.Fatal("a small number that is not a count was accepted")
	}
	if numberguard.OnlyCounts("2.5 agents are on.", counts) {
		t.Fatal("a fraction was accepted as a count")
	}
	if !numberguard.OnlyCounts("Every agent is paused.", counts) {
		t.Fatal("prose with no numbers was refused")
	}
}

// DB-186: a model that read two reports and divided one total by the other in
// its head wrote a share nothing it read held. Money and shares are held to what
// was read; a count, a year, a clock time or a limit the trade knows are not.
func TestCheckAmounts_HoldsMoneyAndSharesOnly(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckAmounts(
		"Of the $87,310.40 billed in 31 days, 52.3% is still open. The 70-hour rule "+
			"applies from 2026 on, and the dock opens at 14:45.",
		values(87310.40),
	)

	require.False(t, check.OK)
	assert.Equal(t, []string{"52.3"}, check.Unsupported)
}

func TestCheckAmounts_ReadsEveryWayMoneyIsWritten(t *testing.T) {
	t.Parallel()

	for _, prose := range []string{
		"It came to $4,800.",
		"It came to -$4,800.",
		"It came to ($4,800).",
		"It came to US$ 4,800.",
		"It came to USD 4,800.",
		"It came to 4,800 CAD.",
		"It came to $4.8k.",
		"It came to €4,800.",
	} {
		check := numberguard.CheckAmounts(prose, values(1250))
		assert.False(t, check.OK, prose)
	}
}

func TestCheckAmounts_ReadsEveryWayAShareIsWritten(t *testing.T) {
	t.Parallel()

	for _, prose := range []string{
		"About 52.3% is open.",
		"About 52.3 % is open.",
		"About 52.3 percent is open.",
	} {
		assert.False(t, numberguard.CheckAmounts(prose, values(41)).OK, prose)
		assert.True(t, numberguard.CheckAmounts(prose, values(52.3)).OK, prose)
	}
}

func TestCheckAmounts_AShareIsTheFractionItRenders(t *testing.T) {
	t.Parallel()

	assert.True(t, numberguard.CheckAmounts("52.3% is still open.", values(0.5231)).OK)
	assert.False(t, numberguard.CheckAmounts("52.3% is still open.", values(0.61)).OK)
}

func TestCheckAmounts_LeavesSmallAmountsAndRoundingAlone(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckAmounts(
		"A $20 fee on 5% of loads; open balances average about $168K.",
		values(167935.06),
	)

	assert.True(t, check.OK, check.Unsupported)
}

// A tool's JSON lists numbers with nothing but a comma between them. Read as
// thousands separators, "[172.5,12937.5,0.708]" was one unparseable run, and
// every figure an analysis table returned counted as never read.
func TestSupportedFromText_ReadsEachNumberOfAJSONArray(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedFromText(
		`{"rows":[["Riverside Cold Storage",39,39,172.5,12937.5,0.708]]}`)

	check := numberguard.CheckNumbers("Riverside cost $12,937.50, 70.8% of the total.", supported)
	assert.True(t, check.OK, check.Unsupported)
	assert.False(t, numberguard.CheckNumbersAs(numberguard.ReadJoined, "Riverside cost $12,937.50.",
		numberguard.SupportedFromTextAs(numberguard.ReadJoined, `[172.5,12937.5,0.708]`)).OK,
		"the joined reading is the one that failed")
}

func TestCheckNumbers_DatesTimesRangesAndCodesAreNotFigures(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"Delivery moves to 2026-10-25 11:00–13:30, or 10/26 if you meant next week. "+
			"Pickup # HL-30117 on INV2610000001, call 214-555-0142, in the 31-60 bucket, "+
			"by the 31st.",
		nil,
	)

	assert.True(t, check.OK, check.Unsupported)
}

func TestCheckNumbers_ListsEachUnsupportedFigureOnce(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers("Short by $500. Write off the $500 or dispute $500.", nil)

	assert.Equal(t, []string{"500"}, check.Unsupported)
}

func TestCheckNumbers_ReadsAJSONRowApart(t *testing.T) {
	t.Parallel()

	supported := numberguard.SupportedFromText(`{"rows":[[90,270,270],[50,150,250]]}`)

	assert.True(t, numberguard.CheckNumbers("50 miles prices at $150 before the minimum.",
		supported).OK)
	assert.True(t, numberguard.CheckNumbers("It came to $150,250.",
		numberguard.SupportedFromText("Total: 150,250 USD")).OK,
		"prose keeps its thousands separators")
}

func TestCheckNumbers_DaysAndYearsOfAWrittenDateAreNotFigures(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers(
		"Deliveries on September 29 and 30; time off Dec 26–31, 2026 and Jan 27–31, 2027; "+
			"due 29 Sep. The $3,200 Oct invoice is open.",
		nil,
	)

	assert.Equal(t, []string{"3,200"}, check.Unsupported)
}

func TestCheckNumbers_AFigureRoundedToItsLastDigitIsTheSameFigure(t *testing.T) {
	t.Parallel()

	supported := values(57323.83, 65438.46, 0.4343)

	check := numberguard.CheckNumbers(
		"One TEAM load is $57K; without it the total is about $65k, 43% accessorial.",
		supported,
	)
	assert.True(t, check.OK, check.Unsupported)

	check = numberguard.CheckNumbers("FreshHaul's other loads add up to roughly $62K.", supported)
	assert.Equal(t, []string{"62"}, check.Unsupported, "a different figure is still different")
}

func TestWorked_ChecksArithmeticOnFiguresTheProseStates(t *testing.T) {
	t.Parallel()

	read := values(6050147.60, 6000000, 112, 3.89)

	for _, prose := range []string{
		"$6,050,147.60 is open; without the $6,000,000 invoice, $50,147.60 is.",
		"112 gallons at $3.89 come to about $435.68.",
		"FreshHaul's $6,000,000 is 99.2% of the $6,050,147.60 open.",
	} {
		check := numberguard.CheckAmounts(prose, append(read, numberguard.Worked(prose, read)...))
		assert.True(t, check.OK, "%s: %v", prose, check.Unsupported)
	}

	wrong := "$6,050,147.60 is open; without the $6,000,000 invoice, $52,147.60 is."
	check := numberguard.CheckAmounts(wrong, append(read, numberguard.Worked(wrong, read)...))
	assert.Equal(t, []string{"52,147.60"}, check.Unsupported, "a wrong result is still caught")

	alone := "About 99.2% is FreshHaul's."
	check = numberguard.CheckAmounts(alone, append(read, numberguard.Worked(alone, read)...))
	assert.False(t, check.OK, "a share whose sides the reply does not give is not worked")
}

func TestWorked_PairsOnlyFiguresThatWereRead(t *testing.T) {
	t.Parallel()

	prose := "A $40,000 fee on the $6,000,000 invoice leaves $6,040,000."
	read := values(6000000)

	check := numberguard.CheckAmounts(prose, append(read, numberguard.Worked(prose, read)...))
	assert.Equal(t, []string{"40,000", "6,040,000"}, check.Unsupported,
		"an invented side makes an invented result")
}

func TestCheckNumbers_CodesAndYearsAreNotFigures(t *testing.T) {
	t.Parallel()

	prose := "Acme has four other $1,121.36 invoices (INV2610000103, 104, 099 and an " +
		"earlier one, 083), dated in 2027, out of 1,940 loads."

	check := numberguard.CheckNumbers(prose, values(1121.36))
	assert.Equal(t, []string{"1,940"}, check.Unsupported)

	separated := numberguard.CheckNumbersAs(numberguard.ReadSeparated, prose, values(1121.36))
	assert.Equal(t, []string{"104", "099", "083", "2027", "1,940"}, separated.Unsupported,
		"the reading executions decided with is unchanged")
}

func TestCheckNumbers_AListAfterACodeEndsAtMoney(t *testing.T) {
	t.Parallel()

	check := numberguard.CheckNumbers("INV2610000103, $104 and 099 are open.", nil)

	assert.Equal(t, []string{"104"}, check.Unsupported)
}
