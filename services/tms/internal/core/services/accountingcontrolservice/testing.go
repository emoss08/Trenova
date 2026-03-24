package accountingcontrolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/validationframework"
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
		ID:                                "ac_123",
		BusinessUnitID:                    "bu_123",
		OrganizationID:                    "org_123",
		AccountingMethod:                  tenant.AccountingMethodAccrual,
		ReconciliationThresholdAction:     tenant.ThresholdActionWarn,
		RevenueRecognitionMethod:          tenant.RevenueRecognitionOnDelivery,
		ExpenseRecognitionMethod:          tenant.ExpenseRecognitionOnIncurrence,
		DefaultCurrencyCode:               "USD",
		ReconciliationThreshold:           decimal.RequireFromString("0.0050"),
		EnableAutomaticTaxCalculation:     true,
		RequireJournalEntryApproval:       true,
		EnableJournalEntryReversal:        true,
		RequirePeriodEndApproval:          true,
		EnableReconciliationNotifications: true,
		RetainDeletedEntries:              true,
	}
}
