package reportfmt

import (
	"testing"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func moneyInBands(band *Band) Resolved {
	return Resolve(ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
		Band: band,
	})
}

// The scanned value is the range's lower bound, so a banded cell only makes
// sense when it renders as the whole range.
func TestBandLabelsRenderAsRanges(t *testing.T) {
	display := moneyInBands(&Band{Edges: []float64{0, 1000, 5000}})

	assert.Equal(t, "$0.00 – $1,000.00", display.String(decimal.NewFromInt(0), nil))
	assert.Equal(t, "$1,000.00 – $5,000.00", display.String(decimal.NewFromInt(1000), nil))
	assert.Equal(t, "$5,000.00+", display.String(decimal.NewFromInt(5000), nil))
}

func TestFixedWidthBandsAreAlwaysBounded(t *testing.T) {
	display := Resolve(ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatCount,
		Band: &Band{Width: 5},
	})

	assert.Equal(t, "0 – 5", display.String(int64(0), nil))
	assert.Equal(t, "25 – 30", display.String(int64(25), nil))
	assert.Equal(t, "-10 – -5", display.String(int64(-10), nil))
}

// A duration band reads through the column's own display, so the ranges come
// out in the units the author chose rather than raw seconds.
func TestBandLabelsFollowTheColumnDisplay(t *testing.T) {
	hours := 3600.0
	display := Resolve(ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatDuration,
		Band: &Band{Edges: []float64{0, 2 * hours, 4 * hours}},
	})

	assert.Equal(t, "0:00 – 2:00", display.String(int64(0), nil))
	assert.Equal(t, "4:00+", display.String(int64(4*hours), nil))
}

// The query groups anything below the first boundary onto that boundary, so a
// stray low value still labels as the first range rather than inventing one
// beneath it. IsLowest is what stops a drill-through re-applying that floor and
// hiding the very rows the range counted.
func TestBandLabelClampsBelowTheFirstBoundary(t *testing.T) {
	band := &Band{Edges: []float64{0, 1000}}
	display := moneyInBands(band)

	assert.Equal(t, "$0.00 – $1,000.00", display.String(decimal.NewFromInt(-250), nil))
	assert.Equal(t, "$0.00 – $1,000.00", display.String(decimal.NewFromInt(0), nil))

	assert.True(t, band.IsLowest(-250))
	assert.True(t, band.IsLowest(0))
	assert.False(t, band.IsLowest(1000))
}

// A range is a label, not a number; writing it into a native numeric Excel cell
// would show the lower bound alone and quietly change what the sheet says.
func TestBandedColumnsExportAsText(t *testing.T) {
	display := moneyInBands(&Band{Width: 1000})

	_, native := display.ExcelFormat()
	assert.False(t, native)

	unbanded := Resolve(ResolveInput{
		Type: reportcatalog.FieldDecimal,
		Hint: reportcatalog.FormatMoney,
	})
	_, native = unbanded.ExcelFormat()
	assert.True(t, native)
}

func TestBandValidation(t *testing.T) {
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

// A malformed band must not read as "no band" — that would silently drop the
// grouping the author asked for instead of telling them it was wrong.
func TestMalformedBandIsNotEmpty(t *testing.T) {
	assert.True(t, (*Band)(nil).IsEmpty())
	assert.True(t, (&Band{}).IsEmpty())

	assert.False(t, (&Band{Width: -100}).IsEmpty())
	assert.False(t, (&Band{Width: -100}).IsUsable())
	assert.False(t, (&Band{Edges: []float64{5}}).IsEmpty())
	assert.True(t, (&Band{Width: 100}).IsUsable())
}

func TestBandIsWhole(t *testing.T) {
	assert.True(t, (&Band{Width: 100}).IsWhole())
	assert.False(t, (&Band{Width: 100.5}).IsWhole())
	assert.True(t, (&Band{Edges: []float64{0, 10, 20}}).IsWhole())
	assert.False(t, (&Band{Edges: []float64{0, 10.25}}).IsWhole())
}

// Resolve must not hand out the author's slice, or a later edit to the spec
// would change a contract already sent to a renderer.
func TestResolveClonesTheBand(t *testing.T) {
	band := &Band{Edges: []float64{0, 100}}
	display := moneyInBands(band)

	band.Edges[1] = 999
	require.NotNil(t, display.Band)
	assert.InDelta(t, 100, display.Band.Edges[1], 0)
}

// A trailing unit describes the whole range, so it is written once. A leading
// currency symbol still leads each end.
func TestBandLabelWritesATrailingUnitOnce(t *testing.T) {
	weight := Resolve(ResolveInput{
		Type: reportcatalog.FieldInt,
		Hint: reportcatalog.FormatWeight,
		Band: &Band{Width: 5000},
	})
	assert.Equal(t, "10,000 – 15,000 lbs", weight.String(int64(10000), nil))

	money := moneyInBands(&Band{Width: 1000})
	assert.Equal(t, "$0.00 – $1,000.00", money.String(decimal.NewFromInt(0), nil))
}
