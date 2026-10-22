package reportfmt

import (
	"testing"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolveOne(t *testing.T, in ResolveInput) Resolved {
	t.Helper()
	displays, err := Resolve(t.Context(), []ResolveInput{in})
	require.NoError(t, err)
	require.Len(t, displays, 1)
	return displays[0]
}

func formatValues(t *testing.T, display Resolved, fieldType reportcatalog.FieldType, values ...any) []Cell {
	t.Helper()
	batch := NewBatch([]Column{{Display: &display, Type: fieldType}}, nil)
	for _, value := range values {
		batch.Add([]any{value})
	}
	rows, err := batch.Format(t.Context())
	require.NoError(t, err)
	cells := make([]Cell, 0, len(rows))
	for _, row := range rows {
		cells = append(cells, row.Cells[0])
	}
	return cells
}

func formatTexts(t *testing.T, display Resolved, fieldType reportcatalog.FieldType, values ...any) []string {
	t.Helper()
	cells := formatValues(t, display, fieldType, values...)
	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.Text)
	}
	return out
}

func moneyInBands(t *testing.T, band *Band) Resolved {
	t.Helper()
	return resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
		Band: band,
	})
}

func TestBandLabelsRenderAsRanges(t *testing.T) {
	t.Parallel()

	display := moneyInBands(t, &Band{Edges: []float64{0, 1000, 5000}})

	assert.Equal(t,
		[]string{"$0.00 – $1,000.00", "$1,000.00 – $5,000.00", "$5,000.00+"},
		formatTexts(t, display, reportcatalog.FieldDecimal,
			decimal.NewFromInt(0), decimal.NewFromInt(1000), decimal.NewFromInt(5000)),
	)
}

func TestFixedWidthBandsAreAlwaysBounded(t *testing.T) {
	t.Parallel()

	display := resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatCount,
		Band: &Band{Width: 5},
	})

	assert.Equal(t,
		[]string{"0 – 5", "25 – 30", "-10 – -5"},
		formatTexts(t, display, reportcatalog.FieldInt, int64(0), int64(25), int64(-10)),
	)
}

func TestBandBoundsAreExactDecimals(t *testing.T) {
	t.Parallel()

	display := resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Spec: &Spec{Style: StyleNumber, Decimals: intOf(1)},
		Band: &Band{Width: 0.1},
	})

	assert.Equal(t, []string{"0.2 – 0.3"},
		formatTexts(t, display, reportcatalog.FieldDecimal, decimal.RequireFromString("0.2")))
}

func TestBandLabelsFollowTheColumnDisplay(t *testing.T) {
	t.Parallel()

	hours := 3600.0
	display := resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatDuration,
		Band: &Band{Edges: []float64{0, 2 * hours, 4 * hours}},
	})

	assert.Equal(t, []string{"0:00 – 2:00", "4:00+"},
		formatTexts(t, display, reportcatalog.FieldInt, int64(0), int64(4*hours)))
}

func TestBandLabelClampsBelowTheFirstBoundary(t *testing.T) {
	t.Parallel()

	band := &Band{Edges: []float64{0, 1000}}
	display := moneyInBands(t, band)

	assert.Equal(t,
		[]string{"$0.00 – $1,000.00", "$0.00 – $1,000.00"},
		formatTexts(t, display, reportcatalog.FieldDecimal,
			decimal.NewFromInt(-250), decimal.NewFromInt(0)),
	)

	assert.True(t, band.IsLowest(-250))
	assert.True(t, band.IsLowest(0))
	assert.False(t, band.IsLowest(1000))
}

func TestAMalformedBandRendersTheValueItself(t *testing.T) {
	t.Parallel()

	display := moneyInBands(t, &Band{Edges: []float64{500}})
	require.NotNil(t, display.Band)

	assert.Equal(t, []string{"$700.00"},
		formatTexts(t, display, reportcatalog.FieldDecimal, decimal.NewFromInt(700)))
}

func TestBandLabelWritesATrailingUnitOnce(t *testing.T) {
	t.Parallel()

	weight := resolveOne(t, ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatWeight,
		Band: &Band{Width: 5000},
	})
	assert.Equal(t, []string{"10,000 – 15,000 lbs"},
		formatTexts(t, weight, reportcatalog.FieldInt, int64(10000)))

	money := moneyInBands(t, &Band{Width: 1000})
	assert.Equal(t, []string{"$0.00 – $1,000.00"},
		formatTexts(t, money, reportcatalog.FieldDecimal, decimal.NewFromInt(0)))
}

func TestBandedColumnsExportAsText(t *testing.T) {
	t.Parallel()

	display := moneyInBands(t, &Band{Width: 1000})
	_, native := display.ExcelFormat()
	assert.False(t, native)

	unbanded := moneyInBands(t, nil)
	_, native = unbanded.ExcelFormat()
	assert.True(t, native)
}

func TestResolveReturnsItsOwnBand(t *testing.T) {
	t.Parallel()

	band := &Band{Edges: []float64{0, 100.5}}
	display := moneyInBands(t, band)

	band.Edges[1] = 999
	require.NotNil(t, display.Band)
	assert.Equal(t, []float64{0, 100.5}, display.Band.Edges)
}

func TestBandValidation(t *testing.T) {
	t.Parallel()

	require.NoError(t, (*Band)(nil).Validate())
	require.NoError(t, (&Band{Width: 100}).Validate())
	require.NoError(t, (&Band{Edges: []float64{0, 1}}).Validate())

	require.ErrorContains(t, (&Band{Width: 1, Edges: []float64{0, 1}}).Validate(), "not both")
	require.ErrorContains(t, (&Band{Width: -1}).Validate(), "positive, finite")
	require.ErrorContains(t, (&Band{Edges: []float64{5}}).Validate(), "at least two")
	require.ErrorContains(t, (&Band{Edges: []float64{5, 5}}).Validate(), "ascend")

	tooMany := make([]float64, MaxBandEdges+1)
	for i := range tooMany {
		tooMany[i] = float64(i)
	}
	require.ErrorContains(t, (&Band{Edges: tooMany}).Validate(), "at most")
}

func TestMalformedBandIsNotEmpty(t *testing.T) {
	t.Parallel()

	assert.True(t, (*Band)(nil).IsEmpty())
	assert.True(t, (&Band{}).IsEmpty())

	assert.False(t, (&Band{Width: -100}).IsEmpty())
	assert.False(t, (&Band{Width: -100}).IsUsable())
	assert.False(t, (&Band{Edges: []float64{5}}).IsEmpty())
	assert.True(t, (&Band{Width: 100}).IsUsable())
}

func TestBandIsWhole(t *testing.T) {
	t.Parallel()

	assert.True(t, (&Band{Width: 100}).IsWhole())
	assert.False(t, (&Band{Width: 100.5}).IsWhole())
	assert.True(t, (&Band{Edges: []float64{0, 10, 20}}).IsWhole())
	assert.False(t, (&Band{Edges: []float64{0, 10.25}}).IsWhole())
}

func TestBandBounds(t *testing.T) {
	t.Parallel()

	band := &Band{Edges: []float64{0, 1000, 5000}}
	lower, upper, bounded := band.Bounds(1200)
	assert.InDelta(t, 1000, lower, 0)
	assert.InDelta(t, 5000, upper, 0)
	assert.True(t, bounded)

	lower, _, bounded = band.Bounds(7000)
	assert.InDelta(t, 5000, lower, 0)
	assert.False(t, bounded)
}

func intOf(v int) *int { return &v }
