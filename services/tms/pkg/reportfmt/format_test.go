package reportfmt_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decimalOf(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(value)
	require.NoError(t, err)
	return d
}

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func TestFormatMoneyTamesRawQuotients(t *testing.T) {
	display := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
	})

	assert.Equal(
		t,
		"$1,082.07",
		display.String(decimalOf(t, "1082.0672643987443142"), time.UTC),
	)
	assert.Equal(
		t,
		"($1,082.07)",
		display.String(decimalOf(t, "-1082.0672643987443142"), time.UTC),
	)
	assert.Equal(t, "", display.String(nil, time.UTC))
}

func TestFormatNumberOptions(t *testing.T) {
	tests := []struct {
		name  string
		spec  *reportfmt.Spec
		value string
		want  string
	}{
		{
			name:  "grouping off",
			spec:  &reportfmt.Spec{Style: reportfmt.StyleNumber, Grouping: boolPtr(false)},
			value: "1234567.891",
			want:  "1234567.89",
		},
		{
			name:  "zero decimals rounds half away from zero",
			spec:  &reportfmt.Spec{Style: reportfmt.StyleNumber, Decimals: intPtr(0)},
			value: "1234.5",
			want:  "1,235",
		},
		{
			name: "compact notation",
			spec: &reportfmt.Spec{
				Style:    reportfmt.StyleNumber,
				Decimals: intPtr(1),
				Notation: reportfmt.NotationCompact,
			},
			value: "1284000",
			want:  "1.3M",
		},
		{
			name: "suffix and prefix",
			spec: &reportfmt.Spec{
				Style:    reportfmt.StyleNumber,
				Decimals: intPtr(1),
				Suffix:   " mi",
			},
			value: "1284.44",
			want:  "1,284.4 mi",
		},
		{
			name:  "percent multiplies by one hundred",
			spec:  &reportfmt.Spec{Style: reportfmt.StylePercent, Decimals: intPtr(1)},
			value: "0.1834",
			want:  "18.3%",
		},
		{
			name: "minus sign instead of parentheses",
			spec: &reportfmt.Spec{
				Style:    reportfmt.StyleCurrency,
				Negative: reportfmt.NegativeMinus,
			},
			value: "-12.5",
			want:  "-$12.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			display := reportfmt.Resolve(reportfmt.ResolveInput{
				Type: reportcatalog.FieldDecimal,
				Spec: tt.spec,
			})
			assert.Equal(t, tt.want, display.String(decimalOf(t, tt.value), time.UTC))
		})
	}
}

func TestFormatDurations(t *testing.T) {
	tests := []struct {
		style reportfmt.DurationStyle
		want  string
	}{
		{reportfmt.DurationStyleClock, "26:10"},
		{reportfmt.DurationStyleCompact, "1d 2h 10m"},
		{reportfmt.DurationStyleVerbose, "1 day 2 hours 10 minutes"},
		{reportfmt.DurationStyleDecimal, "26.2 h"},
	}

	for _, tt := range tests {
		t.Run(string(tt.style), func(t *testing.T) {
			display := reportfmt.Resolve(reportfmt.ResolveInput{
				Type: reportcatalog.FieldDecimal,
				Spec: &reportfmt.Spec{
					Style:         reportfmt.StyleDuration,
					DurationUnit:  reportfmt.DurationUnitSeconds,
					DurationStyle: tt.style,
					Decimals:      intPtr(1),
				},
			})
			assert.Equal(t, tt.want, display.String(decimalOf(t, "94200"), time.UTC))
		})
	}
}

func TestFormatNegativeDurationKeepsSign(t *testing.T) {
	display := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Spec: &reportfmt.Spec{
			Style:         reportfmt.StyleDuration,
			DurationStyle: reportfmt.DurationStyleCompact,
		},
	})

	assert.Equal(t, "-2h 30m", display.String(decimalOf(t, "-9000"), time.UTC))
}

func TestFormatDatesUseRequestLocation(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	instant := time.Date(2026, time.March, 4, 1, 30, 0, 0, time.UTC).Unix()

	tests := []struct {
		style reportfmt.DateStyle
		want  string
	}{
		{reportfmt.DateStyleDate, "2026-03-03"},
		{reportfmt.DateStyleDateTime, "2026-03-03 19:30"},
		{reportfmt.DateStyleDateLong, "Mar 3, 2026"},
		{reportfmt.DateStyleMonthYear, "Mar 2026"},
		{reportfmt.DateStyleQuarter, "Q1 2026"},
		{reportfmt.DateStyleYear, "2026"},
		{reportfmt.DateStyleTime, "19:30"},
	}

	for _, tt := range tests {
		t.Run(string(tt.style), func(t *testing.T) {
			display := reportfmt.Resolve(reportfmt.ResolveInput{
				Type: reportcatalog.FieldEpoch,
				Spec: &reportfmt.Spec{DateStyle: tt.style},
			})
			assert.Equal(t, tt.want, display.String(instant, chicago))
		})
	}
}

// An averaged epoch column comes back as a decimal but still denotes an
// instant, so the date styles have to accept it.
func TestFormatDateAcceptsDecimalEpoch(t *testing.T) {
	display := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Spec: &reportfmt.Spec{Style: reportfmt.StyleDate, DateStyle: reportfmt.DateStyleDate},
	})

	assert.Equal(t, "2026-03-04", display.String(decimalOf(t, "1772587800.5"), time.UTC))
}

func TestFormatBooleansAndText(t *testing.T) {
	yesNo := reportfmt.Resolve(reportfmt.ResolveInput{Type: reportcatalog.FieldBool})
	assert.Equal(t, "Yes", yesNo.String(true, time.UTC))
	assert.Equal(t, "No", yesNo.String(false, time.UTC))

	check := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldBool,
		Spec: &reportfmt.Spec{BoolStyle: reportfmt.BoolStyleCheck},
	})
	assert.Equal(t, "✓", check.String(true, time.UTC))

	dashed := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldString,
		Spec: &reportfmt.Spec{NullText: "—"},
	})
	assert.Equal(t, "—", dashed.String(nil, time.UTC))
	assert.Equal(t, "—", dashed.String("", time.UTC))
	assert.Equal(t, "ACME", dashed.String("ACME", time.UTC))
}

func TestResolveAppliesCatalogHints(t *testing.T) {
	distance := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatDistance,
	})
	assert.Equal(t, "1,284.4 mi", distance.String(decimalOf(t, "1284.44"), time.UTC))

	weight := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatWeight,
	})
	assert.Equal(t, "42,000 lbs", weight.String(int64(42000), time.UTC))

	counted := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatCount,
	})
	assert.Equal(t, "1,204", counted.String(int64(1204), time.UTC))
}

func TestExcelFormats(t *testing.T) {
	money := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
	})
	code, native := money.ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, `"$"#,##0.00;("$"#,##0.00)`, code)

	percent := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatPercent,
	})
	code, native = percent.ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, "#,##0.0%", code)

	duration := reportfmt.Resolve(reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatDuration,
	})
	_, native = duration.ExcelFormat()
	assert.False(t, native)

	dates := reportfmt.Resolve(reportfmt.ResolveInput{Type: reportcatalog.FieldEpoch})
	code, native = dates.ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, "yyyy-mm-dd hh:mm", code)
}

func TestValidateRejectsImpossibleCombinations(t *testing.T) {
	spec := &reportfmt.Spec{Style: reportfmt.StyleCurrency}
	require.Error(t, spec.Validate(reportcatalog.FieldString))
	require.NoError(t, spec.Validate(reportcatalog.FieldDecimal))

	require.Error(
		t,
		(&reportfmt.Spec{Decimals: intPtr(99)}).Validate(reportcatalog.FieldDecimal),
	)
	require.Error(t, (&reportfmt.Spec{Currency: "US"}).Validate(reportcatalog.FieldDecimal))
	require.Error(t, (&reportfmt.Spec{Style: "wat"}).Validate(reportcatalog.FieldDecimal))
	require.NoError(t, (*reportfmt.Spec)(nil).Validate(reportcatalog.FieldDecimal))
}
