package deskbench_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const codeModeScenarios = "../../deskbench/scenarios/codemode.yaml"

func TestCodeModeScenariosArePaired(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{codeModeScenarios})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(scenarios), 16, "at least eight pairs")

	halves := map[string][]*deskbench.Scenario{}
	kinds := map[string]bool{}
	for _, scenario := range scenarios {
		require.NotEmpty(t, scenario.Pair, scenario.Name)
		halves[scenario.Pair] = append(halves[scenario.Pair], scenario)
		for _, tag := range scenario.Tags {
			kinds[tag] = true
		}
	}
	for _, kind := range []string{"fan-out", "reconciliation", "report", "control", "adversarial"} {
		assert.Truef(t, kinds[kind], "the pairs hold a %s task", kind)
	}

	for pair, found := range halves {
		require.Lenf(t, found, 2, "pair %s", pair)
		modes := []deskbench.ScriptMode{found[0].Mode, found[1].Mode}
		assert.ElementsMatchf(t,
			[]deskbench.ScriptMode{deskbench.ModeDiscrete, deskbench.ModeScript}, modes, pair)
		assert.Equalf(t, found[0].Split, found[1].Split, "pair %s shares one split", pair)

		assert.Equalf(t, found[0].Steps[0].Expect.Facts, found[1].Steps[0].Expect.Facts,
			"pair %s is graded on the same facts in both modes", pair)
		for _, half := range found {
			step := half.Steps[0].Expect
			if half.Mode == deskbench.ModeDiscrete || slices.Contains(half.Tags, "control") {
				assert.Containsf(t, step.NoCalls, "agent_run_script",
					"%s never runs a script", half.Name)
			}
		}
	}
}

func TestScriptModeForcesTheGateAndDiscreteShutsIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, codemode.OverrideOn, deskbench.ModeScript.Override())
	assert.Equal(t, codemode.OverrideOff, deskbench.ModeDiscrete.Override())
	assert.Equal(t, codemode.OverrideDefault, deskbench.ScriptMode("").Override())
}

func TestLoadScenarios_RefusesAPairWithoutBothHalves(t *testing.T) {
	t.Parallel()

	half := func(name, mode, say string) string {
		return strings.Join([]string{
			"  - name: " + name,
			"    agent: Dispatch desk",
			"    mode: " + mode,
			"    pair: late",
			"    steps:",
			"      - say: \"" + say + "\"",
		}, "\n") + "\n"
	}
	cases := map[string]string{
		"one half":       half("a", "discrete", "late loads?"),
		"two discrete":   half("a", "discrete", "late loads?") + half("b", "discrete", "late loads?"),
		"different text": half("a", "discrete", "late loads?") + half("b", "script", "on time?"),
	}
	for label, body := range cases {
		dir := t.TempDir()
		file := filepath.Join(dir, "pairs.yaml")
		require.NoError(t, os.WriteFile(file, []byte("scenarios:\n"+body), 0o600))
		_, err := deskbench.LoadScenarios([]string{file})
		require.Errorf(t, err, label)
		assert.Containsf(t, err.Error(), "pair", label)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "modeless.yaml")
	require.NoError(t, os.WriteFile(file, []byte("scenarios:\n"+strings.Join([]string{
		"  - name: a",
		"    agent: Dispatch desk",
		"    mode: sometimes",
		"    pair: late",
		"    steps:",
		"      - say: \"late loads?\"",
	}, "\n")+"\n"), 0o600))
	_, err := deskbench.LoadScenarios([]string{file})
	require.ErrorContains(t, err, "mode is discrete or script")
}
