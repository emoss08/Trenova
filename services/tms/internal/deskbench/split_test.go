package deskbench_test

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateSplits = flag.Bool("update", false, "rewrite deskbench/splits.yaml")

const (
	scenarioDir    = "../../deskbench/scenarios"
	splitLedger    = "../../deskbench/splits.yaml"
	splitsCommand  = "go test -count=1 -run TestScenarioSplits ./internal/deskbench/ -update"
	splitLedgerTop = "# The held-out scenarios, by desk and by chat or unattended. Generated from the rule in\n" +
		"# docs/engineering/desk-bench.md (Labels and splits) by running, in services/tms:\n" +
		"#   " + splitsCommand + "\n" +
		"# Do not edit by hand. A scenario that joins this list after living in the development\n" +
		"# split may have been tuned against; replace it with a new scenario rather than keep it.\n"
)

func TestScenarioSplits_MatchTheLedger(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{scenarioDir})
	require.NoError(t, err)
	ledger := deskbench.SplitLedger(scenarios)

	if *updateSplits {
		encoded, err := ledger.Marshal(splitLedgerTop)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(splitLedger, encoded, 0o600))
		return
	}

	raw, err := os.ReadFile(splitLedger)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s is missing; write it with: %s", splitLedger, splitsCommand)
	}
	require.NoError(t, err)
	recorded, err := evalsplit.ParseLedger(raw)
	require.NoError(t, err)

	diff := ledger.Diff(recorded)
	if !diff.Empty() {
		t.Fatalf("the held-out scenarios changed.\njoined: %s\nleft: %s\n"+
			"A scenario added to a desk moves the quota; one that joins after living in "+
			"development may have been tuned against. Review, then rewrite the ledger with: %s",
			strings.Join(diff.Joined, ", "), strings.Join(diff.Left, ", "), splitsCommand)
	}
}

func TestScenarioSplits_HoldOutEveryDeskWithTwoScenarios(t *testing.T) {
	t.Parallel()

	scenarios, err := deskbench.LoadScenarios([]string{scenarioDir})
	require.NoError(t, err)

	members := map[evalsplit.Stratum]int{}
	held := map[evalsplit.Stratum]int{}
	for _, scenario := range scenarios {
		if scenario.Split == evalsplit.Adversarial {
			continue
		}
		members[scenario.Stratum()]++
		if scenario.Split == evalsplit.HeldOut {
			held[scenario.Stratum()]++
		}
	}
	for stratum, count := range members {
		assert.Equalf(t, evalsplit.HeldOutQuota(count), held[stratum],
			"%s holds out %d of %d", stratum, held[stratum], count)
	}
}

func TestLoadScenarios_AssignsTheSameSplitWhenOneFileIsLoaded(t *testing.T) {
	t.Parallel()

	all, err := deskbench.LoadScenarios([]string{scenarioDir})
	require.NoError(t, err)
	want := make(map[string]evalsplit.Split, len(all))
	for _, scenario := range all {
		want[scenario.Name] = scenario.Split
	}

	one, err := deskbench.LoadScenarios([]string{filepath.Join(scenarioDir, "unattended-monitor.yaml")})
	require.NoError(t, err)
	require.NotEmpty(t, one)
	for _, scenario := range one {
		assert.Equal(t, want[scenario.Name], scenario.Split, scenario.Name)
	}
}

func TestLoadScenarios_RefusesAHandWrittenHeldOutSplit(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "hand.yaml")
	require.NoError(t, os.WriteFile(file, []byte(
		"scenarios:\n  - name: hand\n    split: held-out\n    steps:\n      - say: hi\n"), 0o600))

	_, err := deskbench.LoadScenarios([]string{file})
	require.ErrorContains(t, err, "assigned by rule")
}
