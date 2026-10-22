package ratematrix_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupCell_TiedUnsavedCellsResolveToTheEarlierInTheRequest(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
	}
	first := &ratematrix.RateMatrixCell{D0Key: "SE", Value: decimal.RequireFromString("1")}
	second := &ratematrix.RateMatrixCell{D0Key: "SE", Value: decimal.RequireFromString("2")}
	saved := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "SE", Value: decimal.RequireFromString("3"),
	}

	match, err := ratematrix.LookupCell(
		t.Context(),
		dimensions,
		[]*ratematrix.RateMatrixCell{saved, first, second},
		&ratematrix.LookupValues{exactKeys("SE")},
	)
	require.NoError(t, err)

	assert.Same(t, first, match.Cell)
}

func TestLookupCell_ReportsTheQuantityItMatchedOnABandedAxis(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		dim(0, ratematrix.DimensionKindZone, ratematrix.MatchModeExact),
		dim(1, ratematrix.DimensionKindWeightBreak, ratematrix.MatchModeRange),
	}
	heavy := &ratematrix.RateMatrixCell{
		ID: pulid.MustNew("rmc_"), D0Key: "SE", Value: decimal.RequireFromString("12"),
	}
	heavy.D1Min, heavy.D1Max = band("1000", "")

	match, err := ratematrix.LookupCell(
		t.Context(),
		dimensions,
		[]*ratematrix.RateMatrixCell{nil, heavy},
		&ratematrix.LookupValues{exactKeys("SE", "SOUTH"), quantity("1000")},
	)
	require.NoError(t, err)

	assert.Same(t, heavy, match.Cell)
	assert.Equal(t, [ratematrix.MaxDimensions]string{"SE", "1000", "", ""}, match.MatchedKeys)
}
