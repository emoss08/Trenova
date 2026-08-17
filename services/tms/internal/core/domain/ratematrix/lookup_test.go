package ratematrix_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dim(position int16, kind ratematrix.DimensionKind, mode ratematrix.MatchMode) *ratematrix.RateMatrixDimension {
	return &ratematrix.RateMatrixDimension{Position: position, Kind: kind, MatchMode: mode}
}

func exactKeys(keys ...string) ratematrix.DimensionValue {
	return ratematrix.DimensionValue{Keys: keys}
}

func quantity(value string) ratematrix.DimensionValue {
	return ratematrix.DimensionValue{
		Quantity: decimal.NewNullDecimal(decimal.RequireFromString(value)),
	}
}

func band(minimum, maximum string) (decimal.NullDecimal, decimal.NullDecimal) {
	lower := decimal.NewNullDecimal(decimal.RequireFromString(minimum))
	if maximum == "" {
		return lower, decimal.NullDecimal{}
	}

	return lower, decimal.NewNullDecimal(decimal.RequireFromString(maximum))
}

func TestSelectCellAcrossZoneAndWeightBreak(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
		dim(1, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
		dim(2, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
	}

	light := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "SE", D1Key: "MW",
		Value: decimal.RequireFromString("18.50"),
	}
	light.D2Min, light.D2Max = band("0", "1000")

	heavy := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "SE", D1Key: "MW",
		Value: decimal.RequireFromString("12.25"),
	}
	heavy.D2Min, heavy.D2Max = band("1000", "5000")

	values := ratematrix.LookupValues{
		exactKeys("SE"),
		exactKeys("MW"),
		quantity("4200"),
	}

	match, err := ratematrix.SelectCell(dimensions, []*ratematrix.RateMatrixCell{light, heavy}, values)

	require.NoError(t, err)
	assert.Equal(t, heavy.ID, match.Cell.ID)
	assert.True(t, match.Cell.Value.Equal(decimal.RequireFromString("12.25")))
}

// Weight breaks are half open, so the shipment that weighs exactly the boundary
// belongs to the band above it. Tariffs are written that way, and a closed
// interval would let two adjacent bands both claim the weight.
func TestWeightBreakBoundaryBelongsToTheUpperBand(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
	}

	lower := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), Value: decimal.RequireFromString("20"),
	}
	lower.D0Min, lower.D0Max = band("0", "1000")

	upper := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), Value: decimal.RequireFromString("15"),
	}
	upper.D0Min, upper.D0Max = band("1000", "5000")

	match, err := ratematrix.SelectCell(
		dimensions,
		[]*ratematrix.RateMatrixCell{lower, upper},
		ratematrix.LookupValues{quantity("1000")},
	)

	require.NoError(t, err)
	assert.Equal(t, upper.ID, match.Cell.ID)
}

func TestOpenEndedTopBandCatchesAnythingHeavier(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
	}

	top := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), Value: decimal.RequireFromString("9.10"),
	}
	top.D0Min, top.D0Max = band("20000", "")

	match, err := ratematrix.SelectCell(
		dimensions,
		[]*ratematrix.RateMatrixCell{top},
		ratematrix.LookupValues{quantity("48000")},
	)

	require.NoError(t, err)
	assert.Equal(t, top.ID, match.Cell.ID)
}

// A place can belong to several zones at once. The caller orders its candidate
// keys by preference, and the cell matching the earlier choice has to win —
// otherwise a broad fallback zone could outrank the specific one.
func TestEarlierCandidateKeyWins(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
	}

	preferred := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "CHI_METRO",
		Value: decimal.RequireFromString("3.10"),
	}
	fallback := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "MIDWEST",
		Value: decimal.RequireFromString("2.40"),
	}

	match, err := ratematrix.SelectCell(
		dimensions,
		// Deliberately supplied in the losing order to prove the ranking, not
		// the slice order, decides.
		[]*ratematrix.RateMatrixCell{fallback, preferred},
		ratematrix.LookupValues{exactKeys("CHI_METRO", "MIDWEST")},
	)

	require.NoError(t, err)
	assert.Equal(t, preferred.ID, match.Cell.ID)
	assert.Equal(t, 0, match.Preference)
	assert.Equal(t, "CHI_METRO", match.MatchedKeys[0])
}

// Two cells that are indistinguishable on every axis must always resolve the
// same way, whatever order the database returned them in.
func TestTieBreaksDeterministicallyByCellID(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
	}

	first := &ratematrix.RateMatrixCell{
		ID: pulid.ID("rmc_00000000000000000000000001"), D0Key: "SE",
		Value: decimal.RequireFromString("1"),
	}
	second := &ratematrix.RateMatrixCell{
		ID: pulid.ID("rmc_00000000000000000000000002"), D0Key: "SE",
		Value: decimal.RequireFromString("2"),
	}

	forward, err := ratematrix.SelectCell(
		dimensions,
		[]*ratematrix.RateMatrixCell{first, second},
		ratematrix.LookupValues{exactKeys("SE")},
	)
	require.NoError(t, err)

	reversed, err := ratematrix.SelectCell(
		dimensions,
		[]*ratematrix.RateMatrixCell{second, first},
		ratematrix.LookupValues{exactKeys("SE")},
	)
	require.NoError(t, err)

	assert.Equal(t, forward.Cell.ID, reversed.Cell.ID)
	assert.Equal(t, first.ID, forward.Cell.ID)
}

func TestSelectCellRejectsAMissingDimensionValue(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
		dim(1, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
	}

	_, err := ratematrix.SelectCell(
		dimensions,
		nil,
		ratematrix.LookupValues{exactKeys("SE")},
	)

	require.ErrorIs(t, err, ratematrix.ErrDimensionValueMissing)
}

func TestSelectCellReportsWhenNothingCovers(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
	}

	cell := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "NE", Value: decimal.RequireFromString("1"),
	}

	_, err := ratematrix.SelectCell(
		dimensions,
		[]*ratematrix.RateMatrixCell{cell},
		ratematrix.LookupValues{exactKeys("SE")},
	)

	require.ErrorIs(t, err, ratematrix.ErrNoMatchingCell)
}

func TestOrderedDimensionsSortsByPosition(t *testing.T) {
	t.Parallel()

	matrix := &ratematrix.RateMatrix{
		Dimensions: []*ratematrix.RateMatrixDimension{
			dim(2, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
			dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
			dim(1, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
		},
	}

	ordered := matrix.OrderedDimensions()

	require.Len(t, ordered, 3)
	assert.Equal(t, int16(0), ordered[0].Position)
	assert.Equal(t, int16(1), ordered[1].Position)
	assert.Equal(t, int16(2), ordered[2].Position)
}
