package formula_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

// A two-axis matrix is never addressable by single-key lookup: answering
// lookup() from only its first axis would hand back a rate belonging to some
// other slice of the grid. It lives behind lookup2 instead.
func TestMatrixLookup_TwoAxisMatrixIsNotAddressableBySingleKeyLookup(t *testing.T) {
	t.Parallel()

	data := exactLookupMatrix("class_tariff", map[string]string{"SE": "100"})
	data.Matrix.Dimensions = append(data.Matrix.Dimensions, &ratematrix.RateMatrixDimension{
		Position:  1,
		Kind:      ratematrix.DimensionKindWeightBreak,
		MatchMode: ratematrix.MatchModeRange,
	})

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{data})

	assert.False(t, lookup.Has("class_tariff"))
	assert.True(t, lookup.Has2("class_tariff"))

	_, err := lookup.Lookup("class_tariff", "SE")
	require.ErrorContains(t, err, "lookup2")
}

// The repository restricts to one- and two-axis matrices, but the provider
// does not trust that: a three-axis matrix reaching it cannot be answered by
// two keys.
func TestMatrixLookup_IgnoresAMatrixWithMoreThanTwoAxes(t *testing.T) {
	t.Parallel()

	data := exactLookupMatrix("triple_tariff", map[string]string{"SE": "100"})
	data.Matrix.Dimensions = append(data.Matrix.Dimensions,
		&ratematrix.RateMatrixDimension{
			Position:  1,
			Kind:      ratematrix.DimensionKindCustom,
			MatchMode: ratematrix.MatchModeExact,
		},
		&ratematrix.RateMatrixDimension{
			Position:  2,
			Kind:      ratematrix.DimensionKindWeightBreak,
			MatchMode: ratematrix.MatchModeRange,
		},
	)

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{data})

	assert.False(t, lookup.Has("triple_tariff"))
	assert.False(t, lookup.Has2("triple_tariff"))
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

type twoAxisCellDef struct {
	rowKey string
	rowMin string
	rowMax string
	colKey string
	colMin string
	colMax string
	value  string
}

func twoAxisLookupMatrix(
	code string,
	rowMode, colMode ratematrix.MatchMode,
	defs []twoAxisCellDef,
) *repositories.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	cells := make([]*ratematrix.RateMatrixCell, 0, len(defs))
	for _, def := range defs {
		cell := &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Key:        def.rowKey,
			D1Key:        def.colKey,
			Value:        decimal.RequireFromString(def.value),
		}
		if def.rowMin != "" {
			cell.D0Min = decimal.NewNullDecimal(decimal.RequireFromString(def.rowMin))
		}
		if def.rowMax != "" {
			cell.D0Max = decimal.NewNullDecimal(decimal.RequireFromString(def.rowMax))
		}
		if def.colMin != "" {
			cell.D1Min = decimal.NewNullDecimal(decimal.RequireFromString(def.colMin))
		}
		if def.colMax != "" {
			cell.D1Max = decimal.NewNullDecimal(decimal.RequireFromString(def.colMax))
		}
		cells = append(cells, cell)
	}

	return &repositories.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{
				{Position: 0, Kind: ratematrix.DimensionKindCustom, MatchMode: rowMode},
				{Position: 1, Kind: ratematrix.DimensionKindWeightBreak, MatchMode: colMode},
			},
		},
		Cells: cells,
	}
}

func classRatesLookup() formulatemplatetypes.RateTableLookup {
	return formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		twoAxisLookupMatrix(
			"class_rates",
			ratematrix.MatchModeExact,
			ratematrix.MatchModeRange,
			[]twoAxisCellDef{
				{rowKey: "ZONE_5", colMin: "0", colMax: "5000", value: "18.50"},
				{rowKey: "ZONE_5", colMin: "5000", colMax: "10000", value: "12.25"},
				{rowKey: "ZONE_5", colMin: "10000", value: "9.10"},
				{rowKey: "ZONE_8", colMin: "0", colMax: "5000", value: "24.00"},
			},
		),
	})
}

func TestMatrixLookup2_ExactRowRangeColumn(t *testing.T) {
	t.Parallel()

	lookup := classRatesLookup()

	value, err := lookup.Lookup2("class_rates", "ZONE_5", 8500)

	require.NoError(t, err)
	assert.InDelta(t, 12.25, value, 0.0001)
	assert.True(t, lookup.Has2("class_rates"))
	assert.False(t, lookup.Has("class_rates"))
}

func TestMatrixLookup2_ColumnUpperBoundBelongsToTheNextBand(t *testing.T) {
	t.Parallel()

	value, err := classRatesLookup().Lookup2("class_rates", "ZONE_5", 5000)

	require.NoError(t, err)
	assert.InDelta(t, 12.25, value, 0.0001)
}

func TestMatrixLookup2_OpenEndedColumnBandCatchesEverythingAbove(t *testing.T) {
	t.Parallel()

	value, err := classRatesLookup().Lookup2("class_rates", "ZONE_5", 250000)

	require.NoError(t, err)
	assert.InDelta(t, 9.10, value, 0.0001)
}

func TestMatrixLookup2_UnknownRowErrors(t *testing.T) {
	t.Parallel()

	_, err := classRatesLookup().Lookup2("class_rates", "ZONE_9", 500)

	require.ErrorContains(t, err, "no cell matching")
}

func TestMatrixLookup2_ColumnOutsideEveryBandErrors(t *testing.T) {
	t.Parallel()

	_, err := classRatesLookup().Lookup2("class_rates", "ZONE_8", 7500)

	require.ErrorContains(t, err, "no cell matching")
}

func TestMatrixLookup2_NonNumericColumnKeyAgainstRangeAxisErrors(t *testing.T) {
	t.Parallel()

	_, err := classRatesLookup().Lookup2("class_rates", "ZONE_5", "heavy")

	require.ErrorContains(t, err, "column key")
}

func TestMatrixLookup2_ExactRowExactColumn(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		twoAxisLookupMatrix(
			"lane_class",
			ratematrix.MatchModeExact,
			ratematrix.MatchModeExact,
			[]twoAxisCellDef{
				{rowKey: "ATL", colKey: "MIA", value: "1450"},
				{rowKey: "ATL", colKey: "ORD", value: "1720"},
			},
		),
	})

	value, err := lookup.Lookup2("lane_class", "ATL", "ORD")

	require.NoError(t, err)
	assert.InDelta(t, 1720, value, 0.0001)

	_, err = lookup.Lookup2("lane_class", "ATL", "JFK")
	require.ErrorContains(t, err, "no cell matching")
}

func TestMatrixLookup2_RangeRowExactColumn(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		twoAxisLookupMatrix(
			"distance_zone",
			ratematrix.MatchModeRange,
			ratematrix.MatchModeExact,
			[]twoAxisCellDef{
				{rowMin: "0", rowMax: "500", colKey: "DRY", value: "2.10"},
				{rowMin: "500", colKey: "DRY", value: "1.85"},
				{rowMin: "0", rowMax: "500", colKey: "REEFER", value: "2.65"},
			},
		),
	})

	value, err := lookup.Lookup2("distance_zone", 750, "DRY")

	require.NoError(t, err)
	assert.InDelta(t, 1.85, value, 0.0001)
}

func TestMatrixLookup2_RangeRowRangeColumn(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		twoAxisLookupMatrix(
			"distance_weight",
			ratematrix.MatchModeRange,
			ratematrix.MatchModeRange,
			[]twoAxisCellDef{
				{rowMin: "0", rowMax: "500", colMin: "0", colMax: "10000", value: "3.10"},
				{rowMin: "0", rowMax: "500", colMin: "10000", value: "2.40"},
				{rowMin: "500", colMin: "0", colMax: "10000", value: "2.80"},
				{rowMin: "500", colMin: "10000", value: "2.05"},
			},
		),
	})

	value, err := lookup.Lookup2("distance_weight", 620, 14000)

	require.NoError(t, err)
	assert.InDelta(t, 2.05, value, 0.0001)
}

func TestMatrixLookup2_SingleAxisTableErrorsWithGuidance(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("lane_rate", map[string]string{"ATL-MIA": "1450"}),
	})

	_, err := lookup.Lookup2("lane_rate", "ATL-MIA", "anything")

	require.ErrorContains(t, err, "single axis")
}

func TestMatrixLookup2_UnknownTableErrors(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup(nil)

	_, err := lookup.Lookup2("nothing", "a", "b")

	require.ErrorContains(t, err, "not found")
	assert.False(t, lookup.Has2("nothing"))
}

func validateLookupTablesService(t *testing.T) *formula.Service {
	t.Helper()

	return setupServiceWithMatrixData(t, []*repositories.RateMatrixLookupData{
		exactLookupMatrix("lane_rate", map[string]string{"ATL-MIA": "1450"}),
		twoAxisLookupMatrix(
			"class_rates",
			ratematrix.MatchModeExact,
			ratematrix.MatchModeRange,
			[]twoAxisCellDef{
				{rowKey: "ZONE_5", colMin: "0", colMax: "5000", value: "18.50"},
			},
		),
	})
}

func TestService_ValidateLookupTables_AcceptsMatchingArity(t *testing.T) {
	t.Parallel()

	svc := validateLookupTablesService(t)

	err := svc.ValidateLookupTables(
		t.Context(),
		`lookup("lane_rate", "ATL-MIA") + lookup2("class_rates", "ZONE_5", 2500)`,
		pagination.TenantInfo{},
	)

	require.NoError(t, err)
}

func TestService_ValidateLookupTables_RejectsLookupOnATwoAxisTable(t *testing.T) {
	t.Parallel()

	svc := validateLookupTablesService(t)

	err := svc.ValidateLookupTables(
		t.Context(),
		`lookup("class_rates", "ZONE_5")`,
		pagination.TenantInfo{},
	)

	require.ErrorContains(t, err, "lookup2")
}

func TestService_ValidateLookupTables_RejectsLookup2OnASingleAxisTable(t *testing.T) {
	t.Parallel()

	svc := validateLookupTablesService(t)

	err := svc.ValidateLookupTables(
		t.Context(),
		`lookup2("lane_rate", "ATL", "MIA")`,
		pagination.TenantInfo{},
	)

	require.ErrorContains(t, err, "single axis")
}

func TestService_ValidateLookupTables_RejectsUnknownTwoAxisTable(t *testing.T) {
	t.Parallel()

	svc := validateLookupTablesService(t)

	err := svc.ValidateLookupTables(
		t.Context(),
		`lookup2Or("nowhere", "a", 1, 0)`,
		pagination.TenantInfo{},
	)

	require.ErrorContains(t, err, "Unknown rate table: nowhere")
}

func TestMatrixLookup_MissIsDistinguishableFromMissingTable(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.35"}),
		rangeLookupMatrix("miles", []bandDef{{min: "0", max: "500", value: "2.5"}}),
	})

	_, err := lookup.Lookup("fsc", "GAS")
	require.Error(t, err)
	assert.True(t, errors.Is(err, formulatemplatetypes.ErrRateTableMiss), err.Error())

	_, err = lookup.Lookup("miles", 900)
	require.Error(t, err)
	assert.True(t, errors.Is(err, formulatemplatetypes.ErrRateTableMiss), err.Error())

	_, err = lookup.Lookup("nope", "DIESEL")
	require.Error(t, err)
	assert.False(t, errors.Is(err, formulatemplatetypes.ErrRateTableMiss), err.Error())

	_, err = lookup.Lookup("fsc", struct{}{})
	require.Error(t, err)
	assert.False(t, errors.Is(err, formulatemplatetypes.ErrRateTableMiss), err.Error())
}

func normalizedExactMatrix(
	code string,
	mode ratematrix.KeyNormalization,
	entries map[string]string,
) *repositories.RateMatrixLookupData {
	data := exactLookupMatrix(code, entries)
	data.Matrix.Dimensions[0].KeyNormalization = mode
	return data
}

func overflowRangeMatrix(
	code string,
	overflow ratematrix.RangeOverflow,
	bands []bandDef,
) *repositories.RateMatrixLookupData {
	data := rangeLookupMatrix(code, bands)
	data.Matrix.Dimensions[0].RangeOverflow = overflow
	return data
}

func TestMatrixLookup_KeyNormalizationModes(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		normalizedExactMatrix("zip3_zone", ratematrix.KeyNormalizationZip3, map[string]string{
			"752": "1.10",
			"300": "1.00",
		}),
		normalizedExactMatrix("lane_upper", ratematrix.KeyNormalizationUpper, map[string]string{
			"atl-mia": "1450",
		}),
		normalizedExactMatrix("lane_trim", ratematrix.KeyNormalizationTrim, map[string]string{
			" ATL-JAX ": "980",
		}),
		exactLookupMatrix("lane_plain", map[string]string{"ATL-MIA": "1450"}),
	})

	value, err := lookup.Lookup("zip3_zone", "75201-1234")
	require.NoError(t, err)
	assert.InDelta(t, 1.10, value, 0.0001, "zip3 keeps the first three digits of a ZIP")

	value, err = lookup.Lookup("zip3_zone", 30012)
	require.NoError(t, err)
	assert.InDelta(t, 1.00, value, 0.0001, "numeric ZIPs normalise the same way")

	value, err = lookup.Lookup("lane_upper", "Atl-Mia")
	require.NoError(t, err)
	assert.InDelta(t, 1450, value, 0.0001, "upper normalises both stored and looked-up keys")

	value, err = lookup.Lookup("lane_trim", "ATL-JAX")
	require.NoError(t, err)
	assert.InDelta(t, 980, value, 0.0001, "trim normalises stored keys too")

	_, err = lookup.Lookup("lane_plain", "atl-mia")
	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableMiss, "no mode means exact")
}

func TestMatrixLookup_RangeOverflowModes(t *testing.T) {
	t.Parallel()

	bands := []bandDef{
		{min: "100", max: "500", value: "2.00"},
		{min: "500", max: "1000", value: "1.50"},
		{min: "1000", max: "5000", value: "1.00"},
	}
	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		overflowRangeMatrix("strict", ratematrix.RangeOverflowError, bands),
		overflowRangeMatrix("clamped", ratematrix.RangeOverflowClampToTopBand, bands),
		overflowRangeMatrix("nearest", ratematrix.RangeOverflowNearest, bands),
	})

	_, err := lookup.Lookup("strict", 9000)
	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableMiss)

	value, err := lookup.Lookup("clamped", 9000)
	require.NoError(t, err)
	assert.InDelta(t, 1.00, value, 0.0001, "above the top band prices at the top band")

	_, err = lookup.Lookup("clamped", 50)
	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableMiss, "clamping only covers the top")

	value, err = lookup.Lookup("nearest", 50)
	require.NoError(t, err)
	assert.InDelta(t, 2.00, value, 0.0001, "below the bottom band prices at the bottom band")

	value, err = lookup.Lookup("nearest", 9000)
	require.NoError(t, err)
	assert.InDelta(t, 1.00, value, 0.0001)

	explainer, isExplainer := lookup.(formulatemplatetypes.LookupExplainer)
	require.True(t, isExplainer)
	match, ok := explainer.ExplainLookup("clamped", 9000)
	require.True(t, ok)
	require.NotNil(t, match.BandMin)
	assert.Equal(t, "1000", match.BandMin.String(), "the receipt names the band actually used")
	assert.True(t, match.Adjusted, "the receipt says the key was moved into a band")
}

func TestMatrixLookup_TwoAxisRowOverflowAndColumnNormalization(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	cell := func(rowMin, rowMax, colKey, value string) *ratematrix.RateMatrixCell {
		c := &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Min:        decimal.NewNullDecimal(decimal.RequireFromString(rowMin)),
			D1Key:        colKey,
			Value:        decimal.RequireFromString(value),
		}
		if rowMax != "" {
			c.D0Max = decimal.NewNullDecimal(decimal.RequireFromString(rowMax))
		}
		return c
	}
	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: "weight_zone",
			Dimensions: []*ratematrix.RateMatrixDimension{
				{
					Position:      0,
					Kind:          ratematrix.DimensionKindWeightBreak,
					MatchMode:     ratematrix.MatchModeRange,
					RangeOverflow: ratematrix.RangeOverflowClampToTopBand,
				},
				{
					Position:         1,
					Kind:             ratematrix.DimensionKindZone,
					MatchMode:        ratematrix.MatchModeExact,
					KeyNormalization: ratematrix.KeyNormalizationUpper,
				},
			},
		},
		Cells: []*ratematrix.RateMatrixCell{
			cell("0", "1000", "se", "20"),
			cell("1000", "5000", "se", "15"),
			cell("0", "1000", "mw", "22"),
		},
	}})

	value, err := lookup.Lookup2("weight_zone", 9000, "SE")
	require.NoError(t, err)
	assert.InDelta(t, 15, value, 0.0001, "row clamps to the top band and the column key is upper-cased")

	_, err = lookup.Lookup2("weight_zone", 9000, "MW")
	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableMiss, "clamping never invents a cell")
}

func TestMatrixLookup_LookupInterpBetweenBandFloors(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		rangeLookupMatrix("fuel_curve", []bandDef{
			{min: "2.00", max: "3.00", value: "0.10"},
			{min: "3.00", max: "4.00", value: "0.20"},
			{min: "4.00", max: "", value: "0.40"},
		}),
		exactLookupMatrix("zones", map[string]string{"A": "1"}),
	})
	banded, ok := lookup.(formulatemplatetypes.BandedLookup)
	require.True(t, ok, "matrix lookups support banded helpers")

	value, err := banded.LookupInterp("fuel_curve", 3.5)
	require.NoError(t, err)
	assert.InDelta(t, 0.30, value, 0.0001, "halfway between the 3.00 and 4.00 floors")

	value, err = banded.LookupInterp("fuel_curve", 2.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.10, value, 0.0001, "on a floor returns that band's value")

	value, err = banded.LookupInterp("fuel_curve", 1.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.10, value, 0.0001, "below the first floor holds the first value")

	value, err = banded.LookupInterp("fuel_curve", 9.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.40, value, 0.0001, "past the last floor holds the last value")

	_, err = banded.LookupInterp("zones", 1)
	require.Error(t, err, "a keyed table has nothing to interpolate")

	_, err = banded.LookupInterp("nope", 1)
	require.Error(t, err)
}

func TestMatrixLookup_DeficitWeightRatesAsNextBreakWhenCheaper(t *testing.T) {
	t.Parallel()

	cwt := []bandDef{
		{min: "0", max: "500", value: "30"},
		{min: "500", max: "1000", value: "20"},
		{min: "1000", max: "2000", value: "15"},
		{min: "2000", max: "", value: "12"},
	}
	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		rangeLookupMatrix("cwt", cwt),
		rangeLookupMatrix("capped", cwt[:3]),
	})
	banded, ok := lookup.(formulatemplatetypes.BandedLookup)
	require.True(t, ok)

	weight, err := banded.DeficitWeight("cwt", 450)
	require.NoError(t, err)
	assert.InDelta(t, 500, weight, 0.0001, "450 @ 30 costs more than 500 @ 20, so bill 500")

	weight, err = banded.DeficitWeight("cwt", 300)
	require.NoError(t, err)
	assert.InDelta(t, 300, weight, 0.0001, "300 @ 30 is cheaper than 500 @ 20")

	weight, err = banded.DeficitWeight("cwt", 2500)
	require.NoError(t, err)
	assert.InDelta(t, 2500, weight, 0.0001, "the top band has no next break")

	_, err = banded.DeficitWeight("capped", 9000)
	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableMiss, "a strict table still misses")
}
