package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePerMileDistance(t *testing.T) {
	distance := func(v float64) *float64 { return &v }

	t.Run("per-mile with a positive distance passes", func(t *testing.T) {
		require.NoError(t, validatePerMileDistance(
			shipmentvocab.CarrierRateMethodPerMile, distance(412.5),
		))
	})

	t.Run("per-mile with nil distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipmentvocab.CarrierRateMethodPerMile, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Flat")
	})

	t.Run("per-mile with zero distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipmentvocab.CarrierRateMethodPerMile, distance(0))
		require.Error(t, err)
	})

	t.Run("per-mile with negative distance is rejected", func(t *testing.T) {
		err := validatePerMileDistance(shipmentvocab.CarrierRateMethodPerMile, distance(-3))
		require.Error(t, err)
	})

	t.Run("flat rate ignores distance entirely", func(t *testing.T) {
		require.NoError(t, validatePerMileDistance(shipmentvocab.CarrierRateMethodFlat, nil))
		require.NoError(
			t,
			validatePerMileDistance(shipmentvocab.CarrierRateMethodFlat, distance(0)),
		)
	})
}
