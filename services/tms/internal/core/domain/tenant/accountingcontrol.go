package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AccountingControl)(nil)
	_ validationframework.TenantedEntity = (*AccountingControl)(nil)
)

type AccountingControl struct {
	bun.BaseModel `bun:"table:accounting_controls,alias:ac" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	AccountingBasis          AccountingBasisType          `json:"accountingBasis"          bun:"accounting_basis,type:accounting_basis_enum,notnull,default:'Accrual'"`
	RevenueRecognitionPolicy RevenueRecognitionPolicyType `json:"revenueRecognitionPolicy" bun:"revenue_recognition_policy,type:revenue_recognition_policy_enum,notnull,default:'OnInvoicePost'"`
	ExpenseRecognitionPolicy ExpenseRecognitionPolicyType `json:"expenseRecognitionPolicy" bun:"expense_recognition_policy,type:expense_recognition_policy_enum,notnull,default:'OnVendorBillPost'"`

	JournalPostingMode       JournalPostingModeType    `json:"journalPostingMode"       bun:"journal_posting_mode,type:journal_posting_mode_enum,notnull,default:'Manual'"`
	AutoPostSourceEvents     []JournalSourceEventType  `json:"autoPostSourceEvents"     bun:"auto_post_source_events,type:journal_source_event_enum[],notnull,default:'{}'"`
	ManualJournalEntryPolicy ManualJournalEntryPolicy  `json:"manualJournalEntryPolicy" bun:"manual_journal_entry_policy,type:manual_journal_entry_policy_enum,notnull,default:'AdjustmentOnly'"`
	RequireManualJEApproval  bool                      `json:"requireManualJeApproval"  bun:"require_manual_je_approval,type:BOOLEAN,notnull,default:true"`
	JournalReversalPolicy    JournalReversalPolicyType `json:"journalReversalPolicy"    bun:"journal_reversal_policy,type:journal_reversal_policy_enum,notnull,default:'NextOpenPeriod'"`

	PeriodCloseMode              PeriodCloseModeType       `json:"periodCloseMode"              bun:"period_close_mode,type:period_close_mode_enum,notnull,default:'ManualOnly'"`
	RequirePeriodCloseApproval   bool                      `json:"requirePeriodCloseApproval"   bun:"require_period_close_approval,type:BOOLEAN,notnull,default:true"`
	LockedPeriodPostingPolicy    LockedPeriodPostingPolicy `json:"lockedPeriodPostingPolicy"    bun:"locked_period_posting_policy,type:locked_period_posting_policy_enum,notnull,default:'BlockSubledgerAllowManualJe'"`
	ClosedPeriodPostingPolicy    ClosedPeriodPostingPolicy `json:"closedPeriodPostingPolicy"    bun:"closed_period_posting_policy,type:closed_period_posting_policy_enum,notnull,default:'RequireReopen'"`
	RequireReconciliationToClose bool                      `json:"requireReconciliationToClose" bun:"require_reconciliation_to_close,type:BOOLEAN,notnull,default:false"`

	ReconciliationMode              ReconciliationModeType `json:"reconciliationMode"              bun:"reconciliation_mode,type:reconciliation_mode_enum,notnull,default:'Disabled'"`
	ReconciliationToleranceAmount   decimal.Decimal        `json:"reconciliationToleranceAmount"   bun:"reconciliation_tolerance_amount,type:NUMERIC(19,4),notnull,default:0.0000"`
	NotifyOnReconciliationException bool                   `json:"notifyOnReconciliationException" bun:"notify_on_reconciliation_exception,type:BOOLEAN,notnull,default:true"`

	CurrencyMode               CurrencyModeType         `json:"currencyMode"               bun:"currency_mode,type:currency_mode_enum,notnull,default:'SingleCurrency'"`
	FunctionalCurrencyCode     string                   `json:"functionalCurrencyCode"     bun:"functional_currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	ExchangeRateDatePolicy     ExchangeRateDatePolicy   `json:"exchangeRateDatePolicy"     bun:"exchange_rate_date_policy,type:exchange_rate_date_policy_enum,notnull,default:'DocumentDate'"`
	ExchangeRateOverridePolicy ExchangeRateOverrideType `json:"exchangeRateOverridePolicy" bun:"exchange_rate_override_policy,type:exchange_rate_override_policy_enum,notnull,default:'RequireApproval'"`

	DefaultRevenueAccountID          pulid.ID `json:"defaultRevenueAccountId"          bun:"default_revenue_account_id,type:VARCHAR(100),nullzero"`
	DefaultExpenseAccountID          pulid.ID `json:"defaultExpenseAccountId"          bun:"default_expense_account_id,type:VARCHAR(100),nullzero"`
	DefaultARAccountID               pulid.ID `json:"defaultArAccountId"               bun:"default_ar_account_id,type:VARCHAR(100),nullzero"`
	DefaultAPAccountID               pulid.ID `json:"defaultApAccountId"               bun:"default_ap_account_id,type:VARCHAR(100),nullzero"`
	DefaultTaxLiabilityAccountID     pulid.ID `json:"defaultTaxLiabilityAccountId"     bun:"default_tax_liability_account_id,type:VARCHAR(100),nullzero"`
	DefaultWriteOffAccountID         pulid.ID `json:"defaultWriteOffAccountId"         bun:"default_write_off_account_id,type:VARCHAR(100),nullzero"`
	DefaultRetainedEarningsAccountID pulid.ID `json:"defaultRetainedEarningsAccountId" bun:"default_retained_earnings_account_id,type:VARCHAR(100),nullzero"`
	RealizedFXGainAccountID          pulid.ID `json:"realizedFxGainAccountId"          bun:"realized_fx_gain_account_id,type:VARCHAR(100),nullzero"`
	RealizedFXLossAccountID          pulid.ID `json:"realizedFxLossAccountId"          bun:"realized_fx_loss_account_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ac *AccountingControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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

func (ac *AccountingControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ac.ID.IsNil() {
			ac.ID = pulid.MustNew("ac_")
		}
		ac.CreatedAt = now
	case *bun.UpdateQuery:
		ac.UpdatedAt = now
	}

	return nil
}
