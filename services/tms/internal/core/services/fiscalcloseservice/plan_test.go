package fiscalcloseservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cashID      = pulid.MustNew("gla_")
	arID        = pulid.MustNew("gla_")
	apID        = pulid.MustNew("gla_")
	stockID     = pulid.MustNew("gla_")
	retainedID  = pulid.MustNew("gla_")
	revenueID   = pulid.MustNew("gla_")
	costID      = pulid.MustNew("gla_")
	expenseID   = pulid.MustNew("gla_")
	closingYear = &fiscalyear.FiscalYear{
		ID:      pulid.MustNew("fyr_"),
		Name:    "FY 2026",
		EndDate: 1798761599,
	}
	nextYear = &fiscalyear.FiscalYear{
		ID:        pulid.MustNew("fyr_"),
		Name:      "FY 2027",
		StartDate: 1798761600,
	}
)

// balance builds one year-to-date row. debit and credit are gross totals, the
// same shape gl_account_balances_by_period stores.
func balance(
	id pulid.ID,
	code string,
	category accounttype.Category,
	debit, credit int64,
) *repositories.GLPeriodAccountBalance {
	return &repositories.GLPeriodAccountBalance{
		GLAccountID:       id,
		AccountCode:       code,
		AccountName:       code + " account",
		AccountCategory:   category,
		PeriodDebitMinor:  debit,
		PeriodCreditMinor: credit,
		NetChangeMinor:    debit - credit,
	}
}

func profitableYearInputs() *planInputs {
	return &planInputs{
		fiscalYear:     closingYear,
		nextFiscalYear: nextYear,
		closingPeriod: periodTarget{
			Name:           "Adjusting Period - FY 2026",
			PeriodNumber:   13,
			FiscalYearID:   closingYear.ID,
			AccountingDate: closingYear.EndDate,
			MustCreate:     true,
		},
		openingPeriod: periodTarget{
			ID:             pulid.MustNew("fp_"),
			Name:           "Period 1 - January 2027",
			PeriodNumber:   1,
			FiscalYearID:   nextYear.ID,
			AccountingDate: nextYear.StartDate,
		},
		retainedEarnings: retainedEarningsAccount{
			ID:       retainedID,
			Code:     "3900",
			Name:     "Retained Earnings",
			Category: accounttype.CategoryEquity,
		},
		balances: []*repositories.GLPeriodAccountBalance{
			balance(cashID, "1000", accounttype.CategoryAsset, 9000, 0),
			balance(arID, "1100", accounttype.CategoryAsset, 1000, 0),
			balance(apID, "2000", accounttype.CategoryLiability, 0, 3000),
			balance(stockID, "3000", accounttype.CategoryEquity, 0, 3500),
			balance(revenueID, "4000", accounttype.CategoryRevenue, 0, 10000),
			balance(costID, "5000", accounttype.CategoryCostOfRevenue, 4000, 0),
			balance(expenseID, "6000", accounttype.CategoryExpense, 2500, 0),
		},
	}
}

func lineFor(t *testing.T, entry *fiscalclose.PlanEntry, code string) *fiscalclose.PlanLine {
	t.Helper()

	for _, line := range entry.Lines {
		if line.AccountCode == code {
			return line
		}
	}
	t.Fatalf("no line for account %s", code)

	return nil
}

func TestBuildPlanClosesIncomeStatementIntoRetainedEarnings(t *testing.T) {
	t.Parallel()

	plan := buildPlan(profitableYearInputs())

	assert.Equal(t, int64(10000), plan.RevenueMinor)
	assert.Equal(t, int64(4000), plan.CostOfRevenueMinor)
	assert.Equal(t, int64(2500), plan.OperatingExpenseMinor)
	assert.Equal(t, int64(3500), plan.NetIncomeMinor)

	require.NotNil(t, plan.ClosingEntry)
	assert.Equal(t, fiscalclose.EntryKindClosing, plan.ClosingEntry.Kind)
	assert.True(t, plan.ClosingEntry.CreatesPeriod)
	assert.Equal(t, closingYear.EndDate, plan.ClosingEntry.AccountingDate)
	assert.Len(t, plan.ClosingEntry.Lines, 4)
	assert.Equal(t, int64(10000), plan.ClosingEntry.TotalDebitMinor)
	assert.Equal(t, int64(10000), plan.ClosingEntry.TotalCreditMinor)

	assert.Equal(t, int64(10000), lineFor(t, plan.ClosingEntry, "4000").DebitMinor)
	assert.Equal(t, int64(4000), lineFor(t, plan.ClosingEntry, "5000").CreditMinor)
	assert.Equal(t, int64(2500), lineFor(t, plan.ClosingEntry, "6000").CreditMinor)

	retained := lineFor(t, plan.ClosingEntry, "3900")
	assert.Equal(t, int64(3500), retained.CreditMinor)
	assert.True(t, retained.IsRetainedEarn)
}

func TestBuildPlanCarriesBalanceSheetForward(t *testing.T) {
	t.Parallel()

	plan := buildPlan(profitableYearInputs())

	require.NotNil(t, plan.OpeningEntry)
	assert.Equal(t, fiscalclose.EntryKindOpening, plan.OpeningEntry.Kind)
	assert.Equal(t, nextYear.ID, plan.OpeningEntry.FiscalYearID)
	assert.Equal(t, nextYear.StartDate, plan.OpeningEntry.AccountingDate)
	assert.Equal(t, int64(10000), plan.OpeningEntry.TotalDebitMinor)
	assert.Equal(t, int64(10000), plan.OpeningEntry.TotalCreditMinor)

	assert.Equal(t, int64(9000), lineFor(t, plan.OpeningEntry, "1000").DebitMinor)
	assert.Equal(t, int64(1000), lineFor(t, plan.OpeningEntry, "1100").DebitMinor)
	assert.Equal(t, int64(3000), lineFor(t, plan.OpeningEntry, "2000").CreditMinor)
	assert.Equal(t, int64(3500), lineFor(t, plan.OpeningEntry, "3000").CreditMinor)

	// Retained earnings carries the year result even though it had no activity
	// of its own during the year.
	assert.Equal(t, int64(3500), lineFor(t, plan.OpeningEntry, "3900").CreditMinor)

	for _, line := range plan.OpeningEntry.Lines {
		assert.NotEqual(t, accounttype.CategoryRevenue, line.AccountCategory)
		assert.NotEqual(t, accounttype.CategoryExpense, line.AccountCategory)
		assert.NotEqual(t, accounttype.CategoryCostOfRevenue, line.AccountCategory)
	}
	assert.True(t, plan.CanClose)
}

func TestBuildPlanDebitsRetainedEarningsOnALoss(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	in.balances = []*repositories.GLPeriodAccountBalance{
		balance(cashID, "1000", accounttype.CategoryAsset, 0, 2000),
		balance(stockID, "3000", accounttype.CategoryEquity, 0, 4000),
		balance(retainedID, "3900", accounttype.CategoryEquity, 0, 0),
		balance(revenueID, "4000", accounttype.CategoryRevenue, 0, 4000),
		balance(expenseID, "6000", accounttype.CategoryExpense, 10000, 0),
	}

	plan := buildPlan(in)

	assert.Equal(t, int64(-6000), plan.NetIncomeMinor)

	require.NotNil(t, plan.ClosingEntry)
	assert.Equal(t, int64(6000), lineFor(t, plan.ClosingEntry, "3900").DebitMinor)
	assert.Equal(
		t,
		plan.ClosingEntry.TotalDebitMinor,
		plan.ClosingEntry.TotalCreditMinor,
	)

	require.NotNil(t, plan.OpeningEntry)
	assert.Equal(t, int64(6000), lineFor(t, plan.OpeningEntry, "3900").DebitMinor)
	assert.Equal(t, int64(2000), lineFor(t, plan.OpeningEntry, "1000").CreditMinor)
	assert.Equal(
		t,
		plan.OpeningEntry.TotalDebitMinor,
		plan.OpeningEntry.TotalCreditMinor,
	)
}

func TestBuildPlanNetsExistingRetainedEarningsActivity(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	// Prior-year retained earnings of 1,500, funded by the matching cash.
	in.balances = append(
		in.balances,
		balance(retainedID, "3900", accounttype.CategoryEquity, 0, 1500),
	)
	in.balances[0] = balance(cashID, "1000", accounttype.CategoryAsset, 10500, 0)

	plan := buildPlan(in)

	// 1,500 already sitting in retained earnings plus 3,500 of current-year
	// income carries 5,000 forward.
	assert.Equal(t, int64(5000), lineFor(t, plan.OpeningEntry, "3900").CreditMinor)
	assert.Equal(t, plan.OpeningEntry.TotalDebitMinor, plan.OpeningEntry.TotalCreditMinor)
}

func TestBuildPlanProducesNoEntriesForAnEmptyYear(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	in.balances = nil

	plan := buildPlan(in)

	assert.Nil(t, plan.ClosingEntry)
	assert.Nil(t, plan.OpeningEntry)
	assert.Zero(t, plan.NetIncomeMinor)
	assert.True(t, plan.CanClose)
}

func TestBuildPlanSkipsAccountsThatNetToZero(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	in.balances = append(
		in.balances,
		balance(pulid.MustNew("gla_"), "6100", accounttype.CategoryExpense, 900, 900),
	)

	plan := buildPlan(in)

	for _, line := range plan.ClosingEntry.Lines {
		assert.NotEqual(t, "6100", line.AccountCode)
	}
}

func TestBuildPlanBlocksAnUnbalancedLedger(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	in.balances = []*repositories.GLPeriodAccountBalance{
		balance(cashID, "1000", accounttype.CategoryAsset, 9000, 0),
		balance(apID, "2000", accounttype.CategoryLiability, 0, 1000),
	}

	plan := buildPlan(in)

	assert.False(t, plan.CanClose)
	require.Len(t, plan.Blockers, 1)
	assert.Contains(t, plan.Blockers[0].Message, "does not balance")
	assert.Equal(t, "accounting", plan.Blockers[0].Category)
}

func TestBuildPlanKeepsIncomingBlockers(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	in.blockers = []*fiscalclose.Blocker{
		accountingBlocker("defaultRetainedEarningsAccountId", "not configured"),
	}

	plan := buildPlan(in)

	assert.False(t, plan.CanClose)
	require.Len(t, plan.Blockers, 1)
	assert.Equal(t, "defaultRetainedEarningsAccountId", plan.Blockers[0].Field)
}
