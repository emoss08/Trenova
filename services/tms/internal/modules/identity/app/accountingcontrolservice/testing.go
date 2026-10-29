package accountingcontrolservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: newValidatorBuilder().Build(),
	}
}

func NewTestValidatorWithGLAccountCheck(
	glAccountExists validationframework.CustomReferenceCheckFunc,
) *Validator {
	builder := newValidatorBuilder().WithReferenceChecker(customChecksOnly{})
	addGLAccountReferenceChecks(builder, glAccountExists)

	return &Validator{
		validator: builder.Build(),
	}
}

func NewTestValidatorWithIntegrationRepository(
	integrationRepo integrationports.IntegrationRepository,
) *Validator {
	return &Validator{
		validator: newValidatorBuilderWithIntegrationRepo(integrationRepo).Build(),
	}
}

type customChecksOnly struct{}

func (customChecksOnly) CheckReference(
	context.Context,
	*validationframework.ReferenceRequest,
) (bool, error) {
	return false, errors.New("accounting control references are checked through custom checks")
}

func validAccountingControl() *accountingcontrol.AccountingControl {
	return &accountingcontrol.AccountingControl{
		ID:                       pulid.MustNew("ac_"),
		BusinessUnitID:           pulid.MustNew("bu_"),
		OrganizationID:           pulid.MustNew("org_"),
		AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
		RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
		ExpenseRecognitionPolicy: accountingcontrol.ExpenseRecognitionOnVendorBillPost,
		JournalPostingMode:       accountingcontrol.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
			accountingcontrol.JournalSourceEventInvoicePosted,
			accountingcontrol.JournalSourceEventCreditMemoPosted,
			accountingcontrol.JournalSourceEventDebitMemoPosted,
			accountingcontrol.JournalSourceEventVendorBillPosted,
		},
		ManualJournalEntryPolicy:         accountingcontrol.ManualJournalEntryPolicyAdjustmentOnly,
		RequireManualJEApproval:          true,
		JournalReversalPolicy:            accountingcontrol.JournalReversalPolicyNextOpenPeriod,
		PeriodCloseMode:                  accountingcontrol.PeriodCloseModeManualOnly,
		RequirePeriodCloseApproval:       true,
		LockedPeriodPostingPolicy:        accountingcontrol.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
		ClosedPeriodPostingPolicy:        accountingcontrol.ClosedPeriodPostingPolicyRequireReopen,
		ReconciliationMode:               accountingcontrol.ReconciliationModeWarnOnly,
		ReconciliationToleranceAmount:    decimal.RequireFromString("0.0050"),
		NotifyOnReconciliationException:  true,
		CurrencyMode:                     accountingcontrol.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:           "USD",
		ExchangeRateDatePolicy:           accountingcontrol.ExchangeRateDatePolicyDocumentDate,
		ExchangeRateOverridePolicy:       accountingcontrol.ExchangeRateOverrideDisallow,
		DefaultRevenueAccountID:          pulid.MustNew("gla_"),
		DefaultCashAccountID:             pulid.MustNew("gla_"),
		DefaultUnappliedCashAccountID:    pulid.MustNew("gla_"),
		DefaultExpenseAccountID:          pulid.MustNew("gla_"),
		DefaultARAccountID:               pulid.MustNew("gla_"),
		DefaultAPAccountID:               pulid.MustNew("gla_"),
		DefaultTaxLiabilityAccountID:     pulid.MustNew("gla_"),
		DefaultWriteOffAccountID:         pulid.MustNew("gla_"),
		DefaultRetainedEarningsAccountID: pulid.MustNew("gla_"),
	}
}
