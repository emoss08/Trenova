package agentquerytoolservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresentAmounts_ShowsAmountsInMinorUnitsAsDecimals(t *testing.T) {
	t.Parallel()

	row := compact(map[string]any{
		"id":           "dstl_1",
		"netPayMinor":  float64(230000),
		"currencyCode": "USD",
		"lines": []any{
			map[string]any{"id": "dstll_1", "amountMinor": float64(-1250)},
		},
	})
	assert.InDelta(t, 230000, row["netPayMinor"], 0, "compact alone leaves minor units alone")

	presentAmounts(row)

	assert.Equal(t, "2300.00", row["netPay"])
	assert.NotContains(t, row, "netPayMinor")
	line := row["lines"].([]any)[0].(map[string]any)
	assert.Equal(t, "-12.50", line["amount"])
}

func TestPresentAmounts_KeepsTheDecimalARecordAlreadyCarries(t *testing.T) {
	t.Parallel()

	row := map[string]any{"disputedAmount": "45.10", "disputedAmountMinor": float64(4510)}
	presentAmounts(row)

	assert.Equal(t, map[string]any{"disputedAmount": "45.10"}, row)
}

func TestPresentedPaths_NameAWithheldAmountAsTheResultShowsIt(t *testing.T) {
	t.Parallel()

	got := presentedPaths([]string{"netPayMinor", "lines.amountMinor", "notes", "netPayMinor"})

	assert.Equal(t, []string{"netPay", "lines.amount", "notes"}, got)
}

func TestSelectedColumns_NameAnAmountAsTheResultShowsIt(t *testing.T) {
	t.Parallel()

	fields := []graphQLField{
		{Name: "id", Selected: true},
		{Name: "netPayMinor", Selected: true},
		{Name: "grossEarningsMinor", Selected: true},
	}

	got := selectedColumns(fields, []map[string]any{{"id": "x", "netPay": "1.00"}})

	assert.Equal(t, []string{"id", "netPay"}, got)
}

func TestAmountParams_TakeADecimalAndFilterTheMinorUnits(t *testing.T) {
	t.Parallel()

	params := amountParams(paramsFor(&graphQLField{
		Name: "netPayMinor", Type: graphQLTypeInt, Kind: graphQLKindScalar,
	}))
	require.Len(t, params, 2)
	assert.Equal(t, "netPayMin", params[0].Name)
	assert.Equal(t, "netPayMax", params[1].Name)
	assert.Equal(t, "netPayMinor", params[0].Field)

	filters, err := compileFilters(params, map[string]any{
		"netPayMin": "1250.50",
		"netPayMax": float64(2000),
	}, filtercatalog.NewClock("UTC"))
	require.NoError(t, err)
	assert.ElementsMatch(t, []map[string]any{
		{paramField: "netPayMinor", filterOperator: string(dbtype.OpGreaterThanOrEqual), filterValue: int64(125050)},
		{paramField: "netPayMinor", filterOperator: string(dbtype.OpLessThanOrEqual), filterValue: int64(200000)},
	}, filters)

	_, err = compileFilters(params, map[string]any{"netPayMin": "a lot"}, filtercatalog.NewClock("UTC"))
	require.Error(t, err)
}

func TestPresentMinor_FormatsTheNamedUnmarkedAmounts(t *testing.T) {
	t.Parallel()

	record := map[string]any{"outstandingAdvances": float64(20000), "accruedEventCount": float64(3)}
	presentMinor(record, []string{"outstandingAdvances", "escrowBalance"})

	assert.Equal(t, "200.00", record["outstandingAdvances"])
	assert.InDelta(t, 3, record["accruedEventCount"], 0)
	assert.NotContains(t, record, "escrowBalance")
}

func TestOutsideRecords_MarkOnlyRecordsThatShowTheText(t *testing.T) {
	t.Parallel()

	got := outsideRecords(permission.ResourceDriverExpense, []string{"description"},
		map[string]any{"id": "dexp_1", "description": "Lumper at the dock"},
		map[string]any{"id": "dexp_2"},
		map[string]any{"id": "dexp_3", "description": "  "},
	)

	assert.Equal(t, []agent.RecordRef{{EntityType: "driver_expense", ID: "dexp_1"}}, got)
	assert.Empty(t, outsideRecords(permission.ResourceDriverExpense, nil,
		map[string]any{"id": "dexp_1", "description": "x"}))
}

func TestWithBeside_KeepsThePagesMarks(t *testing.T) {
	t.Parallel()

	marks := []agent.RecordRef{{EntityType: "driver_expense", ID: "dexp_1"}}
	combined, err := withBeside(&graphQLListOutcome{
		Count:   1,
		Items:   []map[string]any{{"id": "dexp_1", "description": "Lumper at the dock"}},
		outside: marks,
	}, map[string]any{"tolerance": "5.00"})
	require.NoError(t, err)

	carrier, ok := combined.(agent.TaintCarrier)
	require.True(t, ok, "records read alongside a page keep the page's marks")
	assert.Equal(t, marks, carrier.TaintedRecords())
	encoded, err := sonic.Marshal(combined)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"tolerance":"5.00"`)

	plain, err := withBeside(&graphQLListOutcome{Count: 0}, map[string]any{"tolerance": "5.00"})
	require.NoError(t, err)
	_, marked := plain.(agent.TaintCarrier)
	assert.False(t, marked)
}

func TestDriverEarnings_AreReadByTheWorkersID(t *testing.T) {
	t.Parallel()

	earnings, statement := graphQLReads[toolGetDriverEarnings], graphQLReads[toolGetDriverSettlement]
	assert.Equal(t, "workerId", earnings.getKey())
	assert.Equal(t, "id", statement.getKey())
}
