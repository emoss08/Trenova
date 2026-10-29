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
