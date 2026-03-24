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

	// --- Accounting Method ---
	// Top-level method that constrains which revenue/expense recognition options are valid.
	AccountingMethod AccountingMethodType `json:"accountingMethod" bun:"accounting_method,type:accounting_method_enum,notnull,default:'Accrual'"`

	// --- Default GL Accounts ---
	DefaultRevenueAccountID          pulid.ID `json:"defaultRevenueAccountId"          bun:"default_revenue_account_id,type:VARCHAR(100),nullzero"`
	DefaultExpenseAccountID          pulid.ID `json:"defaultExpenseAccountId"          bun:"default_expense_account_id,type:VARCHAR(100),nullzero"`
	DefaultARAccountID               pulid.ID `json:"defaultArAccountId"               bun:"default_ar_account_id,type:VARCHAR(100),nullzero"`
	DefaultAPAccountID               pulid.ID `json:"defaultApAccountId"               bun:"default_ap_account_id,type:VARCHAR(100),nullzero"`
	DefaultTaxAccountID              pulid.ID `json:"defaultTaxAccountId"              bun:"default_tax_account_id,type:VARCHAR(100),nullzero"`
	DefaultDeferredRevenueAccountID  pulid.ID `json:"defaultDeferredRevenueAccountId"  bun:"default_deferred_revenue_account_id,type:VARCHAR(100),nullzero"`
	DefaultCostOfServiceAccountID    pulid.ID `json:"defaultCostOfServiceAccountId"    bun:"default_cost_of_service_account_id,type:VARCHAR(100),nullzero"`
	DefaultRetainedEarningsAccountID pulid.ID `json:"defaultRetainedEarningsAccountId" bun:"default_retained_earnings_account_id,type:VARCHAR(100),nullzero"`

	// --- Journal Entry Automation ---
	AutoCreateJournalEntries bool                       `json:"autoCreateJournalEntries" bun:"auto_create_journal_entries,type:BOOLEAN,notnull,default:false"`
	JournalEntryCriteria     []JournalEntryCriteriaType `json:"journalEntryCriteria"     bun:"journal_entry_criteria,type:jsonb,notnull,default:'[]'"`

	// --- Journal Entry Controls ---
	RestrictManualJournalEntries bool `json:"restrictManualJournalEntries" bun:"restrict_manual_journal_entries,type:BOOLEAN,notnull,default:false"`
	RequireJournalEntryApproval  bool `json:"requireJournalEntryApproval"  bun:"require_journal_entry_approval,type:BOOLEAN,notnull,default:true"`
	EnableJournalEntryReversal   bool `json:"enableJournalEntryReversal"   bun:"enable_journal_entry_reversal,type:BOOLEAN,notnull,default:true"`

	// --- Period Controls ---
	AllowPostingToClosedPeriods bool `json:"allowPostingToClosedPeriods" bun:"allow_posting_to_closed_periods,type:BOOLEAN,notnull,default:false"`
	RequirePeriodEndApproval    bool `json:"requirePeriodEndApproval"    bun:"require_period_end_approval,type:BOOLEAN,notnull,default:true"`
	AutoClosePeriods            bool `json:"autoClosePeriods"            bun:"auto_close_periods,type:BOOLEAN,notnull,default:false"`

	// --- Reconciliation Settings ---
	EnableReconciliation              bool                `json:"enableReconciliation"              bun:"enable_reconciliation,type:BOOLEAN,notnull,default:false"`
	ReconciliationThreshold           decimal.Decimal     `json:"reconciliationThreshold"           bun:"reconciliation_threshold,type:NUMERIC(19,4),notnull,default:0.0050"`
	ReconciliationThresholdAction     ThresholdActionType `json:"reconciliationThresholdAction"     bun:"reconciliation_threshold_action,type:threshold_action_enum,notnull,default:'Warn'"`
	HaltOnPendingReconciliation       bool                `json:"haltOnPendingReconciliation"       bun:"halt_on_pending_reconciliation,type:BOOLEAN,notnull,default:false"`
	EnableReconciliationNotifications bool                `json:"enableReconciliationNotifications" bun:"enable_reconciliation_notifications,type:BOOLEAN,notnull,default:true"`

	// --- Revenue Recognition ---
	RevenueRecognitionMethod RevenueRecognitionType `json:"revenueRecognitionMethod" bun:"revenue_recognition_method,type:revenue_recognition_enum,notnull,default:'OnDelivery'"`
	DeferRevenueUntilPaid    bool                   `json:"deferRevenueUntilPaid"    bun:"defer_revenue_until_paid,type:BOOLEAN,notnull,default:false"`

	// --- Expense Recognition ---
	ExpenseRecognitionMethod ExpenseRecognitionType `json:"expenseRecognitionMethod" bun:"expense_recognition_method,type:expense_recognition_enum,notnull,default:'OnIncurrence'"`
	AccrueExpenses           bool                   `json:"accrueExpenses"           bun:"accrue_expenses,type:BOOLEAN,notnull,default:true"`

	// --- Tax Settings ---
	EnableAutomaticTaxCalculation bool `json:"enableAutomaticTaxCalculation" bun:"enable_automatic_tax_calculation,type:BOOLEAN,notnull,default:true"`

	// --- Audit & Compliance ---
	RequireDocumentAttachment bool `json:"requireDocumentAttachment" bun:"require_document_attachment,type:BOOLEAN,notnull,default:false"`
	RetainDeletedEntries      bool `json:"retainDeletedEntries"      bun:"retain_deleted_entries,type:BOOLEAN,notnull,default:true"`

	// --- Multi-Currency ---
	EnableMultiCurrency   bool     `json:"enableMultiCurrency"   bun:"enable_multi_currency,type:BOOLEAN,notnull,default:false"`
	DefaultCurrencyCode   string   `json:"defaultCurrencyCode"   bun:"default_currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	CurrencyGainAccountID pulid.ID `json:"currencyGainAccountId" bun:"currency_gain_account_id,type:VARCHAR(100),nullzero"`
	CurrencyLossAccountID pulid.ID `json:"currencyLossAccountId" bun:"currency_loss_account_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// --- Relationships ---
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ac *AccountingControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		ac,
		// --- Accounting Method ---
		validation.Field(&ac.AccountingMethod,
			validation.Required.Error("Accounting method is required"),
			validation.In(
				AccountingMethodAccrual,
				AccountingMethodCash,
				AccountingMethodHybrid,
			).Error("Accounting method must be a valid type"),
		),

		// --- Revenue Recognition ---
		validation.Field(&ac.RevenueRecognitionMethod,
			validation.Required.Error("Revenue recognition method is required"),
			validation.In(
				RevenueRecognitionOnDelivery,
				RevenueRecognitionOnBilling,
				RevenueRecognitionOnPayment,
				RevenueRecognitionOnPickup,
			).Error("Revenue recognition method must be a valid type"),
		),

		// --- Expense Recognition ---
		validation.Field(&ac.ExpenseRecognitionMethod,
			validation.Required.Error("Expense recognition method is required"),
			validation.In(
				ExpenseRecognitionOnIncurrence,
				ExpenseRecognitionOnPayment,
				ExpenseRecognitionOnAccrual,
			).Error("Expense recognition method must be a valid type"),
		),

		// --- Journal Entry Criteria ---
		validation.Field(&ac.JournalEntryCriteria,
			validation.When(
				ac.AutoCreateJournalEntries,
				validation.Required.Error(
					"At least one journal entry criteria is required when auto-creating journal entries",
				),
			),
			validation.Each(
				validation.In(
					JournalEntryCriteriaInvoicePosted,
					JournalEntryCriteriaBillPosted,
					JournalEntryCriteriaPaymentReceived,
					JournalEntryCriteriaPaymentMade,
					JournalEntryCriteriaDeliveryComplete,
					JournalEntryCriteriaShipmentDispatched,
				).Error("Journal entry criteria must be a valid type"),
			),
		),

		// --- GL Accounts (conditional on auto journal entries) ---
		validation.Field(&ac.DefaultRevenueAccountID,
			validation.When(
				ac.AutoCreateJournalEntries,
				validation.Required.Error(
					"Default revenue account is required when auto-creating journal entries",
				),
			),
		),
		validation.Field(&ac.DefaultExpenseAccountID,
			validation.When(
				ac.AutoCreateJournalEntries,
				validation.Required.Error(
					"Default expense account is required when auto-creating journal entries",
				),
			),
		),
		validation.Field(&ac.DefaultARAccountID,
			validation.When(
				ac.AutoCreateJournalEntries,
				validation.Required.Error(
					"Default AR account is required when auto-creating journal entries",
				),
			),
		),
		validation.Field(&ac.DefaultAPAccountID,
			validation.When(
				ac.AutoCreateJournalEntries,
				validation.Required.Error(
					"Default AP account is required when auto-creating journal entries",
				),
			),
		),

		// --- Deferred Revenue Account ---
		validation.Field(&ac.DefaultDeferredRevenueAccountID,
			validation.When(
				ac.DeferRevenueUntilPaid,
				validation.Required.Error(
					"Default deferred revenue account is required when defer revenue until paid is enabled",
				),
			),
		),

		// --- Retained Earnings Account ---
		validation.Field(&ac.DefaultRetainedEarningsAccountID,
			validation.When(
				ac.AutoClosePeriods,
				validation.Required.Error(
					"Default retained earnings account is required when automatic period close is enabled",
				),
			),
		),

		// --- Reconciliation ---
		validation.Field(&ac.ReconciliationThresholdAction,
			validation.Required.Error("Reconciliation threshold action is required"),
			validation.In(
				ThresholdActionWarn,
				ThresholdActionBlock,
				ThresholdActionNotify,
			).Error("Threshold action must be a valid type"),
		),

		// --- Currency ---
		validation.Field(&ac.DefaultCurrencyCode,
			validation.Required.Error("Default currency code is required"),
			validation.Length(3, 3).Error("Currency code must be exactly 3 characters (ISO 4217)"),
		),
		validation.Field(&ac.CurrencyGainAccountID,
			validation.When(
				ac.EnableMultiCurrency,
				validation.Required.Error(
					"Currency gain account is required when multi-currency is enabled",
				),
			),
		),
		validation.Field(&ac.CurrencyLossAccountID,
			validation.When(
				ac.EnableMultiCurrency,
				validation.Required.Error(
					"Currency loss account is required when multi-currency is enabled",
				),
			),
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
			ac.ID = pulid.MustNew("acc_")
		}

		ac.CreatedAt = now
	case *bun.UpdateQuery:
		ac.UpdatedAt = now
	}

	return nil
}
