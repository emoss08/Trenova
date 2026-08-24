package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exactLookupMatrix(code string, entries map[string]string) *repositories.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	cells := make([]*ratematrix.RateMatrixCell, 0, len(entries))
	for key, value := range entries {
		cells = append(cells, &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Key:        key,
			Value:        decimal.RequireFromString(value),
		})
	}

	return &repositories.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{{
				Position:  0,
				Kind:      ratematrix.DimensionKindCustom,
				MatchMode: ratematrix.MatchModeExact,
			}},
		},
		Cells: cells,
	}
}

type bandDef struct {
	min   string
	max   string
	value string
}

func rangeLookupMatrix(code string, bands []bandDef) *repositories.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	cells := make([]*ratematrix.RateMatrixCell, 0, len(bands))
	for _, band := range bands {
		cell := &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Min:        decimal.NewNullDecimal(decimal.RequireFromString(band.min)),
			Value:        decimal.RequireFromString(band.value),
		}
		if band.max != "" {
			cell.D0Max = decimal.NewNullDecimal(decimal.RequireFromString(band.max))
		}
		cells = append(cells, cell)
	}

	return &repositories.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{{
				Position:  0,
				Kind:      ratematrix.DimensionKindQuantity,
				MatchMode: ratematrix.MatchModeRange,
			}},
		},
		Cells: cells,
	}
}

func TestMatrixLookup_ResolvesAnExactKey(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("lane_rate", map[string]string{
			"ATL-MIA": "1450",
			"ATL-JAX": "980",
		}),
	})

	value, err := lookup.Lookup("lane_rate", "ATL-MIA")

	require.NoError(t, err)
	assert.InDelta(t, 1450, value, 0.0001)
	assert.True(t, lookup.Has("lane_rate"))
}

func TestMatrixLookup_UnknownKeyOnAnExactTableErrors(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("lane_rate", map[string]string{"ATL-MIA": "1450"}),
	})

	_, err := lookup.Lookup("lane_rate", "ATL-ORD")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ATL-ORD")
}

func TestMatrixLookup_PicksTheBandContainingTheQuantity(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		rangeLookupMatrix("fuel_surcharge", []bandDef{
			{min: "0", max: "3", value: "0"},
			{min: "3", max: "3.5", value: "0.12"},
			{min: "3.5", max: "4", value: "0.18"},
			{min: "4", value: "0.25"},
		}),
	})

	value, err := lookup.Lookup("fuel_surcharge", 3.2)

	require.NoError(t, err)
	assert.InDelta(t, 0.12, value, 0.0001)
}

// Bands are half open — the value sitting exactly on an upper bound belongs to
// the next band up. This is how the matrix's own ContainsQuantity matches, how
// the old rate table matched, and how tariffs are written: a "3 to 3.5" tier
// means 3.5 rates at the next tier.
func TestMatrixLookup_UpperBoundBelongsToTheNextBand(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		rangeLookupMatrix("fuel_surcharge", []bandDef{
			{min: "3", max: "3.5", value: "0.12"},
			{min: "3.5", max: "4", value: "0.18"},
		}),
	})

	value, err := lookup.Lookup("fuel_surcharge", 3.5)

	require.NoError(t, err)
	assert.InDelta(t, 0.18, value, 0.0001)
}

func TestMatrixLookup_OpenEndedTopBandCatchesEverythingAbove(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		rangeLookupMatrix("fuel_surcharge", []bandDef{
			{min: "0", max: "4", value: "0.10"},
			{min: "4", value: "0.25"},
		}),
	})

	value, err := lookup.Lookup("fuel_surcharge", 99)

	require.NoError(t, err)
	assert.InDelta(t, 0.25, value, 0.0001)
}

func TestMatrixLookup_UnknownTableErrors(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup(nil)

	_, err := lookup.Lookup("nothing", "key")

	require.Error(t, err)
	assert.False(t, lookup.Has("nothing"))
}

// The repository restricts to single-axis matrices, but the provider does not
// trust that: a multi-axis matrix reaching it would index cells by only their
// first axis and hand back a rate belonging to some other slice of the grid.
func TestMatrixLookup_IgnoresAMatrixWithMoreThanOneAxis(t *testing.T) {
	t.Parallel()

	data := exactLookupMatrix("class_tariff", map[string]string{"SE": "100"})
	data.Matrix.Dimensions = append(data.Matrix.Dimensions, &ratematrix.RateMatrixDimension{
		Position:  1,
		Kind:      ratematrix.DimensionKindWeightBreak,
		MatchMode: ratematrix.MatchModeRange,
	})

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{data})

	assert.False(t, lookup.Has("class_tariff"))
}

func TestMatrixLookup_NumericKeysMatchExactTables(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("zone_rate", map[string]string{"3": "215"}),
	})

	value, err := lookup.Lookup("zone_rate", 3)

	require.NoError(t, err)
	assert.InDelta(t, 215, value, 0.0001)
}
