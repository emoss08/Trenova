package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePerMileDistance(t *testing.T) {
	distance := func(v float64) *float64 { return &v }

	t.Run("per-mile with a positive distance passes", func(t *testing.T) {
		require.NoError(t, validatePerMileDistance(
			shipment.CarrierRateMethodPerMile, distance(412.5),
		))
	})

	t.Run("per-mile with nil distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipment.CarrierRateMethodPerMile, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Flat")
	})

	t.Run("per-mile with zero distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipment.CarrierRateMethodPerMile, distance(0))
		require.Error(t, err)
	})

	t.Run("per-mile with negative distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipment.CarrierRateMethodPerMile, distance(-3))
		require.Error(t, err)
	})

	t.Run("flat rate ignores distance entirely", func(t *testing.T) {
		require.NoError(t, validatePerMileDistance(shipment.CarrierRateMethodFlat, nil))
		require.NoError(t, validatePerMileDistance(shipment.CarrierRateMethodFlat, distance(0)))
	})
}
