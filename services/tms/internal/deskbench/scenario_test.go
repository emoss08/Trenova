package deskbench_test

import (
	"os"
	"path/filepath"
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

func TestLoadScenarios_TheEDIScenariosAskTheBenchEDIDeskAndCheckTheToolsTheyExistFor(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{"../../deskbench/scenarios/edi.yaml"})
	require.NoError(t, err)
	require.NotEmpty(t, scenarios)

	called := map[string]bool{}
	proposed := map[string]bool{}
	for _, scenario := range scenarios {
		assert.Equal(t, deskbench.BenchAgentPrefix+"EDI desk", scenario.Agent, scenario.Name)
		require.NoError(t, scenario.Validate(), scenario.Name)
		for _, step := range scenario.Steps {
			for _, call := range step.Expect.Calls {
				called[call] = true
			}
			for _, proposal := range step.Expect.Proposals {
				proposed[proposal.Tool] = true
			}
			for _, attachment := range step.Attach {
				assert.FileExists(t, filepath.Join("../../deskbench/scenarios", attachment), scenario.Name)
			}
		}
	}
	assert.True(t, proposed["edi_apply_connection_draft"], "no EDI scenario saves a drafted connection")
	for _, tool := range []string{
		"edi_get_ack_health", "edi_list_messages", "edi_inspect_x12", "edi_preview_x12",
		"edi_draft_connection",
	} {
		assert.Truef(t, called[tool], "no EDI scenario expects %s", tool)
	}

	agents, err := deskbench.LoadAgentSpecs("../../deskbench/agents.yaml")
	require.NoError(t, err)
	found := false
	for _, spec := range agents {
		if spec.Name == deskbench.BenchAgentPrefix+"EDI desk" {
			found = true
			assert.Subset(t, spec.Tools, []string{
				"edi_get_ack_health", "edi_list_messages", "edi_inspect_x12", "edi_preview_x12",
				"edi_draft_connection", "edi_apply_connection_draft",
			})
		}
	}
	assert.True(t, found, "the EDI scenarios ask an agent agents.yaml does not keep")
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

func TestLoadAgentSpecs_ReadsDataAccessAndRefusesAnUnknownOne(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	good := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(good, []byte(
		"agents:\n  - name: EDI desk\n    template: EDIDesk\n    dataAccess: Restricted\n"), 0o600))
	specs, err := deskbench.LoadAgentSpecs(good)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "Restricted", specs[0].DataAccess)

	bad := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte(
		"agents:\n  - name: EDI desk\n    dataAccess: Confidential\n"), 0o600))
	_, err = deskbench.LoadAgentSpecs(bad)
	require.ErrorIs(t, err, deskbench.ErrAgentSpecDataAccess)
}

func TestShippedScenarios_LoadWithLabelsAndASplit(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{"../../deskbench/scenarios"})
	require.NoError(t, err)
	require.NotEmpty(t, scenarios)

	for _, scenario := range scenarios {
		assert.True(t, scenario.Split.IsValid(), scenario.Name)
		assert.NotNil(t, scenario.ExpectedTools, scenario.Name)
		assert.NotNil(t, scenario.ForbiddenTools, scenario.Name)
	}
}
