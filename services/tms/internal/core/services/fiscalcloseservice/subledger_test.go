package fiscalcloseservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func arBalances(debit, credit int64) []*repositories.GLPeriodAccountBalance {
	return []*repositories.GLPeriodAccountBalance{
		balance(cashID, "1000", accounttype.CategoryAsset, 5000, 0),
		{
			GLAccountID:       arID,
			AccountCode:       "1110",
			AccountName:       "Accounts Receivable",
			AccountCategory:   accounttype.CategoryAsset,
			PeriodDebitMinor:  debit,
			PeriodCreditMinor: credit,
			NetChangeMinor:    debit - credit,
		},
	}
}

func checkFor(subledgerMinor, toleranceMinor int64, enforced bool) *fiscalclose.SubledgerCheck {
	return newSubledgerCheck(subledgerCheckInput{
		key:              "accounts_receivable",
		label:            "Accounts Receivable",
		glAccountID:      arID,
		balances:         arBalances(90000, 15000),
		subledgerMinor:   subledgerMinor,
		toleranceMinor:   toleranceMinor,
		reconcileEnabled: enforced,
	})
}

func TestSubledgerCheckReconcilesTheControlAccount(t *testing.T) {
	t.Parallel()

	// The GL control account nets to 75,000; the subledger agrees.
	check := checkFor(75000, 0, true)

	assert.Equal(t, "1110", check.AccountCode)
	assert.Equal(t, int64(75000), check.GLBalanceMinor)
	assert.Equal(t, int64(75000), check.SubledgerBalanceMinor)
	assert.Zero(t, check.DifferenceMinor)
	assert.True(t, check.Reconciled)
	assert.Empty(t, subledgerBlockers([]*fiscalclose.SubledgerCheck{check}))
}

func TestSubledgerCheckReportsADifference(t *testing.T) {
	t.Parallel()

	check := checkFor(74000, 0, true)

	assert.Equal(t, int64(1000), check.DifferenceMinor)
	assert.False(t, check.Reconciled)

	blockers := subledgerBlockers([]*fiscalclose.SubledgerCheck{check})
	require.Len(t, blockers, 1)
	assert.Equal(t, "reconciliation", blockers[0].Field)
	// Amounts read as money, not as raw minor units.
	assert.Contains(t, blockers[0].Message, "750.00")
	assert.Contains(t, blockers[0].Message, "740.00")
	assert.Contains(t, blockers[0].Message, "10.00")
}

func TestSubledgerCheckHonoursTheConfiguredTolerance(t *testing.T) {
	t.Parallel()

	assert.False(t, checkFor(74000, 500, true).Reconciled, "1,000 out with 500 allowed")
	assert.True(t, checkFor(74000, 1000, true).Reconciled, "1,000 out with 1,000 allowed")
	assert.True(t, checkFor(76000, 1000, true).Reconciled, "tolerance applies in both directions")
}

// The check is always computed and always shown. Whether a difference stops the
// close is the carrier's call, so a tenant that turned reconciliation off still
// sees the figure but is not blocked by it.
func TestSubledgerCheckOnlyBlocksWhenEnforced(t *testing.T) {
	t.Parallel()

	unenforced := checkFor(74000, 0, false)

	assert.False(t, unenforced.Reconciled)
	assert.False(t, unenforced.Enforced)
	assert.Empty(t, subledgerBlockers([]*fiscalclose.SubledgerCheck{unenforced}))
}

func TestSubledgerCheckIsBlankWhenTheAccountHasNoActivity(t *testing.T) {
	t.Parallel()

	check := newSubledgerCheck(subledgerCheckInput{
		key:            "accounts_receivable",
		label:          "Accounts Receivable",
		glAccountID:    arID,
		balances:       []*repositories.GLPeriodAccountBalance{},
		subledgerMinor: 0,
	})

	assert.Zero(t, check.GLBalanceMinor)
	assert.True(t, check.Reconciled)
}

func TestReconciliationEnforcedFollowsTheAccountingControl(t *testing.T) {
	t.Parallel()

	assert.True(t, reconciliationEnforced(&tenant.AccountingControl{
		RequireReconciliationToClose: true,
		ReconciliationMode:           tenant.ReconciliationModeBlockPosting,
	}))
	assert.False(t, reconciliationEnforced(&tenant.AccountingControl{
		RequireReconciliationToClose: true,
		ReconciliationMode:           tenant.ReconciliationModeDisabled,
	}), "a disabled reconciliation mode wins over the require flag")
	assert.False(t, reconciliationEnforced(&tenant.AccountingControl{
		RequireReconciliationToClose: false,
		ReconciliationMode:           tenant.ReconciliationModeBlockPosting,
	}))
}

func TestToleranceConvertsMajorUnitsToMinor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(0), toleranceMinor(decimal.Zero))
	assert.Equal(t, int64(150), toleranceMinor(decimal.RequireFromString("1.50")))
	assert.Equal(t, int64(2500), toleranceMinor(decimal.RequireFromString("25")))
	assert.Equal(t, int64(0), toleranceMinor(decimal.RequireFromString("-5")),
		"a negative tolerance is not a licence to be wrong")
}

// The opening entry carries one line per GL account and no customer tag: the
// detail behind the control account lives in the AR subledger, which has no year
// boundary to survive.
func TestOpeningEntryCarriesOneLinePerAccount(t *testing.T) {
	t.Parallel()

	in := profitableYearInputs()
	plan := buildPlan(in)

	seen := make(map[string]int, len(plan.OpeningEntry.Lines))
	for _, line := range plan.OpeningEntry.Lines {
		seen[line.AccountCode]++
	}
	for code, count := range seen {
		assert.Equal(t, 1, count, "account %s should carry forward once", code)
	}
}
