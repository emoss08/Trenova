package tariff

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var gridNeed = rules.TariffNeed{Table: "Grid", Keys: []string{"A"}}

type loadingContext struct {
	tables      []rules.TariffTable
	stamp       string
	cellLoads   [][]rules.TariffNeed
	tableLoads  [][]string
	tenants     []pagination.TenantInfo
	loadedCells []rules.TariffCell
}

func (l *loadingContext) Context(
	context.Context,
	*tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	return &rules.TariffContext{Tables: append([]rules.TariffTable(nil), l.tables...)}, nil
}

func (l *loadingContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return l.stamp, nil
}

func (l *loadingContext) LoadCells(
	_ context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	needs []rules.TariffNeed,
) error {
	l.cellLoads = append(l.cellLoads, needs)
	l.tenants = append(l.tenants, tenant)
	for i := range tariffCtx.Tables {
		for _, need := range needs {
			if tariffCtx.Tables[i].Code == need.Table {
				tariffCtx.Tables[i].Cells = append(tariffCtx.Tables[i].Cells, l.loadedCells...)
				tariffCtx.Tables[i].Loaded = append(tariffCtx.Tables[i].Loaded, need.Keys)
			}
		}
	}
	return nil
}

func (l *loadingContext) LoadTables(
	_ context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	tables []string,
) error {
	l.tableLoads = append(l.tableLoads, tables)
	l.tenants = append(l.tenants, tenant)
	for i := range tariffCtx.Tables {
		for _, code := range tables {
			if tariffCtx.Tables[i].Code == code {
				tariffCtx.Tables[i].Cells = append([]rules.TariffCell{}, l.loadedCells...)
				tariffCtx.Tables[i].Loaded = nil
			}
		}
	}
	return nil
}

type needyModule struct {
	fakeModule
	runs          []*rules.TariffTestRun
	scaffolds     []*rules.TariffScaffold
	testCalls     int
	scaffoldCalls int
}

func (n *needyModule) TestTariff(
	_ context.Context,
	req *rules.TariffTestRequest,
) (*rules.TariffTestRun, error) {
	n.tests = append(n.tests, req)
	run := n.runs[min(n.testCalls, len(n.runs)-1)]
	n.testCalls++
	return run, nil
}

func (n *needyModule) ScaffoldTariffTest(
	context.Context,
	*rules.TariffScaffoldRequest,
) (*rules.TariffScaffold, error) {
	scaffold := n.scaffolds[min(n.scaffoldCalls, len(n.scaffolds)-1)]
	n.scaffoldCalls++
	return scaffold, nil
}

func needsRun() *rules.TariffTestRun {
	return &rules.TariffTestRun{
		Tests: []rules.TariffTestResult{{Name: "to A", Status: rules.TariffTestNeeds}},
		Needs: []rules.TariffNeed{gridNeed},
	}
}

func passedRun() *rules.TariffTestRun {
	return &rules.TariffTestRun{
		Tests: []rules.TariffTestResult{{Name: "to A", Status: rules.TariffTestPassed}},
	}
}

func TestTestChecked_LoadsTheCellsItsTestsNeedAndRunsThemAgain(t *testing.T) {
	t.Parallel()

	module := &needyModule{runs: []*rules.TariffTestRun{needsRun(), passedRun()}}
	contexts := &loadingContext{}
	checker := &Checker{module: module, context: contexts}
	tariffCtx := &rules.TariffContext{Tables: []rules.TariffTable{{Code: "Grid"}}}

	run, err := checker.TestChecked(t.Context(), &SourceCheck{
		TenantInfo: testTenant,
		Check:      &rules.TariffCheck{OK: true},
		Context:    tariffCtx,
	}, "tariff \"x\"\n")
	require.NoError(t, err)

	assert.True(t, run.Passed())
	assert.Equal(t, 2, module.testCalls)
	assert.Equal(t, [][]rules.TariffNeed{{gridNeed}}, contexts.cellLoads)
	assert.Equal(t, []pagination.TenantInfo{testTenant}, contexts.tenants)
	assert.Same(t, tariffCtx, module.tests[1].Context, "the second run carries the loaded cells")
	assert.Equal(t, [][]string{{"A"}}, tariffCtx.Tables[0].Loaded)
}

func TestRunTests_LoadsWholeTablesAfterThreeCellPassesAndThenGivesUp(t *testing.T) {
	t.Parallel()

	module := &needyModule{runs: []*rules.TariffTestRun{needsRun()}}
	contexts := &loadingContext{}
	checker := &Checker{module: module, context: contexts}

	_, err := checker.RunTests(t.Context(), &TestRequest{
		TenantInfo: testTenant,
		Source:     "tariff \"x\"\n",
		Context:    &rules.TariffContext{Tables: []rules.TariffTable{{Code: "Grid"}}},
	})
	require.ErrorIs(t, err, ErrUnresolvedNeeds)
	assert.Equal(t, cellPasses+1, module.testCalls)
	assert.Len(t, contexts.cellLoads, cellPasses-1)
	assert.Equal(t, [][]string{{"Grid"}}, contexts.tableLoads)
}

func TestScaffoldTest_LoadsTheCellsTheRatingNeeds(t *testing.T) {
	t.Parallel()

	block := "test \"t\" {\n}\n"
	module := &needyModule{scaffolds: []*rules.TariffScaffold{
		{Status: rules.TariffNeeds, Needs: []rules.TariffNeed{gridNeed}},
		{Test: &block},
	}}
	contexts := &loadingContext{}
	checker := &Checker{module: module, context: contexts}

	scaffold, err := checker.ScaffoldTest(t.Context(), &ScaffoldRequest{
		TenantInfo: testTenant,
		Source:     "tariff \"x\"\n",
		Context:    &rules.TariffContext{Tables: []rules.TariffTable{{Code: "Grid"}}},
		Name:       "t",
	})
	require.NoError(t, err)
	require.NotNil(t, scaffold.Test)
	assert.Equal(t, block, *scaffold.Test)
	assert.Equal(t, [][]rules.TariffNeed{{gridNeed}}, contexts.cellLoads)
}

func TestTableCells_AnswersTheNeededTablesAtTheStampTheyWereReadAt(t *testing.T) {
	t.Parallel()

	cell := rules.TariffCell{
		At:    []rules.TariffCellPosition{{Key: "A"}},
		Value: decimal.RequireFromString("20"),
	}
	contexts := &loadingContext{
		stamp:       "3:7:1700000000",
		tables:      []rules.TariffTable{{Code: "Grid", Loaded: [][]string{}}, {Code: "Other"}},
		loadedCells: []rules.TariffCell{cell},
	}
	checker := &Checker{module: &fakeModule{}, context: contexts}

	cells, err := checker.TableCells(t.Context(), &TableCellsRequest{
		TenantInfo: testTenant,
		Needs:      []rules.TariffNeed{gridNeed, {Table: "Missing", Keys: []string{"Z"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "3:7:1700000000", cells.Stamp)
	require.Len(t, cells.Tables, 1, "a table the organisation does not have is left out")
	assert.Equal(t, "Grid", cells.Tables[0].Code)
	assert.Equal(t, []rules.TariffCell{cell}, cells.Tables[0].Cells)
	assert.Equal(t, [][]string{{"A"}}, cells.Tables[0].Loaded)
	assert.Empty(t, contexts.tableLoads)

	whole, err := checker.TableCells(t.Context(), &TableCellsRequest{
		TenantInfo: testTenant,
		Needs:      []rules.TariffNeed{gridNeed},
		Whole:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"Grid"}}, contexts.tableLoads)
	require.Len(t, whole.Tables, 1)
	assert.Nil(t, whole.Tables[0].Loaded, "a whole table carries no loaded combinations")
	assert.Equal(t, []rules.TariffCell{cell}, whole.Tables[0].Cells)
}
