package costingresolver

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func controlInput() *gqlmodel.CostingControlInput {
	return &gqlmodel.CostingControlInput{
		MilesPerGallon:  "6.5",
		GlRollingMonths: 3,
		Version:         2,
	}
}

func refusedField(t *testing.T, err error) string {
	t.Helper()

	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	return fieldErr.Field()
}

func TestCostingSaveRequestFromInput(t *testing.T) {
	t.Parallel()

	t.Run("maps the control and each category change", func(t *testing.T) {
		t.Parallel()

		categoryID := pulid.MustNew("ccat_")
		glAccountID := pulid.MustNew("gla_")
		rate := "0.75"

		req, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{
			Control: controlInput(),
			Categories: []*gqlmodel.CostCategoryUpdateInput{{
				ID:                  categoryID.String(),
				RateSource:          gqlmodel.CostRateSourceOverride,
				OverrideRatePerMile: &rate,
				IsActive:            true,
				GlAccountIds:        []string{glAccountID.String()},
				Version:             4,
			}},
		})
		require.NoError(t, err)

		require.NotNil(t, req.Control)
		assert.Equal(t, "6.5", req.Control.MilesPerGallon.String())
		assert.Equal(t, int16(3), req.Control.GLRollingMonths)
		assert.Equal(t, int64(2), req.Control.Version)
		require.Len(t, req.Categories, 1)
		assert.Equal(t, categoryID, req.Categories[0].ID)
		assert.Equal(t, costingcontrol.RateSourceOverride, req.Categories[0].RateSource)
		assert.Equal(t, "0.75", req.Categories[0].OverrideRatePerMile.Decimal.String())
		assert.True(t, req.Categories[0].IsActive)
		assert.Equal(t, []pulid.ID{glAccountID}, req.Categories[0].GLAccountIDs)
		assert.Equal(t, int64(4), req.Categories[0].Version)
	})

	t.Run("a null control leaves the control out", func(t *testing.T) {
		t.Parallel()

		req, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{})
		require.NoError(t, err)
		assert.Nil(t, req.Control)
		assert.Empty(t, req.Categories)
	})

	t.Run("a rolling window outside one to twelve is left to the save rule", func(t *testing.T) {
		t.Parallel()

		input := controlInput()
		input.GlRollingMonths = 0
		req, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{Control: input})
		require.NoError(t, err)
		assert.Equal(t, int16(0), req.Control.GLRollingMonths)
	})

	t.Run("a rolling window past a small integer is refused under control", func(t *testing.T) {
		t.Parallel()

		input := controlInput()
		input.GlRollingMonths = math.MaxInt16 + 4
		_, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{Control: input})
		assert.Equal(t, "control.glRollingMonths", refusedField(t, err))

		var fieldErr *errortypes.Error
		require.ErrorAs(t, err, &fieldErr)
		assert.Equal(t, "Must be between {0} and {1}", fieldErr.Message)
		assert.Equal(t, []any{int64(math.MinInt16), int64(math.MaxInt16)}, fieldErr.Args)
	})

	t.Run("a bad control value is refused under control", func(t *testing.T) {
		t.Parallel()

		input := controlInput()
		input.MilesPerGallon = "lots"
		_, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{Control: input})
		assert.Equal(t, "control.milesPerGallon", refusedField(t, err))
	})

	t.Run("a bad category value is refused at its place in the save", func(t *testing.T) {
		t.Parallel()

		rate := "cheap"
		_, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{
			Categories: []*gqlmodel.CostCategoryUpdateInput{
				{ID: pulid.MustNew("ccat_").String(), RateSource: gqlmodel.CostRateSourceBenchmark},
				{
					ID:                  pulid.MustNew("ccat_").String(),
					RateSource:          gqlmodel.CostRateSourceOverride,
					OverrideRatePerMile: &rate,
				},
			},
		})
		assert.Equal(t, "categories[1].overrideRatePerMile", refusedField(t, err))
	})

	t.Run("a bad GL account is refused at its place in the category", func(t *testing.T) {
		t.Parallel()

		_, err := costingSaveRequestFromInput(&gqlmodel.CostingControlSaveInput{
			Categories: []*gqlmodel.CostCategoryUpdateInput{{
				ID:           pulid.MustNew("ccat_").String(),
				RateSource:   gqlmodel.CostRateSourceGLActual,
				GlAccountIds: []string{"not an id"},
			}},
		})
		assert.Equal(t, "categories[0].glAccountIds[0]", refusedField(t, err))
	})
}
