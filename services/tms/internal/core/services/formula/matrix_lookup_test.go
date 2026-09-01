package formula_test

import (
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
