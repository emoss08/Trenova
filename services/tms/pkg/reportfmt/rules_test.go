package reportfmt

import (
	"testing"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tones(cells []Cell) []Tone {
	out := make([]Tone, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.Tone)
	}
	return out
}

func bandedMoney(t *testing.T, rules ...Rule) Resolved {
	t.Helper()
	return resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
		Spec: &Spec{Rules: rules},
	})
}

func TestToneFirstMatchWins(t *testing.T) {
	t.Parallel()

	display := bandedMoney(t,
		Rule{Op: RuleGreaterThan, Value: 100_000, Tone: ToneNegative},
		Rule{Op: RuleGreaterThan, Value: 10_000, Tone: ToneWarning},
	)

	assert.Equal(t, []Tone{ToneNegative, ToneWarning, ToneNone},
		tones(formatValues(t, display, reportcatalog.FieldDecimal,
			decimal.NewFromInt(250_000), decimal.NewFromInt(50_000), decimal.NewFromInt(500))))
}

func TestToneIgnoresNonNumericAndNull(t *testing.T) {
	t.Parallel()

	display := bandedMoney(t, Rule{Op: RuleGreaterThan, Value: 0, Tone: ToneNegative})

	assert.Equal(t, []Tone{ToneNone, ToneNone, ToneNone},
		tones(formatValues(t, display, reportcatalog.FieldDecimal, nil, "ACME", true)))
}

func TestTonePercentComparesOnDisplayedScale(t *testing.T) {
	t.Parallel()

	display := resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatPercent,
		Spec: &Spec{Rules: []Rule{{Op: RuleGreaterThan, Value: 12, Tone: ToneNegative}}},
	})

	assert.Equal(t, []Tone{ToneNegative, ToneNone},
		tones(formatValues(t, display, reportcatalog.FieldDecimal,
			decimal.RequireFromString("0.15"), decimal.RequireFromString("0.08"))))
}

func TestToneBetweenIsInclusive(t *testing.T) {
	t.Parallel()

	display := bandedMoney(t, Rule{Op: RuleBetween, Value: 10, Upper: 20, Tone: ToneWarning})

	assert.Equal(t, []Tone{ToneWarning, ToneWarning, ToneNone},
		tones(formatValues(t, display, reportcatalog.FieldInt, int64(10), int64(20), int64(21))))
}

func TestToneComparesExactly(t *testing.T) {
	t.Parallel()

	display := bandedMoney(t, Rule{Op: RuleEqual, Value: 0.3, Tone: ToneNeutral})

	assert.Equal(t, []Tone{ToneNeutral},
		tones(formatValues(t, display, reportcatalog.FieldDecimal, decimal.RequireFromString("0.3"))))
}

func TestResolveKeepsTheRules(t *testing.T) {
	t.Parallel()

	display := bandedMoney(t,
		Rule{Op: RuleBetween, Value: 10.5, Upper: 20, Tone: ToneWarning},
		Rule{Op: RuleLessThan, Value: -1, Tone: ToneNegative},
	)

	assert.Equal(t, []Rule{
		{Op: RuleBetween, Value: 10.5, Upper: 20, Tone: ToneWarning},
		{Op: RuleLessThan, Value: -1, Tone: ToneNegative},
	}, display.Rules)
}

func TestRuleValidation(t *testing.T) {
	t.Parallel()

	spec := &Spec{Rules: []Rule{{Op: "nonsense", Value: 1, Tone: ToneNegative}}}
	require.ErrorContains(t, spec.Validate(reportcatalog.FieldDecimal), "unknown comparison")

	spec = &Spec{Rules: []Rule{{Op: RuleGreaterThan, Value: 1, Tone: "chartreuse"}}}
	require.ErrorContains(t, spec.Validate(reportcatalog.FieldDecimal), "unknown emphasis")

	spec = &Spec{Rules: []Rule{{Op: RuleBetween, Value: 20, Upper: 10, Tone: ToneWarning}}}
	require.ErrorContains(t, spec.Validate(reportcatalog.FieldDecimal), "upper bound")

	tooMany := make([]Rule, MaxRules+1)
	for i := range tooMany {
		tooMany[i] = Rule{Op: RuleGreaterThan, Value: float64(i), Tone: ToneWarning}
	}
	require.ErrorContains(
		t,
		(&Spec{Rules: tooMany}).Validate(reportcatalog.FieldDecimal),
		"at most",
	)
}

func TestSpecIsEmpty(t *testing.T) {
	t.Parallel()

	assert.True(t, (&Spec{}).IsEmpty())
	assert.True(t, (*Spec)(nil).IsEmpty())
	assert.False(t, (&Spec{Rules: []Rule{{Op: RuleGreaterThan, Tone: ToneWarning}}}).IsEmpty())
	assert.False(t, (&Spec{Style: StyleCurrency}).IsEmpty())
}
