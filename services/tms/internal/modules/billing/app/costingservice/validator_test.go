package costingservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refusal(t *testing.T, err error) *errortypes.MultiError {
	t.Helper()

	require.Error(t, err)
	multiErr, ok := errortypes.RefusalOf(err)
	require.True(t, ok, "expected a refusal, got %v", err)
	return multiErr
}

func refusedFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field())
	}
	return fields
}

func glActualCategory(id string) *costingcontrol.CostCategory {
	return &costingcontrol.CostCategory{
		ID:                   pulid.ID(id),
		BusinessUnitID:       testBuID,
		OrganizationID:       testOrgID,
		Category:             costingcontrol.CategoryTypeFuel,
		Name:                 "Fuel",
		CostBehavior:         costingcontrol.CostBehaviorVariable,
		RateSource:           costingcontrol.RateSourceGLActual,
		BenchmarkRatePerMile: decimal.RequireFromString("0.5"),
		IsActive:             true,
	}
}

func controlPlan(control *costingcontrol.CostingControl) *costingservice.SavePlan {
	return &costingservice.SavePlan{Original: control, Control: control, ControlChanged: true}
}

func TestValidator_ValidateSave_Control(t *testing.T) {
	t.Parallel()

	validator := costingservice.NewValidator()

	t.Run("valid control passes", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.Categories = nil

		assert.NoError(t, validator.ValidateSave(t.Context(), controlPlan(control)))
	})

	t.Run(
		"rolling window outside one to twelve is refused while GL actuals are on",
		func(t *testing.T) {
			t.Parallel()

			for _, months := range []int16{0, 13} {
				control := benchmarkControl()
				control.Categories = nil
				control.GLActualsEnabled = true
				control.GLRollingMonths = months

				multiErr := refusal(t, validator.ValidateSave(t.Context(), controlPlan(control)))
				assert.Equal(t, []string{"control.glRollingMonths"}, refusedFields(multiErr))
				assert.Equal(
					t,
					"GL rolling months must be between 1 and 12",
					multiErr.Errors[0].Message,
				)
			}
		},
	)

	t.Run("rolling window is left alone while GL actuals are off", func(t *testing.T) {
		t.Parallel()

		for _, months := range []int16{0, 13} {
			control := benchmarkControl()
			control.Categories = nil
			control.GLActualsEnabled = false
			control.GLRollingMonths = months

			assert.NoError(t, validator.ValidateSave(t.Context(), controlPlan(control)))
		}
	})

	t.Run(
		"planned monthly miles below one are refused while GL actuals are on",
		func(t *testing.T) {
			t.Parallel()

			for _, miles := range []int64{0, -5} {
				control := benchmarkControl()
				control.Categories = nil
				control.GLActualsEnabled = true
				control.PlannedMonthlyMiles = &miles

				multiErr := refusal(t, validator.ValidateSave(t.Context(), controlPlan(control)))
				assert.Equal(t, []string{"control.plannedMonthlyMiles"}, refusedFields(multiErr))
				assert.Equal(
					t,
					"Planned monthly miles must be greater than 0",
					multiErr.Errors[0].Message,
				)
			}
		},
	)

	t.Run("planned monthly miles are optional while GL actuals are on", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.Categories = nil
		control.GLActualsEnabled = true
		control.PlannedMonthlyMiles = nil

		assert.NoError(t, validator.ValidateSave(t.Context(), controlPlan(control)))
	})

	t.Run("planned monthly miles are left alone while GL actuals are off", func(t *testing.T) {
		t.Parallel()

		for _, miles := range []int64{0, -5} {
			control := benchmarkControl()
			control.Categories = nil
			control.GLActualsEnabled = false
			control.PlannedMonthlyMiles = &miles

			assert.NoError(t, validator.ValidateSave(t.Context(), controlPlan(control)))
		}
	})

	t.Run("live fuel without an index is refused under control", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.Categories = nil
		control.UseLiveFuelPrice = true
		control.FuelIndexID = nil

		multiErr := refusal(t, validator.ValidateSave(t.Context(), controlPlan(control)))
		assert.Equal(t, []string{"control.fuelIndexId"}, refusedFields(multiErr))
	})

	t.Run("an unchanged control is not validated again", func(t *testing.T) {
		t.Parallel()

		control := benchmarkControl()
		control.Categories = nil
		control.MilesPerGallon = control.MilesPerGallon.Neg()

		plan := controlPlan(control)
		plan.ControlChanged = false
		assert.NoError(t, validator.ValidateSave(t.Context(), plan))
	})
}

func TestValidator_ValidateSave_Categories(t *testing.T) {
	t.Parallel()

	validator := costingservice.NewValidator()
	glAccountID := pulid.MustNew("gla_")

	glOn := func() *costingcontrol.CostingControl {
		control := benchmarkControl()
		control.Categories = nil
		control.GLActualsEnabled = true
		return control
	}

	t.Run("an override without a rate is refused at its place in the batch", func(t *testing.T) {
		t.Parallel()

		category := glActualCategory("ccat_override")
		category.RateSource = costingcontrol.RateSourceOverride

		plan := controlPlan(glOn())
		plan.ControlChanged = false
		plan.Categories = []*costingservice.PlannedCategory{
			{Category: glActualCategory("ccat_ok"), GLAccountIDs: []pulid.ID{glAccountID}},
			{Category: category},
		}

		multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
		assert.Equal(t, []string{"categories[1].overrideRatePerMile"}, refusedFields(multiErr))
	})

	t.Run("an active category on GL actuals with no account is refused", func(t *testing.T) {
		t.Parallel()

		plan := controlPlan(glOn())
		plan.ControlChanged = false
		plan.Categories = []*costingservice.PlannedCategory{
			{Category: glActualCategory("ccat_ok"), GLAccountIDs: []pulid.ID{glAccountID}},
			{Category: glActualCategory("ccat_none")},
		}

		multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
		assert.Equal(t, []string{"categories[1].glAccountIds"}, refusedFields(multiErr))
		assert.Equal(
			t,
			[]string{string(errortypes.ErrRequired)},
			validationtest.FieldCodes(multiErr, "categories[1].glAccountIds"),
		)
		assert.Equal(
			t,
			"Map at least one GL account to use GL actuals for this category",
			multiErr.Errors[0].Message,
		)
	})

	t.Run("GL accounts are not asked for while GL actuals are off", func(t *testing.T) {
		t.Parallel()

		control := glOn()
		control.GLActualsEnabled = false
		plan := controlPlan(control)
		plan.Categories = []*costingservice.PlannedCategory{
			{Category: glActualCategory("ccat_none")},
		}

		assert.NoError(t, validator.ValidateSave(t.Context(), plan))
	})

	t.Run("GL accounts are not asked for of an inactive category", func(t *testing.T) {
		t.Parallel()

		category := glActualCategory("ccat_none")
		category.IsActive = false
		plan := controlPlan(glOn())
		plan.Categories = []*costingservice.PlannedCategory{{Category: category}}

		assert.NoError(t, validator.ValidateSave(t.Context(), plan))
	})

	t.Run("GL accounts are not asked for of a category on another rate", func(t *testing.T) {
		t.Parallel()

		category := glActualCategory("ccat_none")
		category.RateSource = costingcontrol.RateSourceBenchmark
		plan := controlPlan(glOn())
		plan.Categories = []*costingservice.PlannedCategory{{Category: category}}

		assert.NoError(t, validator.ValidateSave(t.Context(), plan))
	})

	t.Run(
		"an unchanged category on GL actuals with no account is refused on the switch",
		func(t *testing.T) {
			t.Parallel()

			plan := controlPlan(glOn())
			unchanged := glActualCategory("ccat_untouched")
			unchanged.Name = "Tolls"
			mapped := glActualCategory("ccat_mapped")
			mapped.GLAccounts = []*costingcontrol.CostCategoryGLAccount{{GLAccountID: glAccountID}}
			plan.Unchanged = []*costingcontrol.CostCategory{unchanged, mapped}

			multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
			assert.Equal(t, []string{"control.glActualsEnabled"}, refusedFields(multiErr))
			assert.Equal(t, "Tolls reads its rate from GL actuals and has no GL account mapped",
				multiErr.Errors[0].Error())
		},
	)

	t.Run("an unknown category is refused at its id", func(t *testing.T) {
		t.Parallel()

		plan := controlPlan(glOn())
		plan.ControlChanged = false
		plan.Categories = []*costingservice.PlannedCategory{{}}

		multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
		assert.Equal(t, []string{"categories[0].id"}, refusedFields(multiErr))
		assert.Equal(
			t,
			[]string{string(errortypes.ErrNotFound)},
			validationtest.FieldCodes(multiErr, "categories[0].id"),
		)
	})

	t.Run("a category changed twice in one save is refused the second time", func(t *testing.T) {
		t.Parallel()

		plan := controlPlan(glOn())
		plan.ControlChanged = false
		plan.Categories = []*costingservice.PlannedCategory{
			{Category: glActualCategory("ccat_same"), GLAccountIDs: []pulid.ID{glAccountID}},
			{Category: glActualCategory("ccat_same"), GLAccountIDs: []pulid.ID{glAccountID}},
		}

		multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
		assert.Equal(t, []string{"categories[1].id"}, refusedFields(multiErr))
		assert.Equal(
			t,
			[]string{string(errortypes.ErrDuplicate)},
			validationtest.FieldCodes(multiErr, "categories[1].id"),
		)
	})

	t.Run("the control and its categories are refused together", func(t *testing.T) {
		t.Parallel()

		control := glOn()
		control.GLRollingMonths = 0
		plan := controlPlan(control)
		plan.Categories = []*costingservice.PlannedCategory{
			{Category: glActualCategory("ccat_none")},
		}

		multiErr := refusal(t, validator.ValidateSave(t.Context(), plan))
		assert.Equal(
			t,
			[]string{"control.glRollingMonths", "categories[0].glAccountIds"},
			refusedFields(multiErr),
		)
	})
}
