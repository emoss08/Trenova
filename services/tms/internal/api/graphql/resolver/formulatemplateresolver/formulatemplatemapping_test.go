package formulatemplateresolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormulaInputDeclarationsToModel_CarriesEachConstraint(t *testing.T) {
	t.Parallel()

	check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source: "tariff \"Bounded\"\n\n" +
			"input rate: Money/mi where $0.50/mi..=$15/mi\n" +
			"input stops: Int = 2 where 1..\n" +
			"input service: Text = \"Standard\" where [\"Standard\", \"Guaranteed\"]\n" +
			"input flat: Money = $0\n\n" +
			"charge \"Linehaul\" (quantity: shipment.distance, rate: rate)\n" +
			"charge \"Stops\" = flat * stops\n" +
			"if service == \"Guaranteed\" {\n  charge \"Guarantee\" = $50\n}\n",
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.True(t, check.OK, "%+v", check.Diagnostics)

	models, err := formulaInputDeclarationsToModel(tarifftypes.InputDeclsFrom(check.Inputs))
	require.NoError(t, err)
	require.Len(t, models, 4)

	assert.Equal(t, map[string]any{"range": map[string]any{
		"min":          map[string]any{"$q": "0.5", "u": "$/mi"},
		"max":          map[string]any{"$q": "15", "u": "$/mi"},
		"maxInclusive": true,
	}}, models[0].Constraint)
	assert.Equal(t, map[string]any{"range": map[string]any{
		"min":          map[string]any{"$n": "1"},
		"max":          nil,
		"maxInclusive": false,
	}}, models[1].Constraint)
	assert.Equal(t, map[string]any{"oneOf": []any{"Standard", "Guaranteed"}}, models[2].Constraint)
	assert.Nil(t, models[3].Constraint)
}
