package tariff

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const laneProperty = `tariff "Lane"

let rate = tables.Lanes[lane.origin.state ?? "XX"] ?? fail "No lane rate"
charge "Linehaul" = rate

test "Texas bills its lane" {
  given lane.origin.state = "TX"
  expect total == $900
}

property "every lane bills something" {
  vary lane.origin.state in ["TX", "OK"]
  expect total > $0
}
`

func TestRunTests_LoadsTheCellsAPropertysCasesNeed(t *testing.T) {
	t.Parallel()

	contexts := &loadingContext{loadedCells: []rules.TariffCell{
		{At: []rules.TariffCellPosition{{Key: "TX"}}, Value: decimal.NewFromInt(900)},
		{At: []rules.TariffCellPosition{{Key: "OK"}}, Value: decimal.NewFromInt(700)},
	}}
	checker := &Checker{module: defaultModule{}, context: contexts}
	tariffCtx := &rules.TariffContext{Tables: []rules.TariffTable{{
		Code:      "Lanes",
		ValueType: "Money",
		Axes:      []rules.TariffAxis{{Match: "Exact", Kind: "State"}},
		Cells:     []rules.TariffCell{},
		Loaded:    [][]string{},
	}}}

	run, err := checker.RunTests(t.Context(), &TestRequest{
		TenantInfo: testTenant,
		Source:     laneProperty,
		Context:    tariffCtx,
	})
	require.NoError(t, err)

	require.Len(t, run.Tests, 2)
	assert.True(t, run.Passed(), "%+v", run.Tests)
	assert.True(t, run.Tests[1].IsProperty())
	assert.Equal(t, 2, run.Tests[1].Cases)
	require.NotEmpty(t, contexts.cellLoads)
	loaded := map[string]bool{}
	for _, needs := range contexts.cellLoads {
		for _, need := range needs {
			loaded[need.Keys[0]] = true
		}
	}
	assert.True(t, loaded["OK"], "the property's own case asked for its cells: %v", contexts.cellLoads)
	require.NotNil(t, run.Coverage, "coverage is measured once no test needs cells")
}
