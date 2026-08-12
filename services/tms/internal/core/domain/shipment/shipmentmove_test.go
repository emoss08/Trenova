package shipment

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestMoveCoverageType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value MoveCoverageType
		want  bool
	}{
		{name: "unassigned", value: MoveCoverageTypeUnassigned, want: true},
		{name: "driver", value: MoveCoverageTypeDriver, want: true},
		{name: "carrier", value: MoveCoverageTypeCarrier, want: true},
		{name: "empty", value: MoveCoverageType(""), want: false},
		{name: "unknown", value: MoveCoverageType("broker"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.value.IsValid())
		})
	}
}

func TestShipmentMove_BeforeAppendModelDefaultsCoverageToUnassigned(t *testing.T) {
	t.Parallel()

	t.Run("insert without coverage is born uncovered", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{}
		require.NoError(t, move.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))

		assert.Equal(t, MoveCoverageTypeUnassigned, move.CoverageType)
		assert.False(t, move.ID.IsNil())
		assert.NotZero(t, move.CreatedAt)
	})

	t.Run("insert keeps an explicit coverage", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{CoverageType: MoveCoverageTypeCarrier}
		require.NoError(t, move.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))

		assert.Equal(t, MoveCoverageTypeCarrier, move.CoverageType)
	})

	t.Run("update without coverage is born uncovered", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{ID: pulid.MustNew("sm_")}
		require.NoError(t, move.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil)))

		assert.Equal(t, MoveCoverageTypeUnassigned, move.CoverageType)
		assert.NotZero(t, move.UpdatedAt)
	})
}

func TestShipmentMove_CoveragePredicatesAreCoverageTypeAgnostic(t *testing.T) {
	t.Parallel()

	driverAssignment := &Assignment{ID: pulid.MustNew("asn_")}
	activeCarrier := &CarrierAssignment{
		ID:     pulid.MustNew("ca_"),
		Status: CarrierAssignmentStatusConfirmed,
	}
	canceledCarrier := &CarrierAssignment{
		ID:     pulid.MustNew("ca_"),
		Status: CarrierAssignmentStatusCanceled,
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
			move: &ShipmentMove{CoverageType: MoveCoverageTypeUnassigned},
		},
		{
			name:          "driver covered move",
			move:          &ShipmentMove{CoverageType: MoveCoverageTypeDriver, Assignment: driverAssignment},
			hasCoverage:   true,
			hasAssignment: true,
		},
		{
			name:            "carrier covered move",
			move:            &ShipmentMove{CoverageType: MoveCoverageTypeCarrier, CarrierAssignment: activeCarrier},
			hasCoverage:     true,
			carrierCovered:  true,
			hasCarrierAssig: true,
		},
		{
			name:           "carrier label with a canceled carrier assignment is uncovered",
			move:           &ShipmentMove{CoverageType: MoveCoverageTypeCarrier, CarrierAssignment: canceledCarrier},
			carrierCovered: true,
		},
		{
			name:          "unassigned label still reports the driver assignment it carries",
			move:          &ShipmentMove{CoverageType: MoveCoverageTypeUnassigned, Assignment: driverAssignment},
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

	for _, coverage := range []MoveCoverageType{
		MoveCoverageTypeUnassigned,
		MoveCoverageTypeDriver,
		MoveCoverageTypeCarrier,
	} {
		t.Run(coverage.String(), func(t *testing.T) {
			t.Parallel()

			move := &ShipmentMove{Status: MoveStatusNew, CoverageType: coverage}
			multiErr := errortypes.NewMultiError()
			move.Validate(multiErr)

			assert.False(t, multiErr.HasErrors(), "%v", multiErr)
		})
	}

	t.Run("unknown coverage is rejected", func(t *testing.T) {
		t.Parallel()

		move := &ShipmentMove{Status: MoveStatusNew, CoverageType: MoveCoverageType("broker")}
		multiErr := errortypes.NewMultiError()
		move.Validate(multiErr)

		assert.True(t, multiErr.HasErrors())
	})
}
