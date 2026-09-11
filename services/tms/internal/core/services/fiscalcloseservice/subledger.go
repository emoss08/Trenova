package fiscalcloseservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const minorUnitsPerMajor = 100

// reconcileSubledgers checks each general-ledger control account against the
// subledger that carries its detail.
//
// This is the control that makes the carryforward's shape correct. The opening
// entry aggregates by GL account and drops the customer tag on purpose: customer
// balances live in customer_ledger_entries, which is append-only and has no
// fiscal year, so they were never at risk at the year boundary. Tagging the
// opening entry by customer would duplicate the subledger and, worse, make a new
// year's AR movement-by-customer report open with a phantom spike equal to the
// prior year's closing balance. What the close owes instead is proof that the one
// total it does carry still agrees with the detail behind it.
func (s *Service) reconcileSubledgers(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	control *tenant.AccountingControl,
	balances []*repositories.GLPeriodAccountBalance,
) ([]*fiscalclose.SubledgerCheck, error) {
	checks := make([]*fiscalclose.SubledgerCheck, 0, 1)

	if control == nil || control.DefaultARAccountID.IsNil() || s.customerLedgerRepo == nil {
		return checks, nil
	}

	subledgerBalance, err := s.customerLedgerRepo.SumBalanceAsOf(
		ctx,
		repositories.SumCustomerLedgerBalanceRequest{
			TenantInfo: tenantOf(fy),
			AsOfDate:   fy.EndDate,
		},
	)
	if err != nil {
		return nil, err
	}

	check := newSubledgerCheck(subledgerCheckInput{
		key:              "accounts_receivable",
		label:            "Accounts Receivable",
		glAccountID:      control.DefaultARAccountID,
		balances:         balances,
		subledgerMinor:   subledgerBalance,
		toleranceMinor:   toleranceMinor(control.ReconciliationToleranceAmount),
		reconcileEnabled: reconciliationEnforced(control),
	})

	return append(checks, check), nil
}

type subledgerCheckInput struct {
	key              string
	label            string
	glAccountID      pulid.ID
	balances         []*repositories.GLPeriodAccountBalance
	subledgerMinor   int64
	toleranceMinor   int64
	reconcileEnabled bool
}

func newSubledgerCheck(in subledgerCheckInput) *fiscalclose.SubledgerCheck {
	check := &fiscalclose.SubledgerCheck{
		Key:                   in.key,
		Label:                 in.label,
		GLAccountID:           in.glAccountID,
		SubledgerBalanceMinor: in.subledgerMinor,
		ToleranceMinor:        in.toleranceMinor,
		Enforced:              in.reconcileEnabled,
	}

	for _, balance := range in.balances {
		if balance == nil || balance.GLAccountID != in.glAccountID {
			continue
		}
		check.AccountCode = balance.AccountCode
		check.AccountName = balance.AccountName
		check.GLBalanceMinor = balance.PeriodDebitMinor - balance.PeriodCreditMinor
		break
	}

	check.DifferenceMinor = check.GLBalanceMinor - check.SubledgerBalanceMinor
	check.Reconciled = abs64(check.DifferenceMinor) <= in.toleranceMinor

	return check
}

// reconciliationEnforced mirrors the gate the fiscal period close already uses,
// so a carrier that turned reconciliation off does not meet it again at year end.
func reconciliationEnforced(control *tenant.AccountingControl) bool {
	return control.RequireReconciliationToClose &&
		control.ReconciliationMode != tenant.ReconciliationModeDisabled
}

// toleranceMinor converts the configured tolerance, which is held in major
// currency units alongside the invoice amounts it was written for, into the minor
// units the ledger works in.
func toleranceMinor(tolerance decimal.Decimal) int64 {
	if tolerance.IsNegative() {
		return 0
	}

	return tolerance.Mul(decimal.NewFromInt(minorUnitsPerMajor)).Round(0).IntPart()
}

func subledgerBlockers(checks []*fiscalclose.SubledgerCheck) []*fiscalclose.Blocker {
	blockers := make([]*fiscalclose.Blocker, 0, len(checks))
	for _, check := range checks {
		if check == nil || check.Reconciled || !check.Enforced {
			continue
		}
		blockers = append(blockers, accountingBlocker(
			"reconciliation",
			fmt.Sprintf(
				"%s does not reconcile: control account %s carries %s and the subledger carries %s, a difference of %s. Resolve it before closing.",
				check.Label,
				check.AccountCode,
				formatMinor(check.GLBalanceMinor),
				formatMinor(check.SubledgerBalanceMinor),
				formatMinor(check.DifferenceMinor),
			),
		))
	}

	return blockers
}

// formatMinor renders a minor-unit amount for a message a controller reads. The
// division happens at the very end so the arithmetic above stays in integers.
func formatMinor(minor int64) string {
	return decimal.NewFromInt(minor).Shift(-2).StringFixed(2)
}

func abs64(value int64) int64 {
	if value < 0 {
		return -value
	}

	return value
}
