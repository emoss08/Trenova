package accountingcontrol

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	defaultFunctionalCurrency = "USD"
)

func NewDefaultAccountingControl(orgID, buID pulid.ID) *AccountingControl {
	return &AccountingControl{
		OrganizationID:                  orgID,
		BusinessUnitID:                  buID,
		AccountingBasis:                 AccountingBasisAccrual,
		RevenueRecognitionPolicy:        RevenueRecognitionOnInvoicePost,
		ExpenseRecognitionPolicy:        ExpenseRecognitionOnVendorBillPost,
		JournalPostingMode:              JournalPostingModeManual,
		ManualJournalEntryPolicy:        ManualJournalEntryPolicyAdjustmentOnly,
		RequireManualJEApproval:         true,
		JournalReversalPolicy:           JournalReversalPolicyNextOpenPeriod,
		PeriodCloseMode:                 PeriodCloseModeManualOnly,
		RequirePeriodCloseApproval:      true,
		LockedPeriodPostingPolicy:       LockedPeriodPostingPolicy("BlockSubledgerAllowManualJe"),
		ClosedPeriodPostingPolicy:       ClosedPeriodPostingPolicyRequireReopen,
		ReconciliationMode:              ReconciliationModeDisabled,
		ReconciliationToleranceAmount:   decimal.Zero,
		NotifyOnReconciliationException: true,
		CurrencyMode:                    CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:          defaultFunctionalCurrency,
		ExchangeRateDatePolicy:          ExchangeRateDatePolicyDocumentDate,
		ExchangeRateOverridePolicy:      ExchangeRateOverrideRequireApproval,
	}
}
