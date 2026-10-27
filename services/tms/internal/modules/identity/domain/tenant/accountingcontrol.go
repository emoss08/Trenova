package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*AccountingControl)(nil)
)

type AccountingControl struct {
	keel.Table `keel:"accounting_controls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ac_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	AccountingBasis          AccountingBasisType          `json:"accountingBasis"          keel:"accounting_basis"`
	RevenueRecognitionPolicy RevenueRecognitionPolicyType `json:"revenueRecognitionPolicy" keel:"revenue_recognition_policy"`
	ExpenseRecognitionPolicy ExpenseRecognitionPolicyType `json:"expenseRecognitionPolicy" keel:"expense_recognition_policy"`

	JournalPostingMode       JournalPostingModeType    `json:"journalPostingMode"       keel:"journal_posting_mode"`
	AutoPostSourceEvents     []JournalSourceEventType  `json:"autoPostSourceEvents"     keel:"auto_post_source_events"`
	ManualJournalEntryPolicy ManualJournalEntryPolicy  `json:"manualJournalEntryPolicy" keel:"manual_journal_entry_policy"`
	RequireManualJEApproval  bool                      `json:"requireManualJeApproval"  keel:"require_manual_je_approval"`
	JournalReversalPolicy    JournalReversalPolicyType `json:"journalReversalPolicy"    keel:"journal_reversal_policy"`

	PeriodCloseMode              PeriodCloseModeType       `json:"periodCloseMode"              keel:"period_close_mode"`
	RequirePeriodCloseApproval   bool                      `json:"requirePeriodCloseApproval"   keel:"require_period_close_approval"`
	LockedPeriodPostingPolicy    LockedPeriodPostingPolicy `json:"lockedPeriodPostingPolicy"    keel:"locked_period_posting_policy"`
	ClosedPeriodPostingPolicy    ClosedPeriodPostingPolicy `json:"closedPeriodPostingPolicy"    keel:"closed_period_posting_policy"`
	RequireReconciliationToClose bool                      `json:"requireReconciliationToClose" keel:"require_reconciliation_to_close"`

	ReconciliationMode              ReconciliationModeType `json:"reconciliationMode"              keel:"reconciliation_mode"`
	ReconciliationToleranceAmount   decimal.Decimal        `json:"reconciliationToleranceAmount"   keel:"reconciliation_tolerance_amount"`
	NotifyOnReconciliationException bool                   `json:"notifyOnReconciliationException" keel:"notify_on_reconciliation_exception"`

	CurrencyMode               CurrencyModeType         `json:"currencyMode"               keel:"currency_mode"`
	FunctionalCurrencyCode     string                   `json:"functionalCurrencyCode"     keel:"functional_currency_code"`
	ExchangeRateDatePolicy     ExchangeRateDatePolicy   `json:"exchangeRateDatePolicy"     keel:"exchange_rate_date_policy"`
	ExchangeRateOverridePolicy ExchangeRateOverrideType `json:"exchangeRateOverridePolicy" keel:"exchange_rate_override_policy"`

	DefaultRevenueAccountID          pulid.ID `json:"defaultRevenueAccountId"          keel:"default_revenue_account_id,nullzero"`
	DefaultCashAccountID             pulid.ID `json:"defaultCashAccountId"             keel:"default_cash_account_id,nullzero"`
	DefaultUnappliedCashAccountID    pulid.ID `json:"defaultUnappliedCashAccountId"    keel:"default_unapplied_cash_account_id,nullzero"`
	DefaultExpenseAccountID          pulid.ID `json:"defaultExpenseAccountId"          keel:"default_expense_account_id,nullzero"`
	DefaultARAccountID               pulid.ID `json:"defaultArAccountId"               keel:"default_ar_account_id,nullzero"`
	DefaultAPAccountID               pulid.ID `json:"defaultApAccountId"               keel:"default_ap_account_id,nullzero"`
	DefaultTaxLiabilityAccountID     pulid.ID `json:"defaultTaxLiabilityAccountId"     keel:"default_tax_liability_account_id,nullzero"`
	DefaultWriteOffAccountID         pulid.ID `json:"defaultWriteOffAccountId"         keel:"default_write_off_account_id,nullzero"`
	DefaultRetainedEarningsAccountID pulid.ID `json:"defaultRetainedEarningsAccountId" keel:"default_retained_earnings_account_id,nullzero"`
	RealizedFXGainAccountID          pulid.ID `json:"realizedFxGainAccountId"          keel:"realized_fx_gain_account_id,nullzero"`
	RealizedFXLossAccountID          pulid.ID `json:"realizedFxLossAccountId"          keel:"realized_fx_loss_account_id,nullzero"`

	DefaultDriverPayExpenseAccountID        pulid.ID `json:"defaultDriverPayExpenseAccountId"        keel:"default_driver_pay_expense_account_id,nullzero"`
	DefaultPurchasedTransportationAccountID pulid.ID `json:"defaultPurchasedTransportationAccountId" keel:"default_purchased_transportation_account_id,nullzero"`
	DefaultSettlementsPayableAccountID      pulid.ID `json:"defaultSettlementsPayableAccountId"      keel:"default_settlements_payable_account_id,nullzero"`
	DefaultDriverAdvanceAccountID           pulid.ID `json:"defaultDriverAdvanceAccountId"           keel:"default_driver_advance_account_id,nullzero"`
	DefaultEscrowLiabilityAccountID         pulid.ID `json:"defaultEscrowLiabilityAccountId"         keel:"default_escrow_liability_account_id,nullzero"`
	DefaultDriverReimbursementAccountID     pulid.ID `json:"defaultDriverReimbursementAccountId"     keel:"default_driver_reimbursement_account_id,nullzero"`
	DefaultEscrowInterestExpenseAccountID   pulid.ID `json:"defaultEscrowInterestExpenseAccountId"   keel:"default_escrow_interest_expense_account_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

func (ac *AccountingControl) BeforeCreate(_ context.Context) error {
	if ac.AccountingBasis == "" {
		ac.AccountingBasis = AccountingBasisAccrual
	}
	if ac.RevenueRecognitionPolicy == "" {
		ac.RevenueRecognitionPolicy = RevenueRecognitionOnInvoicePost
	}
	if ac.ExpenseRecognitionPolicy == "" {
		ac.ExpenseRecognitionPolicy = ExpenseRecognitionOnVendorBillPost
	}
	if ac.JournalPostingMode == "" {
		ac.JournalPostingMode = JournalPostingModeManual
	}
	if ac.ManualJournalEntryPolicy == "" {
		ac.ManualJournalEntryPolicy = ManualJournalEntryPolicyAdjustmentOnly
	}
	if ac.JournalReversalPolicy == "" {
		ac.JournalReversalPolicy = JournalReversalPolicyNextOpenPeriod
	}
	if ac.PeriodCloseMode == "" {
		ac.PeriodCloseMode = PeriodCloseModeManualOnly
	}
	if ac.LockedPeriodPostingPolicy == "" {
		ac.LockedPeriodPostingPolicy = LockedPeriodPostingPolicyBlockSubledgerAllowManualJe
	}
	if ac.ClosedPeriodPostingPolicy == "" {
		ac.ClosedPeriodPostingPolicy = ClosedPeriodPostingPolicyRequireReopen
	}
	if ac.ReconciliationMode == "" {
		ac.ReconciliationMode = ReconciliationModeDisabled
	}
	if ac.CurrencyMode == "" {
		ac.CurrencyMode = CurrencyModeSingleCurrency
	}
	if ac.FunctionalCurrencyCode == "" {
		ac.FunctionalCurrencyCode = "USD"
	}
	if ac.ExchangeRateDatePolicy == "" {
		ac.ExchangeRateDatePolicy = ExchangeRateDatePolicyDocumentDate
	}
	if ac.ExchangeRateOverridePolicy == "" {
		ac.ExchangeRateOverridePolicy = ExchangeRateOverrideRequireApproval
	}

	return nil
}

func (ac *AccountingControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		ac,
		validation.Field(&ac.AccountingBasis, validation.Required),
		validation.Field(&ac.RevenueRecognitionPolicy, validation.Required),
		validation.Field(&ac.ExpenseRecognitionPolicy, validation.Required),
		validation.Field(&ac.JournalPostingMode, validation.Required),
		validation.Field(&ac.ManualJournalEntryPolicy, validation.Required),
		validation.Field(&ac.JournalReversalPolicy, validation.Required),
		validation.Field(&ac.PeriodCloseMode, validation.Required),
		validation.Field(&ac.LockedPeriodPostingPolicy, validation.Required),
		validation.Field(&ac.ClosedPeriodPostingPolicy, validation.Required),
		validation.Field(&ac.ReconciliationMode, validation.Required),
		validation.Field(&ac.CurrencyMode, validation.Required),
		validation.Field(&ac.ExchangeRateDatePolicy, validation.Required),
		validation.Field(&ac.ExchangeRateOverridePolicy, validation.Required),
		validation.Field(
			&ac.FunctionalCurrencyCode,
			validation.Required,
			validation.Length(3, 3),
		),
	))
}

func (ac *AccountingControl) GetID() pulid.ID {
	return ac.ID
}

func (ac *AccountingControl) GetTableName() string {
	return "accounting_controls"
}

func (ac *AccountingControl) GetOrganizationID() pulid.ID {
	return ac.OrganizationID
}

func (ac *AccountingControl) GetBusinessUnitID() pulid.ID {
	return ac.BusinessUnitID
}
