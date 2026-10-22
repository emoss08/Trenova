package rules

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringerValue struct{}

func (stringerValue) String() string { return "from a stringer" }

func TestReportValuesEncodeNumbersExactly(t *testing.T) {
	t.Parallel()

	encoded, err := ReportValues{
		nil,
		decimal.RequireFromString("1082.0672643987443142"),
		decimal.NullDecimal{},
		int64(-7),
		0.1,
		math.Inf(1),
		true,
		`say "hi"`,
		stringerValue{},
		struct{}{},
	}.MarshalJSON()
	require.NoError(t, err)

	assert.JSONEq(t,
		`[null, 1082.0672643987443142, null, -7, 0.1, "+Inf", true, "say \"hi\"", "from a stringer", null]`,
		string(encoded),
	)
	assert.Contains(t, string(encoded), "1082.0672643987443142")
}

func TestFormatReportReadsEveryColumnBack(t *testing.T) {
	t.Parallel()

	columns, err := FormatReport(t.Context(), []ReportColumn{
		{
			Display: &ReportDisplay{
				Style:    "currency",
				Decimals: 2,
				Grouping: true,
				Currency: "USD",
				Negative: "parentheses",
				Rules: []ReportRule{{
					Op:    "gt",
					Value: decimal.NewFromInt(10000),
					Tone:  "warning",
				}},
			},
			ValueType: "decimal",
			Values:    ReportValues{decimal.RequireFromString("-1.005"), "50000", nil},
		},
		{
			Display:   &ReportDisplay{Style: "date", DateStyle: "date"},
			ValueType: "epoch",
			Values:    ReportValues{int64(1772587800)},
		},
	})
	require.NoError(t, err)
	require.Len(t, columns, 2)

	assert.Equal(t, []ReportCell{
		{Text: "($1.01)"},
		{Text: "$50,000.00", Tone: "warning"},
		{Text: ""},
	}, columns[0])
	require.Len(t, columns[1], 1)
	assert.True(t, columns[1][0].Date.Valid)
	assert.Equal(t, int64(1772587800), columns[1][0].Date.Decimal.IntPart())
}

func TestResolveReportDisplaysReadsDecimalsBack(t *testing.T) {
	t.Parallel()

	decimals := 1
	displays, err := ResolveReportDisplays(t.Context(), []ReportResolveInput{{
		Type: "decimal",
		Hint: "weight",
		Band: &ReportBand{Edges: []decimal.Decimal{decimal.Zero, decimal.RequireFromString("0.1")}},
		Spec: &ReportSpec{
			Decimals: &decimals,
			Rules: []ReportRule{{
				Op:    "between",
				Value: decimal.RequireFromString("0.2"),
				Upper: decimal.NewNullDecimal(decimal.RequireFromString("0.3")),
				Tone:  "neutral",
			}},
		},
	}})
	require.NoError(t, err)
	require.Len(t, displays, 1)

	display := displays[0]
	assert.Equal(t, "number", display.Style)
	assert.Equal(t, 1, display.Decimals)
	assert.Equal(t, " lbs", display.Suffix)
	require.NotNil(t, display.Band)
	assert.Equal(t, "0.1", display.Band.Edges[1].String())
	require.Len(t, display.Rules, 1)
	assert.Equal(t, "0.3", display.Rules[0].Upper.Decimal.String())
}

func TestReportCallsWithNothingToDoSkipTheModule(t *testing.T) {
	t.Parallel()

	columns, err := FormatReport(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, columns)

	displays, err := ResolveReportDisplays(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, displays)
}
