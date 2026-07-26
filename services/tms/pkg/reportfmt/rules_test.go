package reportfmt

import (
	"testing"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bandedMoney(rules ...Rule) Resolved {
	resolved := Resolve(ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
		Spec: &Spec{Rules: rules},
	})
	return resolved
}

func TestToneFirstMatchWins(t *testing.T) {
	display := bandedMoney(
		Rule{Op: RuleGreaterThan, Value: 100_000, Tone: ToneNegative},
		Rule{Op: RuleGreaterThan, Value: 10_000, Tone: ToneWarning},
	)

	assert.Equal(t, ToneNegative, display.ToneFor(decimal.NewFromInt(250_000)))
	assert.Equal(t, ToneWarning, display.ToneFor(decimal.NewFromInt(50_000)))
	assert.Equal(t, ToneNone, display.ToneFor(decimal.NewFromInt(500)))
}

func TestToneIgnoresNonNumericAndNull(t *testing.T) {
	display := bandedMoney(Rule{Op: RuleGreaterThan, Value: 0, Tone: ToneNegative})

	assert.Equal(t, ToneNone, display.ToneFor(nil))
	assert.Equal(t, ToneNone, display.ToneFor("ACME"))
	assert.Equal(t, ToneNone, display.ToneFor(true))
}

// A percent column stores a fraction and displays it times 100, so a threshold
// the author typed as "12" has to mean 12%.
func TestTonePercentComparesOnDisplayedScale(t *testing.T) {
	display := Resolve(ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatPercent,
		Spec: &Spec{Rules: []Rule{{Op: RuleGreaterThan, Value: 12, Tone: ToneNegative}}},
	})

	assert.Equal(t, ToneNegative, display.ToneFor(decimal.NewFromFloat(0.15)))
	assert.Equal(t, ToneNone, display.ToneFor(decimal.NewFromFloat(0.08)))
}

func TestToneBetweenIsInclusive(t *testing.T) {
	display := bandedMoney(Rule{Op: RuleBetween, Value: 10, Upper: 20, Tone: ToneWarning})

	assert.Equal(t, ToneWarning, display.ToneFor(int64(10)))
	assert.Equal(t, ToneWarning, display.ToneFor(int64(20)))
	assert.Equal(t, ToneNone, display.ToneFor(int64(21)))
}

func TestRuleValidation(t *testing.T) {
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

// A display carrying only rules must still be persisted; the emptiness check
// moved off a struct comparison when Spec gained a slice.
func TestSpecIsEmpty(t *testing.T) {
	assert.True(t, (&Spec{}).IsEmpty())
	assert.True(t, (*Spec)(nil).IsEmpty())
	assert.False(t, (&Spec{Rules: []Rule{{Op: RuleGreaterThan, Tone: ToneWarning}}}).IsEmpty())
	assert.False(t, (&Spec{Style: StyleCurrency}).IsEmpty())
}
