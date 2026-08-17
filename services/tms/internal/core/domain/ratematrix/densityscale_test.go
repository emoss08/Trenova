package ratematrix_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func standardScale() *ratematrix.DensityScale {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return &ratematrix.DensityScale{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           ratematrix.StandardDensityScaleCode,
		Name:           "NMFC 2025 Density Scale",
		EffectiveFrom:  1,
		Tiers:          ratematrix.StandardDensityTiers(orgID, buID),
	}
}

func TestStandardScaleHasThirteenTiers(t *testing.T) {
	t.Parallel()

	assert.Len(t, standardScale().Tiers, 13)
}

func TestStandardScaleClassifiesEveryPublishedBand(t *testing.T) {
	t.Parallel()

	scale := standardScale()

	tests := []struct {
		densityPcf string
		want       commodity.FreightClass
	}{
		{"0", commodity.FreightClass400},
		{"0.9", commodity.FreightClass400},
		{"1", commodity.FreightClass300},
		{"3.5", commodity.FreightClass250},
		{"5", commodity.FreightClass175},
		{"7.2", commodity.FreightClass125},
		{"8.2", commodity.FreightClass100},
		{"11", commodity.FreightClass92_5},
		{"13.4", commodity.FreightClass85},
		{"20", commodity.FreightClass70},
		{"25", commodity.FreightClass65},
		{"32", commodity.FreightClass60},
		{"44", commodity.FreightClass55},
		{"50", commodity.FreightClass50},
		{"120", commodity.FreightClass50},
	}

	for _, tt := range tests {
		t.Run(tt.densityPcf, func(t *testing.T) {
			t.Parallel()

			class, tier, ok := scale.ClassFor(decimal.RequireFromString(tt.densityPcf))

			require.True(t, ok, "density %s should classify", tt.densityPcf)
			require.NotNil(t, tier)
			assert.Equal(t, tt.want, class)
		})
	}
}

// A density sitting exactly on a boundary belongs to the denser band, which is
// the lower class. Getting this backwards overcharges the shipment by a whole
// classification.
func TestDensityBoundaryFallsIntoTheDenserBand(t *testing.T) {
	t.Parallel()

	scale := standardScale()

	below, _, ok := scale.ClassFor(decimal.RequireFromString("7.9999"))
	require.True(t, ok)
	assert.Equal(t, commodity.FreightClass125, below)

	atBoundary, _, ok := scale.ClassFor(decimal.RequireFromString("8"))
	require.True(t, ok)
	assert.Equal(t, commodity.FreightClass100, atBoundary)
}

func TestStandardScaleValidates(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	standardScale().Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), "standard scale should be valid: %v", multiErr)
}

func TestScaleWithAGapIsRejected(t *testing.T) {
	t.Parallel()

	scale := standardScale()
	// Open a hole between the 4-6 and 6-8 bands.
	scale.Tiers[4].FromPcf = decimal.RequireFromString("7")

	multiErr := errortypes.NewMultiError()
	scale.Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "a gap in the scale must be rejected")
}

func TestScaleWithAClosedTopBandIsRejected(t *testing.T) {
	t.Parallel()

	scale := standardScale()
	scale.Tiers[len(scale.Tiers)-1].ToPcf = decimal.NewNullDecimal(
		decimal.RequireFromString("999"),
	)

	multiErr := errortypes.NewMultiError()
	scale.Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "a closed densest band leaves heavier freight unrated")
}

func TestScaleThatDoesNotStartAtZeroIsRejected(t *testing.T) {
	t.Parallel()

	scale := standardScale()
	scale.Tiers[0].FromPcf = decimal.RequireFromString("0.5")

	multiErr := errortypes.NewMultiError()
	scale.Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "the lightest band must start at zero")
}
