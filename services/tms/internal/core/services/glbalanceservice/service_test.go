package glbalanceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/glbalance"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIncomeStatementAggregatesCategories(t *testing.T) {
	t.Parallel()

	periodID := pulid.MustNew("fp_")
	svc := &Service{repo: fakeGLBalanceRepo{balances: []*glbalance.PeriodAccountBalance{
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "4000", AccountName: "Revenue", AccountCategory: accounttype.CategoryRevenue, PeriodCreditMinor: 10000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "5000", AccountName: "Cost", AccountCategory: accounttype.CategoryCostOfRevenue, PeriodDebitMinor: 4000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "6000", AccountName: "Expense", AccountCategory: accounttype.CategoryExpense, PeriodDebitMinor: 2500},
	}}}

	statement, err := svc.GetIncomeStatement(t.Context(), pagination.TenantInfo{}, periodID)
	require.NoError(t, err)
	require.NotNil(t, statement)
	assert.Equal(t, int64(10000), statement.Revenue.TotalMinor)
	assert.Equal(t, int64(4000), statement.CostOfRevenue.TotalMinor)
	assert.Equal(t, int64(2500), statement.OperatingExpense.TotalMinor)
	assert.Equal(t, int64(6000), statement.GrossProfitMinor)
	assert.Equal(t, int64(3500), statement.NetIncomeMinor)
}

func TestGetBalanceSheetIncludesCurrentPeriodIncome(t *testing.T) {
	t.Parallel()

	periodID := pulid.MustNew("fp_")
	svc := &Service{repo: fakeGLBalanceRepo{balances: []*glbalance.PeriodAccountBalance{
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "1110", AccountName: "AR", AccountCategory: accounttype.CategoryAsset, PeriodDebitMinor: 10000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "2100", AccountName: "AP", AccountCategory: accounttype.CategoryLiability, PeriodCreditMinor: 3000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "3000", AccountName: "Equity", AccountCategory: accounttype.CategoryEquity, PeriodCreditMinor: 2000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "4000", AccountName: "Revenue", AccountCategory: accounttype.CategoryRevenue, PeriodCreditMinor: 7000},
		{GLAccountID: pulid.MustNew("gla_"), AccountCode: "6000", AccountName: "Expense", AccountCategory: accounttype.CategoryExpense, PeriodDebitMinor: 2000},
	}}}

	statement, err := svc.GetBalanceSheet(t.Context(), pagination.TenantInfo{}, periodID)
	require.NoError(t, err)
	require.NotNil(t, statement)
	assert.Equal(t, int64(10000), statement.TotalAssetsMinor)
	assert.Equal(t, int64(3000), statement.TotalLiabilitiesMinor)
	assert.Equal(t, int64(2000), statement.Equity.TotalMinor)
	assert.Equal(t, int64(5000), statement.CurrentPeriodNetIncomeMinor)
	assert.Equal(t, int64(7000), statement.TotalEquityMinor)
}

type fakeGLBalanceRepo struct {
	balances []*glbalance.PeriodAccountBalance
}

func (f fakeGLBalanceRepo) ListTrialBalanceByPeriod(context.Context, repositories.ListTrialBalanceByPeriodRequest) ([]*glbalance.PeriodAccountBalance, error) {
	return f.balances, nil
}
