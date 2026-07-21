package costingservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateUpdate(t *testing.T) {
	t.Parallel()

	validator := costingservice.NewTestValidator()

	t.Run("valid control passes", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		multiErr := validator.ValidateUpdate(t.Context(), control)
		assert.Nil(t, multiErr)
	})

	t.Run("gl actuals with invalid rolling months rejected", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.GLActualsEnabled = true
		control.GLRollingMonths = 0

		multiErr := validator.ValidateUpdate(t.Context(), control)
		require.NotNil(t, multiErr)
		assert.Contains(t, multiErr.ToJSON(), "glRollingMonths")
	})

	t.Run("live fuel without index rejected", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.UseLiveFuelPrice = true
		control.FuelIndexID = nil

		multiErr := validator.ValidateUpdate(t.Context(), control)
		require.NotNil(t, multiErr)
		assert.Contains(t, multiErr.ToJSON(), "fuelIndexId")
	})
}

func TestValidator_ValidateCategoryUpdate(t *testing.T) {
	t.Parallel()

	validator := costingservice.NewTestValidator()

	t.Run("override without rate rejected", func(t *testing.T) {
		t.Parallel()

		category := &costingcontrol.CostCategory{
			ID:               pulid.MustNew("ccat_"),
			BusinessUnitID:   testBuID,
			OrganizationID:   testOrgID,
			CostingControlID: pulid.MustNew("cstc_"),
			Category:         costingcontrol.CategoryTypeFuel,
			Name:             "Fuel",
			CostBehavior:     costingcontrol.CostBehaviorVariable,
			RateSource:       costingcontrol.RateSourceOverride,
		}

		multiErr := validator.ValidateCategoryUpdate(t.Context(), category)
		require.NotNil(t, multiErr)
		assert.Contains(t, multiErr.ToJSON(), "overrideRatePerMile")
	})
}
