package fiscalcloseservice

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// periodTarget names the fiscal period an entry will post into. ID is nil when
// the period does not exist yet: the year-end adjusting period is created by the
// close itself so that the closing entry never lands in an operating period.
type periodTarget struct {
	ID             pulid.ID
	Name           string
	Status         fiscalperiod.Status
	PeriodNumber   int
	FiscalYearID   pulid.ID
	AccountingDate int64
	MustCreate     bool
	StartDate      int64
	EndDate        int64
}

// retainedEarningsAccount is the equity account the year result rolls into.
type retainedEarningsAccount struct {
	ID       pulid.ID
	Code     string
	Name     string
	Category accounttype.Category
}

type planInputs struct {
	fiscalYear       *fiscalyear.FiscalYear
	nextFiscalYear   *fiscalyear.FiscalYear
	closingPeriod    periodTarget
	openingPeriod    periodTarget
	retainedEarnings retainedEarningsAccount
	balances         []*repositories.GLPeriodAccountBalance
	revision         int
	subledgerChecks  []*fiscalclose.SubledgerCheck
	blockers         []*fiscalclose.Blocker
}

type accountBalance struct {
	glAccountID pulid.ID
	code        string
	name        string
	category    accounttype.Category
	// minor is the debit-positive balance: debits less credits.
	minor int64
}

// buildPlan turns a ledger position into the two entries that close a year. It
// is pure: every database read happens before it is called, which is what makes
// the close preview and the close itself produce the same numbers.
func buildPlan(in *planInputs) *fiscalclose.Plan {
	plan := &fiscalclose.Plan{
		FiscalYearID:              in.fiscalYear.ID,
		FiscalYearName:            in.fiscalYear.Name,
		RetainedEarningsAccountID: in.retainedEarnings.ID,
		RetainedEarningsCode:      in.retainedEarnings.Code,
		RetainedEarningsName:      in.retainedEarnings.Name,
		Revision:                  in.revision,
		Blockers:                  in.blockers,
	}
	if plan.Blockers == nil {
		plan.Blockers = make([]*fiscalclose.Blocker, 0)
	}
	if in.nextFiscalYear != nil {
		plan.NextFiscalYearID = in.nextFiscalYear.ID
		plan.NextFiscalYearName = in.nextFiscalYear.Name
	}

	income, balanceSheet := partitionBalances(in.balances)
	plan.RevenueMinor, plan.CostOfRevenueMinor, plan.OperatingExpenseMinor = summarizeIncome(income)
	plan.NetIncomeMinor = plan.RevenueMinor - plan.CostOfRevenueMinor - plan.OperatingExpenseMinor

	plan.ClosingEntry = buildClosingEntry(in, income, plan.NetIncomeMinor)
	plan.OpeningEntry = buildOpeningEntry(in, balanceSheet, plan.NetIncomeMinor)

	plan.SubledgerChecks = in.subledgerChecks
	if plan.SubledgerChecks == nil {
		plan.SubledgerChecks = make([]*fiscalclose.SubledgerCheck, 0)
	}

	plan.Blockers = append(plan.Blockers, balanceBlockers(plan)...)
	plan.Blockers = append(plan.Blockers, subledgerBlockers(plan.SubledgerChecks)...)
	plan.CanClose = len(plan.Blockers) == 0

	return plan
}

func partitionBalances(
	balances []*repositories.GLPeriodAccountBalance,
) (income, balanceSheet []accountBalance) {
	income = make([]accountBalance, 0, len(balances))
	balanceSheet = make([]accountBalance, 0, len(balances))

	for _, balance := range balances {
		if balance == nil {
			continue
		}
		entry := accountBalance{
			glAccountID: balance.GLAccountID,
			code:        balance.AccountCode,
			name:        balance.AccountName,
			category:    balance.AccountCategory,
			minor:       balance.PeriodDebitMinor - balance.PeriodCreditMinor,
		}
		switch balance.AccountCategory {
		case accounttype.CategoryRevenue,
			accounttype.CategoryCostOfRevenue,
			accounttype.CategoryExpense:
			income = append(income, entry)
		case accounttype.CategoryAsset,
			accounttype.CategoryLiability,
			accounttype.CategoryEquity:
			balanceSheet = append(balanceSheet, entry)
		}
	}

	return income, balanceSheet
}

func summarizeIncome(income []accountBalance) (revenue, costOfRevenue, operatingExpense int64) {
	for _, entry := range income {
		switch entry.category {
		case accounttype.CategoryRevenue:
			revenue -= entry.minor
		case accounttype.CategoryCostOfRevenue:
			costOfRevenue += entry.minor
		case accounttype.CategoryExpense:
			operatingExpense += entry.minor
		case accounttype.CategoryAsset,
			accounttype.CategoryLiability,
			accounttype.CategoryEquity:
		}
	}

	return revenue, costOfRevenue, operatingExpense
}

// buildClosingEntry reverses every income-statement balance and books the
// difference to retained earnings. A year with no income-statement activity
// produces no entry at all rather than an empty one.
func buildClosingEntry(
	in *planInputs,
	income []accountBalance,
	netIncomeMinor int64,
) *fiscalclose.PlanEntry {
	lines := make([]*fiscalclose.PlanLine, 0, len(income)+1)
	for _, entry := range income {
		if entry.minor == 0 {
			continue
		}
		lines = append(lines, newPlanLine(&entry, -entry.minor, false))
	}

	if len(lines) == 0 {
		return nil
	}

	// The income-statement lines above net to the year result as a debit; the
	// balancing entry is the opposite, so a profit credits retained earnings and
	// a loss debits it.
	if netIncomeMinor != 0 {
		lines = append(lines, newPlanLine(&accountBalance{
			glAccountID: in.retainedEarnings.ID,
			code:        in.retainedEarnings.Code,
			name:        in.retainedEarnings.Name,
			category:    in.retainedEarnings.Category,
		}, -netIncomeMinor, true))
	}

	sortLines(lines)

	return newPlanEntry(
		fiscalclose.EntryKindClosing,
		&in.closingPeriod,
		fmt.Sprintf("Year-end close of %s", in.fiscalYear.Name),
		lines,
	)
}

// buildOpeningEntry re-establishes every balance-sheet account in the first
// period of the next fiscal year. Retained earnings is adjusted by the result
// first, because the closing entry books that result before the balance sheet is
// carried forward.
func buildOpeningEntry(
	in *planInputs,
	balanceSheet []accountBalance,
	netIncomeMinor int64,
) *fiscalclose.PlanEntry {
	carried := make([]accountBalance, 0, len(balanceSheet)+1)
	retainedEarningsSeen := false

	for _, entry := range balanceSheet {
		if entry.glAccountID == in.retainedEarnings.ID {
			entry.minor -= netIncomeMinor
			retainedEarningsSeen = true
		}
		carried = append(carried, entry)
	}

	if !retainedEarningsSeen && netIncomeMinor != 0 && in.retainedEarnings.ID.IsNotNil() {
		carried = append(carried, accountBalance{
			glAccountID: in.retainedEarnings.ID,
			code:        in.retainedEarnings.Code,
			name:        in.retainedEarnings.Name,
			category:    in.retainedEarnings.Category,
			minor:       -netIncomeMinor,
		})
	}

	lines := make([]*fiscalclose.PlanLine, 0, len(carried))
	for _, entry := range carried {
		if entry.minor == 0 {
			continue
		}
		lines = append(
			lines,
			newPlanLine(&entry, entry.minor, entry.glAccountID == in.retainedEarnings.ID),
		)
	}

	if len(lines) == 0 {
		return nil
	}

	sortLines(lines)

	nextName := ""
	if in.nextFiscalYear != nil {
		nextName = in.nextFiscalYear.Name
	}

	return newPlanEntry(
		fiscalclose.EntryKindOpening,
		&in.openingPeriod,
		fmt.Sprintf(
			"Opening balances carried forward from %s to %s",
			in.fiscalYear.Name,
			nextName,
		),
		lines,
	)
}

// newPlanLine converts a debit-positive amount into the debit or credit column.
func newPlanLine(
	entry *accountBalance,
	minor int64,
	isRetainedEarnings bool,
) *fiscalclose.PlanLine {
	line := &fiscalclose.PlanLine{
		GLAccountID:     entry.glAccountID,
		AccountCode:     entry.code,
		AccountName:     entry.name,
		AccountCategory: entry.category,
		IsRetainedEarn:  isRetainedEarnings,
	}
	if minor >= 0 {
		line.DebitMinor = minor
		return line
	}
	line.CreditMinor = -minor

	return line
}

func newPlanEntry(
	kind fiscalclose.EntryKind,
	target *periodTarget,
	description string,
	lines []*fiscalclose.PlanLine,
) *fiscalclose.PlanEntry {
	entry := &fiscalclose.PlanEntry{
		Kind:             kind,
		FiscalYearID:     target.FiscalYearID,
		FiscalPeriodID:   target.ID,
		FiscalPeriodName: target.Name,
		CreatesPeriod:    target.MustCreate,
		AccountingDate:   target.AccountingDate,
		Description:      description,
		Lines:            lines,
	}
	for _, line := range lines {
		entry.TotalDebitMinor += line.DebitMinor
		entry.TotalCreditMinor += line.CreditMinor
	}

	return entry
}

func sortLines(lines []*fiscalclose.PlanLine) {
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].AccountCode == lines[j].AccountCode {
			return lines[i].GLAccountID.String() < lines[j].GLAccountID.String()
		}
		return lines[i].AccountCode < lines[j].AccountCode
	})
}

// balanceBlockers is the last line of defence: a close must never post an
// unbalanced entry, so a ledger that does not foot stops the close instead.
func balanceBlockers(plan *fiscalclose.Plan) []*fiscalclose.Blocker {
	blockers := make([]*fiscalclose.Blocker, 0, 2)
	for _, entry := range []*fiscalclose.PlanEntry{plan.ClosingEntry, plan.OpeningEntry} {
		if entry.IsEmpty() || entry.TotalDebitMinor == entry.TotalCreditMinor {
			continue
		}
		blockers = append(blockers, &fiscalclose.Blocker{
			Field: "__all__",
			Code:  errortypes.ErrInvalid,
			Message: fmt.Sprintf(
				"The %s entry does not balance (debits %d, credits %d). Review the trial balance before closing.",
				entry.Kind,
				entry.TotalDebitMinor,
				entry.TotalCreditMinor,
			),
			Category: "accounting",
		})
	}

	return blockers
}
