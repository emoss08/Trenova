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

func resolve(t *testing.T, in reportfmt.ResolveInput) reportfmt.Resolved {
	t.Helper()
	displays, err := reportfmt.Resolve(t.Context(), []reportfmt.ResolveInput{in})
	require.NoError(t, err)
	require.Len(t, displays, 1)
	return displays[0]
}

func formatColumn(
	t *testing.T,
	display reportfmt.Resolved,
	fieldType reportcatalog.FieldType,
	loc *time.Location,
	values ...any,
) []reportfmt.Cell {
	t.Helper()
	batch := reportfmt.NewBatch(
		[]reportfmt.Column{{Display: &display, Type: fieldType}},
		loc,
	)
	for _, value := range values {
		batch.Add([]any{value})
	}
	rows, err := batch.Format(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, len(values))

	cells := make([]reportfmt.Cell, 0, len(rows))
	for _, row := range rows {
		require.Len(t, row.Cells, 1)
		cells = append(cells, row.Cells[0])
	}
	return cells
}

func texts(cells []reportfmt.Cell) []string {
	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.Text)
	}
	return out
}

func TestFormatMoneyTamesRawQuotients(t *testing.T) {
	t.Parallel()

	display := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
	})

	cells := formatColumn(t, display, reportcatalog.FieldDecimal, time.UTC,
		decimalOf(t, "1082.0672643987443142"),
		decimalOf(t, "-1082.0672643987443142"),
		nil,
	)
	assert.Equal(t, []string{"$1,082.07", "($1,082.07)", ""}, texts(cells))
}

func TestFormatNumberOptions(t *testing.T) {
	t.Parallel()

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
		{
			name:  "exact half rounds up where a float would not",
			spec:  &reportfmt.Spec{Style: reportfmt.StyleNumber, Decimals: intPtr(2)},
			value: "1.005",
			want:  "1.01",
		},
	}

	inputs := make([]reportfmt.ResolveInput, 0, len(tests))
	for _, tt := range tests {
		inputs = append(inputs, reportfmt.ResolveInput{
			Type: reportcatalog.FieldDecimal,
			Spec: tt.spec,
		})
	}
	displays, err := reportfmt.Resolve(t.Context(), inputs)
	require.NoError(t, err)
	require.Len(t, displays, len(tests))

	columns := make([]reportfmt.Column, 0, len(tests))
	row := make([]any, 0, len(tests))
	for i, tt := range tests {
		columns = append(columns, reportfmt.Column{
			Display: &displays[i],
			Type:    reportcatalog.FieldDecimal,
		})
		row = append(row, decimalOf(t, tt.value))
	}

	cells, err := reportfmt.FormatRow(t.Context(), columns, row, time.UTC)
	require.NoError(t, err)
	for i, tt := range tests {
		assert.Equal(t, tt.want, cells[i].Text, tt.name)
	}
}

func TestFormatDurations(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			display := resolve(t, reportfmt.ResolveInput{
				Type: reportcatalog.FieldDecimal,
				Spec: &reportfmt.Spec{
					Style:         reportfmt.StyleDuration,
					DurationUnit:  reportfmt.DurationUnitSeconds,
					DurationStyle: tt.style,
					Decimals:      intPtr(1),
				},
			})
			cells := formatColumn(t, display, reportcatalog.FieldDecimal, time.UTC,
				decimalOf(t, "94200"))
			assert.Equal(t, tt.want, cells[0].Text)
		})
	}
}

func TestFormatNegativeDurationKeepsSign(t *testing.T) {
	t.Parallel()

	display := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Spec: &reportfmt.Spec{
			Style:         reportfmt.StyleDuration,
			DurationStyle: reportfmt.DurationStyleCompact,
		},
	})

	cells := formatColumn(t, display, reportcatalog.FieldDecimal, time.UTC, decimalOf(t, "-9000"))
	assert.Equal(t, "-2h 30m", cells[0].Text)
}

func TestFormatDatesUseRequestLocation(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			display := resolve(t, reportfmt.ResolveInput{
				Type: reportcatalog.FieldEpoch,
				Spec: &reportfmt.Spec{DateStyle: tt.style},
			})
			cells := formatColumn(t, display, reportcatalog.FieldEpoch, chicago, instant)
			assert.Equal(t, tt.want, cells[0].Text)
		})
	}
}

func TestFormatDateAcceptsDecimalEpoch(t *testing.T) {
	t.Parallel()

	display := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Spec: &reportfmt.Spec{Style: reportfmt.StyleDate, DateStyle: reportfmt.DateStyleDate},
	})

	cells := formatColumn(t, display, reportcatalog.FieldDecimal, time.UTC,
		decimalOf(t, "1772587800.5"))
	assert.Equal(t, "2026-03-04", cells[0].Text)
}

func TestFormatTimeValuesStayInGo(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	instant := time.Date(2026, time.March, 4, 1, 30, 0, 0, time.UTC)

	dated := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldEpoch,
		Spec: &reportfmt.Spec{DateStyle: reportfmt.DateStyleDateTime},
	})
	cells := formatColumn(t, dated, reportcatalog.FieldEpoch, chicago, instant)
	assert.Equal(t, "2026-03-03 19:30", cells[0].Text)

	raw := formatColumn(t, reportfmt.Resolved{}, reportcatalog.FieldEpoch, chicago, instant)
	assert.Equal(t, "2026-03-04T01:30:00Z", raw[0].Text)
}

func TestFormatBooleansAndText(t *testing.T) {
	t.Parallel()

	yesNo := resolve(t, reportfmt.ResolveInput{Type: reportcatalog.FieldBool})
	assert.Equal(t, []string{"Yes", "No"},
		texts(formatColumn(t, yesNo, reportcatalog.FieldBool, time.UTC, true, false)))

	check := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldBool,
		Spec: &reportfmt.Spec{BoolStyle: reportfmt.BoolStyleCheck},
	})
	assert.Equal(t, []string{"✓"},
		texts(formatColumn(t, check, reportcatalog.FieldBool, time.UTC, true)))

	dashed := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldString,
		Spec: &reportfmt.Spec{NullText: "—"},
	})
	assert.Equal(t, []string{"—", "—", "ACME"},
		texts(formatColumn(t, dashed, reportcatalog.FieldString, time.UTC, nil, "", "ACME")))
}

func TestFormatARawDisplayWritesTheValueAsIs(t *testing.T) {
	t.Parallel()

	cells := formatColumn(t, reportfmt.Resolved{}, reportcatalog.FieldDecimal, time.UTC,
		decimalOf(t, "1.50"), int64(7), true, "ACME")
	assert.Equal(t, []string{"1.5", "7", "true", "ACME"}, texts(cells))
}

func TestResolveAppliesCatalogHints(t *testing.T) {
	t.Parallel()

	displays, err := reportfmt.Resolve(t.Context(), []reportfmt.ResolveInput{
		{Type: reportcatalog.FieldDecimal, Hint: reportcatalog.FormatDistance},
		{Type: reportcatalog.FieldInt, Hint: reportcatalog.FormatWeight},
		{Type: reportcatalog.FieldInt, Hint: reportcatalog.FormatCount},
	})
	require.NoError(t, err)
	require.Len(t, displays, 3)

	cells, err := reportfmt.FormatRow(t.Context(), []reportfmt.Column{
		{Display: &displays[0], Type: reportcatalog.FieldDecimal},
		{Display: &displays[1], Type: reportcatalog.FieldInt},
		{Display: &displays[2], Type: reportcatalog.FieldInt},
	}, []any{decimalOf(t, "1284.44"), int64(42000), int64(1204)}, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, []string{"1,284.4 mi", "42,000 lbs", "1,204"}, texts(cells))
}

func TestResolveCarriesTheBaseline(t *testing.T) {
	t.Parallel()

	money := resolve(t, reportfmt.ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
		Spec: &reportfmt.Spec{Currency: "cad"},
	})
	assert.Equal(t, reportfmt.StyleCurrency, money.Style)
	assert.Equal(t, 2, money.Decimals)
	assert.Equal(t, reportfmt.NegativeParentheses, money.Negative)
	assert.Equal(t, "CAD", money.Currency)
	assert.True(t, money.Grouping)
	assert.Equal(t, reportfmt.DateStyleDateTime, money.DateStyle)
}

func TestBatchFormatsEveryRowOfEveryColumnAndStartsOver(t *testing.T) {
	t.Parallel()

	displays, err := reportfmt.Resolve(t.Context(), []reportfmt.ResolveInput{
		{Type: reportcatalog.FieldString},
		{Type: reportcatalog.FieldDecimal, Hint: reportcatalog.FormatMoney},
	})
	require.NoError(t, err)

	batch := reportfmt.NewBatch([]reportfmt.Column{
		{Display: &displays[0], Type: reportcatalog.FieldString},
		{Display: &displays[1], Type: reportcatalog.FieldDecimal},
	}, time.UTC)

	reused := []any{"ACME", decimalOf(t, "10")}
	batch.Add(reused)
	reused[0], reused[1] = "Globex", decimalOf(t, "20.5")
	batch.Add(reused)
	batch.Add([]any{"Short row"})
	assert.Equal(t, 3, batch.Len())

	rows, err := batch.Format(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []reportfmt.Cell{{Text: "ACME"}, {Text: "$10.00"}}, rows[0].Cells)
	assert.Equal(t, []reportfmt.Cell{{Text: "Globex"}, {Text: "$20.50"}}, rows[1].Cells)
	assert.Equal(t, []reportfmt.Cell{{Text: "Short row"}, {Text: ""}}, rows[2].Cells)
	assert.Equal(t, []any{"ACME", decimalOf(t, "10")}, rows[0].Values)
	assert.Equal(t, 0, batch.Len())

	empty, err := batch.Format(t.Context())
	require.NoError(t, err)
	assert.Nil(t, empty)
}

func TestBatchIsFullAtTheChunkSize(t *testing.T) {
	t.Parallel()

	batch := reportfmt.NewBatch(nil, nil)
	for range reportfmt.BatchRows - 1 {
		batch.Add(nil)
	}
	assert.False(t, batch.Full())
	batch.Add(nil)
	assert.True(t, batch.Full())
}

func TestExcelFormats(t *testing.T) {
	t.Parallel()

	displays, err := reportfmt.Resolve(t.Context(), []reportfmt.ResolveInput{
		{Type: reportcatalog.FieldDecimal, Hint: reportcatalog.FormatMoney},
		{Type: reportcatalog.FieldDecimal, Hint: reportcatalog.FormatPercent},
		{Type: reportcatalog.FieldDecimal, Hint: reportcatalog.FormatDuration},
		{Type: reportcatalog.FieldEpoch},
	})
	require.NoError(t, err)

	code, native := displays[0].ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, `"$"#,##0.00;("$"#,##0.00)`, code)

	code, native = displays[1].ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, "#,##0.0%", code)

	_, native = displays[2].ExcelFormat()
	assert.False(t, native)

	code, native = displays[3].ExcelFormat()
	assert.True(t, native)
	assert.Equal(t, "yyyy-mm-dd hh:mm", code)
}

func TestValidateRejectsImpossibleCombinations(t *testing.T) {
	t.Parallel()

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
