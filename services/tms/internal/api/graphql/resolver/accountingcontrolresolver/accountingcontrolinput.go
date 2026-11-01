package accountingcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
)

func accountingControlFromInput(
	existing *accountingcontrol.AccountingControl,
	input *gqlmodel.UpdateAccountingControlInput,
) (*accountingcontrol.AccountingControl, error) {
	parse := gqlinput.NewParser()
	entity := &accountingcontrol.AccountingControl{
		ID:                           existing.ID,
		BusinessUnitID:               existing.BusinessUnitID,
		OrganizationID:               existing.OrganizationID,
		Version:                      int64(input.Version),
		CreatedAt:                    existing.CreatedAt,
		UpdatedAt:                    existing.UpdatedAt,
		BusinessUnit:                 existing.BusinessUnit,
		Organization:                 existing.Organization,
		AccountingBasis:              input.AccountingBasis,
		RevenueRecognitionPolicy:     input.RevenueRecognitionPolicy,
		ExpenseRecognitionPolicy:     input.ExpenseRecognitionPolicy,
		JournalPostingMode:           input.JournalPostingMode,
		AutoPostSourceEvents:         input.AutoPostSourceEvents,
		ManualJournalEntryPolicy:     input.ManualJournalEntryPolicy,
		RequireManualJEApproval:      input.RequireManualJeApproval,
		JournalReversalPolicy:        input.JournalReversalPolicy,
		PeriodCloseMode:              input.PeriodCloseMode,
		RequirePeriodCloseApproval:   input.RequirePeriodCloseApproval,
		LockedPeriodPostingPolicy:    input.LockedPeriodPostingPolicy,
		ClosedPeriodPostingPolicy:    input.ClosedPeriodPostingPolicy,
		RequireReconciliationToClose: input.RequireReconciliationToClose,
		ReconciliationMode:           input.ReconciliationMode,
		ReconciliationToleranceAmount: parse.Decimal(
			"reconciliationToleranceAmount",
			input.ReconciliationToleranceAmount,
		),
		NotifyOnReconciliationException: input.NotifyOnReconciliationException,
		CurrencyMode:                    input.CurrencyMode,
		FunctionalCurrencyCode:          input.FunctionalCurrencyCode,
		ExchangeRateDatePolicy:          input.ExchangeRateDatePolicy,
		ExchangeRateOverridePolicy:      input.ExchangeRateOverridePolicy,
		DefaultRevenueAccountID: parse.OptionalID(
			"defaultRevenueAccountId",
			input.DefaultRevenueAccountID,
		),
		DefaultCashAccountID: parse.OptionalID(
			"defaultCashAccountId",
			input.DefaultCashAccountID,
		),
		DefaultUnappliedCashAccountID: parse.OptionalID(
			"defaultUnappliedCashAccountId",
			input.DefaultUnappliedCashAccountID,
		),
		DefaultExpenseAccountID: parse.OptionalID(
			"defaultExpenseAccountId",
			input.DefaultExpenseAccountID,
		),
		DefaultARAccountID: parse.OptionalID(
			"defaultArAccountId",
			input.DefaultArAccountID,
		),
		DefaultAPAccountID: parse.OptionalID(
			"defaultApAccountId",
			input.DefaultApAccountID,
		),
		DefaultTaxLiabilityAccountID: parse.OptionalID(
			"defaultTaxLiabilityAccountId",
			input.DefaultTaxLiabilityAccountID,
		),
		DefaultWriteOffAccountID: parse.OptionalID(
			"defaultWriteOffAccountId",
			input.DefaultWriteOffAccountID,
		),
		DefaultRetainedEarningsAccountID: parse.OptionalID(
			"defaultRetainedEarningsAccountId",
			input.DefaultRetainedEarningsAccountID,
		),
		RealizedFXGainAccountID: parse.OptionalID(
			"realizedFxGainAccountId",
			input.RealizedFxGainAccountID,
		),
		RealizedFXLossAccountID: parse.OptionalID(
			"realizedFxLossAccountId",
			input.RealizedFxLossAccountID,
		),
		DefaultDriverPayExpenseAccountID: parse.OptionalID(
			"defaultDriverPayExpenseAccountId",
			input.DefaultDriverPayExpenseAccountID,
		),
		DefaultPurchasedTransportationAccountID: parse.OptionalID(
			"defaultPurchasedTransportationAccountId",
			input.DefaultPurchasedTransportationAccountID,
		),
		DefaultSettlementsPayableAccountID: parse.OptionalID(
			"defaultSettlementsPayableAccountId",
			input.DefaultSettlementsPayableAccountID,
		),
		DefaultDriverAdvanceAccountID: parse.OptionalID(
			"defaultDriverAdvanceAccountId",
			input.DefaultDriverAdvanceAccountID,
		),
		DefaultEscrowLiabilityAccountID: parse.OptionalID(
			"defaultEscrowLiabilityAccountId",
			input.DefaultEscrowLiabilityAccountID,
		),
		DefaultDriverReimbursementAccountID: parse.OptionalID(
			"defaultDriverReimbursementAccountId",
			input.DefaultDriverReimbursementAccountID,
		),
		DefaultEscrowInterestExpenseAccountID: parse.OptionalID(
			"defaultEscrowInterestExpenseAccountId",
			input.DefaultEscrowInterestExpenseAccountID,
		),
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
