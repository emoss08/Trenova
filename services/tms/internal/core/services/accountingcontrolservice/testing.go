package accountingcontrolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: newValidatorBuilder().Build(),
	}
}

func NewTestValidatorWithReferenceChecker(
	checker validationframework.ReferenceChecker,
) *Validator {
	builder := newValidatorBuilder().
		WithReferenceChecker(checker)
	addGLAccountReferenceChecks(builder)

	return &Validator{
		validator: builder.Build(),
	}
}

func validAccountingControl() *tenant.AccountingControl {
	return &tenant.AccountingControl{
		ID:                               pulid.MustNew("ac_"),
		BusinessUnitID:                   pulid.MustNew("bu_"),
		OrganizationID:                   pulid.MustNew("org_"),
		AccountingBasis:                  tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy:         tenant.RevenueRecognitionOnInvoicePost,
		ExpenseRecognitionPolicy:         tenant.ExpenseRecognitionOnVendorBillPost,
		JournalPostingMode:               tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents:             []tenant.JournalSourceEventType{tenant.JournalSourceEventInvoicePosted, tenant.JournalSourceEventCreditMemoPosted, tenant.JournalSourceEventDebitMemoPosted, tenant.JournalSourceEventVendorBillPosted},
		ManualJournalEntryPolicy:         tenant.ManualJournalEntryPolicyAdjustmentOnly,
		RequireManualJEApproval:          true,
		JournalReversalPolicy:            tenant.JournalReversalPolicyNextOpenPeriod,
		PeriodCloseMode:                  tenant.PeriodCloseModeManualOnly,
		RequirePeriodCloseApproval:       true,
		LockedPeriodPostingPolicy:        tenant.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
		ClosedPeriodPostingPolicy:        tenant.ClosedPeriodPostingPolicyRequireReopen,
		ReconciliationMode:               tenant.ReconciliationModeWarnOnly,
		ReconciliationToleranceAmount:    decimal.RequireFromString("0.0050"),
		NotifyOnReconciliationException:  true,
		CurrencyMode:                     tenant.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:           "USD",
		ExchangeRateDatePolicy:           tenant.ExchangeRateDatePolicyDocumentDate,
		ExchangeRateOverridePolicy:       tenant.ExchangeRateOverrideDisallow,
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
