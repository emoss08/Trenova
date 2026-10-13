package deskbench_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paletteScenario() *deskbench.Scenario {
	return &deskbench.Scenario{
		Name:    "palette-case",
		Palette: true,
		Steps:   []deskbench.Step{{Say: "Tell me about SEED-PAY-002?"}},
	}
}

func TestValidate_APaletteScenarioLeavesTheChoiceOfAgentToThePalette(t *testing.T) {
	t.Parallel()

	require.NoError(t, paletteScenario().Validate())

	withAgent := paletteScenario()
	withAgent.Agent = "Dispatch desk"
	require.ErrorContains(t, withAgent.Validate(), "the palette picks who answers")

	handedOver := paletteScenario()
	handedOver.Steps[0].Agent = "Dispatch desk"
	require.ErrorContains(t, handedOver.Validate(), "asked from the palette")

	onDesk := paletteScenario()
	onDesk.Steps[0].Surface = agent.SurfaceDesk
	require.ErrorContains(t, onDesk.Validate(), "asked from the palette")

	studio := paletteScenario()
	studio.Formula = &deskbench.FormulaPage{}
	require.ErrorContains(t, studio.Validate(), "not both")
}

func TestLoadScenarios_ThePaletteScenariosAskFromThePalette(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{"../../deskbench/scenarios/palette.yaml"})
	require.NoError(t, err)
	require.NotEmpty(t, scenarios)
	for _, scenario := range scenarios {
		assert.True(t, scenario.Palette, scenario.Name)
		assert.Empty(t, scenario.Agent, scenario.Name)
	}
}

func TestValidate_AFoundMentionNamesItsTypeAndFindsItsIDWithOneSelect(t *testing.T) {
	t.Parallel()

	withMention := func(mention map[string]any) *deskbench.Scenario {
		scenario := paletteScenario()
		scenario.Steps[0].Mentions = []map[string]any{mention}

		return scenario
	}

	require.NoError(t, withMention(map[string]any{
		"type": "shipment",
		"find": "select id from shipments where pro_number = 'SEED-PAY-002'",
	}).Validate())
	require.ErrorContains(t, withMention(map[string]any{
		"type": "shipment",
		"id":   "shp_1",
		"find": "select id from shipments",
	}).Validate(), "an id or a find query, not both")
	require.ErrorContains(t, withMention(map[string]any{
		"find": "select id from shipments",
	}).Validate(), "still names its type")
	require.ErrorContains(t, withMention(map[string]any{
		"type": "shipment",
		"find": "delete from shipments",
	}).Validate(), "a fact is a SELECT")
}
