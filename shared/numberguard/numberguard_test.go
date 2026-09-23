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
