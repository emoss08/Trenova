package agentevalgate_test

import (
	"os"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/require"
)

const (
	selectionSuite  = "evals/toolselection.yaml"
	selectionFloors = "evals/toolselection.floors.json"
	minSelection    = 40
)

func TestToolSelectionAgainstFloors(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suite, err := agentevalgate.LoadSelectionSuite(selectionSuite)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(suite.Cases), minSelection,
		"the selection suite keeps at least %d requests", minSelection)
	require.Empty(t, kit.UnknownTools(suite),
		"every expected tool must be registered; a renamed tool renames its cases")

	report := kit.EvaluateSelection(suite)
	t.Logf("recall@5 %.2f, top-1 %.2f over %d requests", report.RecallAt5, report.Top1,
		len(report.Outcomes))
	if misses := report.Misses(); misses != "" {
		t.Logf("requests whose first tool was not an expected one:\n%s", misses)
	}

	if *update {
		encoded, encodeErr := agentevalgate.MarshalJSON(report.Floors())
		require.NoError(t, encodeErr)
		require.NoError(t, agentevalgate.WriteFile(selectionFloors, encoded))
		return
	}

	raw, err := os.ReadFile(selectionFloors)
	require.NoError(t, err, "create the floors with -update")
	var floors agentevalgate.SelectionFloors
	require.NoError(t, sonic.Unmarshal(raw, &floors))

	require.GreaterOrEqualf(t, report.RecallAt5, floors.RecallAt5,
		"recall@5 fell below its floor: a tool description no longer lets find_tools "+
			"reach the tool a request needs\n%s", report.Misses())
	require.GreaterOrEqualf(t, report.Top1, floors.Top1,
		"top-1 fell below its floor: find_tools now ranks another tool first\n%s",
		report.Misses())
}
