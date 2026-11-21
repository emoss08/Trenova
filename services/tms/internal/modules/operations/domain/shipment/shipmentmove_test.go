package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentMove_BeforeCreateDefaultsCoverageToUnassigned(t *testing.T) {
	t.Parallel()

	t.Run("insert without coverage is born uncovered", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{}
		require.NoError(t, move.BeforeCreate(t.Context()))

		assert.Equal(t, shipmentvocab.MoveCoverageTypeUnassigned, move.CoverageType)
	})

	t.Run("insert keeps an explicit coverage", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{CoverageType: shipmentvocab.MoveCoverageTypeCarrier}
		require.NoError(t, move.BeforeCreate(t.Context()))

		assert.Equal(t, shipmentvocab.MoveCoverageTypeCarrier, move.CoverageType)
	})

	t.Run("update without coverage is born uncovered", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{ID: pulid.MustNew("sm_")}
		require.NoError(t, move.BeforeUpdate(t.Context()))

		assert.Equal(t, shipmentvocab.MoveCoverageTypeUnassigned, move.CoverageType)
	})
}

func TestShipmentMove_CoveragePredicatesAreCoverageTypeAgnostic(t *testing.T) {
	t.Parallel()

	driverAssignment := &Assignment{ID: pulid.MustNew("asn_")}
	activeCarrier := &CarrierAssignment{
		ID:     pulid.MustNew("ca_"),
		Status: shipmentvocab.CarrierAssignmentStatusConfirmed,
	}
	canceledCarrier := &CarrierAssignment{
		ID:     pulid.MustNew("ca_"),
		Status: shipmentvocab.CarrierAssignmentStatusCanceled,
	}

	tests := []struct {
		name            string
		move            *ShipmentMove
		hasCoverage     bool
		carrierCovered  bool
		hasAssignment   bool
		hasCarrierAssig bool
	}{
		{
			name: "unassigned move with nothing attached",
			move: &ShipmentMove{CoverageType: shipmentvocab.MoveCoverageTypeUnassigned},
		},
		{
			name: "driver covered move",
			move: &ShipmentMove{
				CoverageType: shipmentvocab.MoveCoverageTypeDriver,
				Assignment:   driverAssignment,
			},
			hasCoverage:   true,
			hasAssignment: true,
		},
		{
			name: "carrier covered move",
			move: &ShipmentMove{
				CoverageType:      shipmentvocab.MoveCoverageTypeCarrier,
				CarrierAssignment: activeCarrier,
			},
			hasCoverage:     true,
			carrierCovered:  true,
			hasCarrierAssig: true,
		},
		{
			name: "carrier label with a canceled carrier assignment is uncovered",
			move: &ShipmentMove{
				CoverageType:      shipmentvocab.MoveCoverageTypeCarrier,
				CarrierAssignment: canceledCarrier,
			},
			carrierCovered: true,
		},
		{
			name: "unassigned label still reports the driver assignment it carries",
			move: &ShipmentMove{
				CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
				Assignment:   driverAssignment,
			},
			hasCoverage:   true,
			hasAssignment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.hasCoverage, tt.move.HasCoverage())
			assert.Equal(t, tt.carrierCovered, tt.move.IsCarrierCovered())
			assert.Equal(t, tt.hasAssignment, tt.move.HasAssignment())
			assert.Equal(t, tt.hasCarrierAssig, tt.move.HasCarrierAssignment())
		})
	}
}

func TestShipmentMove_ValidateAcceptsEveryCoverageType(t *testing.T) {
	t.Parallel()

	for _, coverage := range []shipmentvocab.MoveCoverageType{
		shipmentvocab.MoveCoverageTypeUnassigned,
		shipmentvocab.MoveCoverageTypeDriver,
		shipmentvocab.MoveCoverageTypeCarrier,
	} {
		t.Run(coverage.String(), func(t *testing.T) {
			t.Parallel()

			move := &ShipmentMove{Status: shipmentvocab.MoveStatusNew, CoverageType: coverage}
			multiErr := errortypes.NewMultiError()
			move.Validate(multiErr)

			assert.False(t, multiErr.HasErrors(), "%v", multiErr)
		})
	}

	t.Run("unknown coverage is rejected", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{
			Status:       shipmentvocab.MoveStatusNew,
			CoverageType: shipmentvocab.MoveCoverageType("broker"),
		}
		multiErr := errortypes.NewMultiError()
		move.Validate(multiErr)

		assert.True(t, multiErr.HasErrors())
	})
}

func TestShipmentMove_PurposeDistancesFallBackToTheRoutedDistance(t *testing.T) {
	t.Parallel()

	routed, paid, billed, fuel := 500.0, 512.5, 480.25, 505.0

	t.Run("a purpose routed with its own profile reads its own distance", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{
			Distance:        &routed,
			PayDistance:     &paid,
			BillingDistance: &billed,
			FuelDistance:    &fuel,
		}

		assert.Equal(t, &paid, move.PaidDistance())
		assert.Equal(t, &billed, move.BilledDistance())
		assert.Equal(t, &fuel, move.FuelTaxDistance())
	})

	t.Run("a purpose sharing the move's profile reads the move's distance", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{Distance: &routed}

		assert.Equal(t, &routed, move.PaidDistance())
		assert.Equal(t, &routed, move.BilledDistance())
		assert.Equal(t, &routed, move.FuelTaxDistance())
	})

	t.Run("a move with no distance has none for any purpose", func(t *testing.T) {
		t.Parallel()

		var nilMove *ShipmentMove
		assert.Nil(t, (&ShipmentMove{}).PaidDistance())
		assert.Nil(t, nilMove.BilledDistance())
	})

	t.Run("clearing drops every purpose distance", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{
			Distance:        &routed,
			PayDistance:     &paid,
			BillingDistance: &billed,
			FuelDistance:    &fuel,
		}
		move.ClearPurposeDistances()

		assert.Nil(t, move.PayDistance)
		assert.Nil(t, move.BillingDistance)
		assert.Nil(t, move.FuelDistance)
	})
}

func TestShipmentMove_ValidateRejectsNegativePurposeDistances(t *testing.T) {
	t.Parallel()

	negative := -1.0
	move := &ShipmentMove{
		Status:          shipmentvocab.MoveStatusNew,
		PayDistance:     &negative,
		BillingDistance: &negative,
		FuelDistance:    &negative,
	}
	multiErr := errortypes.NewMultiError()
	move.Validate(multiErr)

	assert.True(t, multiErr.HasErrorAt("payDistance"))
	assert.True(t, multiErr.HasErrorAt("billingDistance"))
	assert.True(t, multiErr.HasErrorAt("fuelDistance"))
}
