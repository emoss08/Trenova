package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompact_ShowsAmountsInMinorUnitsAsDecimals(t *testing.T) {
	t.Parallel()

	row := compact(map[string]any{
		"id":           "dstl_1",
		"netPayMinor":  float64(230000),
		"currencyCode": "USD",
		"lines": []any{
			map[string]any{"id": "dstll_1", "amountMinor": float64(-1250)},
		},
	})

	assert.Equal(t, "2300.00", row["netPay"])
	assert.NotContains(t, row, "netPayMinor")
	line := row["lines"].([]any)[0].(map[string]any)
	assert.Equal(t, "-12.50", line["amount"])
}

func TestCompact_KeepsTheDecimalARecordAlreadyCarries(t *testing.T) {
	t.Parallel()

	row := compact(map[string]any{
		"disputedAmount":      "45.10",
		"disputedAmountMinor": float64(4510),
	})

	assert.Equal(t, map[string]any{"disputedAmount": "45.10"}, row)
}

func TestRelativePaths_NameAWithheldAmountAsTheResultShowsIt(t *testing.T) {
	t.Parallel()

	got := relativePaths([]string{
		"driverSettlement.netPayMinor",
		"driverSettlement.lines.amountMinor",
		"driverSettlement.notes",
	}, "driverSettlement.")

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

	params := paramsFor(&graphQLField{Name: "netPayMinor", Type: graphQLTypeInt, Kind: "scalar"})
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

func TestKeyVariable_IsTheOnlyRequiredIDOfAKeyedGet(t *testing.T) {
	t.Parallel()

	keyed := graphQLRead{Variables: []graphQLVariable{{Name: "workerId", Type: "ID", NonNull: true}}}
	byID := graphQLRead{Variables: []graphQLVariable{{Name: "id", Type: "ID", NonNull: true}}}

	assert.Equal(t, "workerId", keyed.keyVariable())
	assert.Equal(t, "id", byID.keyVariable())
	earnings, statement := graphQLReads[toolGetDriverEarnings], graphQLReads[toolGetDriverSettlement]
	assert.Equal(t, "workerId", earnings.keyVariable())
	assert.Equal(t, "id", statement.keyVariable())
}
