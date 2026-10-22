package formula_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exactLookupMatrix(code string, entries map[string]string) *ratematrixports.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	cells := make([]*ratematrix.RateMatrixCell, 0, len(entries))
	for _, key := range keys {
		cells = append(cells, &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Key:        key,
			Value:        decimal.RequireFromString(entries[key]),
		})
	}

	return &ratematrixports.RateMatrixLookupData{
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

func rangeLookupMatrix(code string, bands []bandDef) *ratematrixports.RateMatrixLookupData {
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

	return &ratematrixports.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{{
				Position:      0,
				Kind:          ratematrix.DimensionKindQuantity,
				MatchMode:     ratematrix.MatchModeRange,
				RangeOverflow: ratematrix.RangeOverflowClampToTopBand,
			}},
		},
		Cells: cells,
	}
}

func zoneWeightMatrix() *ratematrixports.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")
	cell := func(zone, low, high, value string) *ratematrix.RateMatrixCell {
		entry := &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Key:        zone,
			D1Min:        decimal.NewNullDecimal(decimal.RequireFromString(low)),
			Value:        decimal.RequireFromString(value),
		}
		if high != "" {
			entry.D1Max = decimal.NewNullDecimal(decimal.RequireFromString(high))
		}
		return entry
	}

	return &ratematrixports.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: "zone_weight",
			Dimensions: []*ratematrix.RateMatrixDimension{
				{
					Position:  1,
					Kind:      ratematrix.DimensionKindWeightBreak,
					MatchMode: ratematrix.MatchModeRange,
				},
				{
					Position:         0,
					Kind:             ratematrix.DimensionKindZone,
					MatchMode:        ratematrix.MatchModeExact,
					KeyNormalization: ratematrix.KeyNormalizationUpper,
				},
			},
		},
		Cells: []*ratematrix.RateMatrixCell{
			cell("Z1", "0", "5000", "18.50"),
			cell("Z1", "5000", "", "15.25"),
		},
	}
}

func TestNewRateTables_EncodesEachMatrixFieldForFieldInAxisOrder(t *testing.T) {
	t.Parallel()

	tables, err := formula.NewRateTables([]*ratematrixports.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.350000", "GAS": "0.1"}),
		rangeLookupMatrix("tiers", []bandDef{{"0", "100", "2"}, {"100", "", "1.5"}}),
		zoneWeightMatrix(),
		nil,
		{Matrix: &ratematrix.RateMatrix{Code: "three", Dimensions: []*ratematrix.RateMatrixDimension{
			{Position: 0}, {Position: 1}, {Position: 2},
		}}},
	}, "3:7:1700000000")
	require.NoError(t, err)

	assert.Equal(t, "3:7:1700000000", tables.Stamp)
	require.Len(t, tables.Tables, 3, "a matrix with more than two axes is no rate table")

	assert.Equal(
		t,
		`{"name":"fsc","axes":[{"match":"Exact","kind":"Custom"}],`+
			`"cells":[{"at":[{"key":"DIESEL"}],"value":"0.35"},{"at":[{"key":"GAS"}],"value":"0.1"}]}`,
		string(tables.Tables[0].Encoded),
	)
	assert.Equal(
		t,
		`{"name":"tiers","axes":[{"match":"Range","kind":"Quantity","overflow":"ClampToTopBand"}],`+
			`"cells":[{"at":[{"min":"0","max":"100"}],"value":"2"},{"at":[{"min":"100"}],"value":"1.5"}]}`,
		string(tables.Tables[1].Encoded),
	)
	assert.Equal(
		t,
		`{"name":"zone_weight","axes":[{"match":"Exact","kind":"Zone","normalization":"Upper"},`+
			`{"match":"Range","kind":"WeightBreak"}],`+
			`"cells":[{"at":[{"key":"Z1"},{"min":"0","max":"5000"}],"value":"18.5"},`+
			`{"at":[{"key":"Z1"},{"min":"5000"}],"value":"15.25"}]}`,
		string(tables.Tables[2].Encoded),
	)

	assert.Equal(t, 1, tables.Tables[0].Axes)
	assert.False(t, tables.Tables[0].Banded)
	assert.True(t, tables.Tables[1].Banded)
	assert.Equal(t, 2, tables.Tables[2].Axes)
}

func TestNewRateTables_ALaterMatrixOfTheSameNameAndShapeReplacesTheEarlier(t *testing.T) {
	t.Parallel()

	tables, err := formula.NewRateTables([]*ratematrixports.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.30"}),
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.35"}),
	}, "")
	require.NoError(t, err)

	require.Len(t, tables.Tables, 1)
	assert.Contains(t, string(tables.Tables[0].Encoded), `"value":"0.35"`)
}

func TestRateTables_SelectSendsOnlyTheTablesAFormulaNames(t *testing.T) {
	t.Parallel()

	tables, err := formula.NewRateTables([]*ratematrixports.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.35"}),
		rangeLookupMatrix("tiers", []bandDef{{"0", "", "2"}}),
	}, "stamp")
	require.NoError(t, err)

	none := tables.Select("baseRate * totalDistance")
	assert.Equal(t, rules.RawJSON("[]"), none.Encoded, "a formula with no lookup reads no table")
	assert.Empty(t, none.Shapes)

	one := tables.Select(`lookup("fsc", "DIESEL") * 2`, `"tiers" == laneCode ? 1 : 0`)
	assert.Equal(t, []rules.FormulaTable{
		{Name: "fsc", Axes: 1},
		{Name: "tiers", Axes: 1, Banded: true},
	}, one.Shapes, "a quoted name in any line of the rating counts")

	single := tables.Select("lookupInterp('tiers', weight)")
	assert.Equal(t, []rules.FormulaTable{{Name: "tiers", Axes: 1, Banded: true}}, single.Shapes)
	assert.Equal(t, "stamp", single.Stamp)

	escaped := tables.Select(`lookup("f\x73c", 1)`)
	assert.Len(t, escaped.Shapes, 2, "a name written with escapes sends every table")

	var absent *formulatemplatetypes.RateTables
	assert.Nil(t, absent.Select("lookup(\"fsc\", 1)"), "no loaded tables is not the same as none")
}
