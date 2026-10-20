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
	for _, kind := range []string{
		"fan-out", "reconciliation", "report", "control", "adversarial", "batch-write",
	} {
		assert.Truef(t, kinds[kind], "the pairs hold a %s task", kind)
	}

	for pair, found := range halves {
		require.Lenf(t, found, 2, "pair %s", pair)
		modes := []deskbench.ScriptMode{found[0].Mode, found[1].Mode}
		script := deskbench.ModeScript
		if slices.Contains(found[0].Tags, "batch-write") {
			script = deskbench.ModeScriptWrites
		}
		assert.ElementsMatchf(t,
			[]deskbench.ScriptMode{deskbench.ModeDiscrete, script}, modes,
			"pair %s: a batch-write pair's script half may write, every other reads only", pair)
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

// The batch-write pairs are graded the same in both modes on what the approval
// left in the database, so a script's batch and the discrete proposals face
// the same checks, and both halves name the same changes.
func TestBatchWritePairsAreGradedOnTheSameApproval(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{codeModeScenarios})
	require.NoError(t, err)

	halves := map[string][]*deskbench.Scenario{}
	for _, scenario := range scenarios {
		if slices.Contains(scenario.Tags, "batch-write") {
			halves[scenario.Pair] = append(halves[scenario.Pair], scenario)
		}
	}
	require.Len(t, halves, 7, "five batch writes and two stale approvals")
	for pair, found := range halves {
		require.Lenf(t, found, 2, "pair %s", pair)
		require.Equalf(t, len(found[0].Steps), len(found[1].Steps), pair)
		approved := false
		for idx := range found[0].Steps {
			one, other := found[0].Steps[idx], found[1].Steps[idx]
			assert.Equalf(t, one.Decide, other.Decide, "pair %s step %d decides alike", pair, idx+1)
			assert.Equalf(t, one.Setup, other.Setup, "pair %s step %d sets up alike", pair, idx+1)
			assert.Equalf(t, one.Expect.DB, other.Expect.DB, "pair %s step %d checks alike", pair, idx+1)
			approved = approved || one.Decide != ""
		}
		assert.Truef(t, approved, "pair %s is graded after a person decides", pair)
	}
}

func TestScriptModeForcesTheGateAndDiscreteShutsIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, codemode.OverrideWrites, deskbench.ModeScriptWrites.Override())
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
	require.ErrorContains(t, err, "mode is discrete, script or script_writes")
}
