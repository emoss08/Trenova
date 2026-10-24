package tenant

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*BillingControl)(nil)
	_ validationframework.TenantedEntity = (*BillingControl)(nil)
)

type BillingControl struct {
	bun.BaseModel `json:"-" bun:"table:billing_controls,alias:bc"`
	keel.Table    `keel:"billing_controls"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:bc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`

	DefaultPaymentTerm                    PaymentTerm                    `json:"defaultPaymentTerm"                    bun:"default_payment_term,type:payment_term_enum,notnull,default:'Net30'" keel:"default_payment_term"`
	DefaultInvoiceTerms                   string                         `json:"defaultInvoiceTerms"                   bun:"default_invoice_terms,type:TEXT,nullzero" keel:"default_invoice_terms,nullzero"`
	DefaultInvoiceFooter                  string                         `json:"defaultInvoiceFooter"                  bun:"default_invoice_footer,type:TEXT,nullzero" keel:"default_invoice_footer,nullzero"`
	ShowDueDateOnInvoice                  bool                           `json:"showDueDateOnInvoice"                  bun:"show_due_date_on_invoice,type:BOOLEAN,notnull" keel:"show_due_date_on_invoice"`
	ShowBalanceDueOnInvoice               bool                           `json:"showBalanceDueOnInvoice"               bun:"show_balance_due_on_invoice,type:BOOLEAN,notnull" keel:"show_balance_due_on_invoice"`
	ReadyToBillAssignmentMode             ReadyToBillAssignmentMode      `json:"readyToBillAssignmentMode"             bun:"ready_to_bill_assignment_mode,type:ready_to_bill_assignment_mode_enum,notnull,default:'ManualOnly'" keel:"ready_to_bill_assignment_mode"`
	BillingQueueTransferMode              BillingQueueTransferMode       `json:"billingQueueTransferMode"              bun:"billing_queue_transfer_mode,type:billing_queue_transfer_mode_enum,notnull,default:'ManualOnly'" keel:"billing_queue_transfer_mode"`
	BillingQueueTransferSchedule          TransferSchedule               `json:"billingQueueTransferSchedule"          bun:"billing_queue_transfer_schedule,type:transfer_schedule_enum,default:'Continuous'" keel:"billing_queue_transfer_schedule,nullzero"`
	BillingQueueTransferBatchSize         int                            `json:"billingQueueTransferBatchSize"         bun:"billing_queue_transfer_batch_size,type:INTEGER,nullzero" keel:"billing_queue_transfer_batch_size,nullzero"`
	InvoiceDraftCreationMode              InvoiceDraftCreationMode       `json:"invoiceDraftCreationMode"              bun:"invoice_draft_creation_mode,type:invoice_draft_creation_mode_enum,notnull,default:'ManualOnly'" keel:"invoice_draft_creation_mode"`
	InvoicePostingMode                    InvoicePostingMode             `json:"invoicePostingMode"                    bun:"invoice_posting_mode,type:invoice_posting_mode_enum,notnull,default:'ManualReviewRequired'" keel:"invoice_posting_mode"`
	AutoInvoiceBatchSize                  int                            `json:"autoInvoiceBatchSize"                  bun:"auto_invoice_batch_size,type:INTEGER,nullzero" keel:"auto_invoice_batch_size,nullzero"`
	NotifyOnAutoInvoiceCreation           bool                           `json:"notifyOnAutoInvoiceCreation"           bun:"notify_on_auto_invoice_creation,type:BOOLEAN,notnull" keel:"notify_on_auto_invoice_creation"`
	ShipmentBillingRequirementEnforcement EnforcementLevel               `json:"shipmentBillingRequirementEnforcement" bun:"shipment_billing_requirement_enforcement,type:enforcement_level_enum,notnull,default:'Block'" keel:"shipment_billing_requirement_enforcement"`
	RateValidationEnforcement             EnforcementLevel               `json:"rateValidationEnforcement"             bun:"rate_validation_enforcement,type:enforcement_level_enum,notnull,default:'RequireReview'" keel:"rate_validation_enforcement"`
	BillingExceptionDisposition           BillingExceptionDisposition    `json:"billingExceptionDisposition"           bun:"billing_exception_disposition,type:billing_exception_disposition_enum,notnull,default:'RouteToBillingReview'" keel:"billing_exception_disposition"`
	NotifyOnBillingExceptions             bool                           `json:"notifyOnBillingExceptions"             bun:"notify_on_billing_exceptions,type:BOOLEAN,notnull" keel:"notify_on_billing_exceptions"`
	RateVarianceTolerancePercent          decimal.Decimal                `json:"rateVarianceTolerancePercent"          bun:"rate_variance_tolerance_percent,type:NUMERIC(9,6),notnull,default:0.000000" keel:"rate_variance_tolerance_percent"`
	RateVarianceAutoResolutionMode        RateVarianceAutoResolutionMode `json:"rateVarianceAutoResolutionMode"        bun:"rate_variance_auto_resolution_mode,type:rate_variance_auto_resolution_mode_enum,notnull,default:'Disabled'" keel:"rate_variance_auto_resolution_mode"`

	// UnratedShipmentDisposition decides what happens when no rate agreement
	// covers a shipment's lane. It defaults to falling back to the formula
	// template on the shipment, which is exactly how rating worked before
	// agreements existed, so an organization that has not written a contract
	// yet sees no change at all.
	UnratedShipmentDisposition UnratedShipmentDisposition `json:"unratedShipmentDisposition" bun:"unrated_shipment_disposition,type:unrated_shipment_disposition_enum,notnull,default:'FallbackFormulaTemplate'" keel:"unrated_shipment_disposition"`
	FallbackFormulaTemplateID  *pulid.ID                  `json:"fallbackFormulaTemplateId"  bun:"fallback_formula_template_id,type:VARCHAR(100),nullzero" keel:"fallback_formula_template_id"`
	RequireRateOverrideReason  bool                       `json:"requireRateOverrideReason"  bun:"require_rate_override_reason,type:BOOLEAN,notnull" keel:"require_rate_override_reason"`
	EnforceMarginFloor         bool                       `json:"enforceMarginFloor"         bun:"enforce_margin_floor,type:BOOLEAN,notnull,default:false" keel:"enforce_margin_floor"`

	// LateChargeAssessmentMode says whether the nightly run previews or raises
	// late-charge debit memos; the rate and grace live on each customer.
	LateChargeAssessmentMode LateChargeAssessmentMode `json:"lateChargeAssessmentMode" bun:"late_charge_assessment_mode,type:late_charge_assessment_mode_enum,notnull,default:'Disabled'" keel:"late_charge_assessment_mode"`
	// LateChargeMinimumAmount is the smallest total a run will bill a customer.
	LateChargeMinimumAmount decimal.Decimal `json:"lateChargeMinimumAmount" bun:"late_charge_minimum_amount,type:NUMERIC(19,4),notnull,default:0" keel:"late_charge_minimum_amount"`

	// PaymentRemindersEnabled emails customers about their overdue invoices on
	// the days past due in PaymentReminderDays, one reminder per step.
	PaymentRemindersEnabled bool    `json:"paymentRemindersEnabled" bun:"payment_reminders_enabled,type:BOOLEAN,notnull,default:false" keel:"payment_reminders_enabled"`
	PaymentReminderDays     []int32 `json:"paymentReminderDays"     bun:"payment_reminder_days,type:INTEGER[],array,notnull" keel:"payment_reminder_days"`

	// MonthlyStatementsEnabled emails every customer with a balance or
	// activity a statement of the month just closed, on MonthlyStatementDay.
	MonthlyStatementsEnabled bool  `json:"monthlyStatementsEnabled" bun:"monthly_statements_enabled,type:BOOLEAN,notnull,default:false" keel:"monthly_statements_enabled"`
	MonthlyStatementDay      int16 `json:"monthlyStatementDay"      bun:"monthly_statement_day,type:SMALLINT,notnull,default:1" keel:"monthly_statement_day"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// MaxPaymentReminderSteps bounds how many reminders one invoice can draw.
const MaxPaymentReminderSteps = 6

// DefaultPaymentReminderDays is the schedule an organization starts from: a
// day late, two weeks late, a month late.
var DefaultPaymentReminderDays = []int32{1, 15, 30}

// MaxMonthlyStatementDay is the latest day of the month statements can go
// out on, so the day exists in every month.
const MaxMonthlyStatementDay = 28

func (bc *BillingControl) Validate(multiErr *errortypes.MultiError) {
	bc.validatePaymentReminderDays(multiErr)
	if bc.MonthlyStatementDay < 0 || bc.MonthlyStatementDay > MaxMonthlyStatementDay {
		multiErr.Add("monthlyStatementDay", errortypes.ErrInvalid,
			"Choose a day between 1 and 28, so it falls in every month")
	}
	if bc.LateChargeMinimumAmount.LessThan(decimal.Zero) {
		multiErr.Add(
			"lateChargeMinimumAmount",
			errortypes.ErrInvalid,
			"Late charge minimum cannot be negative",
		)
	}
	multiErr.AddOzzoError(validation.ValidateStruct(
		bc,
		validation.Field(&bc.DefaultPaymentTerm, validation.Required),
		validation.Field(&bc.ReadyToBillAssignmentMode, validation.Required),
		validation.Field(&bc.BillingQueueTransferMode, validation.Required),
		validation.Field(&bc.InvoiceDraftCreationMode, validation.Required),
		validation.Field(&bc.InvoicePostingMode, validation.Required),
		validation.Field(&bc.ShipmentBillingRequirementEnforcement, validation.Required),
		validation.Field(&bc.RateValidationEnforcement, validation.Required),
		validation.Field(&bc.BillingExceptionDisposition, validation.Required),
		validation.Field(
			&bc.LateChargeAssessmentMode,
			validation.When(
				bc.LateChargeAssessmentMode != "",
				domainvalidation.ValidEnum[LateChargeAssessmentMode](
					"Invalid late charge assessment mode",
				),
			),
		),
		validation.Field(&bc.RateVarianceAutoResolutionMode, validation.Required),
		validation.Field(
			&bc.RateVarianceTolerancePercent,
			validation.Required,
		),
	))
}

// validatePaymentReminderDays requires a schedule of distinct days past due,
// earliest first, when reminders are on.
func (bc *BillingControl) validatePaymentReminderDays(multiErr *errortypes.MultiError) {
	if !bc.PaymentRemindersEnabled {
		return
	}
	days := bc.PaymentReminderDays
	switch {
	case len(days) == 0:
		multiErr.Add("paymentReminderDays", errortypes.ErrRequired,
			"Choose at least one day past due to send a reminder on")
		return
	case len(days) > MaxPaymentReminderSteps:
		multiErr.Add("paymentReminderDays", errortypes.ErrInvalid,
			"Send at most six reminders per invoice")
		return
	}
	for i, day := range days {
		if day < 1 || day > 365 {
			multiErr.Add("paymentReminderDays", errortypes.ErrInvalid,
				"Reminder days must be between 1 and 365 days past due")
			return
		}
		if i > 0 && day <= days[i-1] {
			multiErr.Add("paymentReminderDays", errortypes.ErrInvalid,
				"List reminder days from earliest to latest, each once")
			return
		}
	}
}

func (bc *BillingControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if len(bc.PaymentReminderDays) == 0 {
		bc.PaymentReminderDays = slices.Clone(DefaultPaymentReminderDays)
	}
	if bc.MonthlyStatementDay == 0 {
		bc.MonthlyStatementDay = 1
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if bc.ID.IsNil() {
			bc.ID = pulid.MustNew("bc_")
		}
		bc.CreatedAt = now
	case *bun.UpdateQuery:
		bc.UpdatedAt = now
	}

	return nil
}

func (bc *BillingControl) GetID() pulid.ID {
	return bc.ID
}

func (bc *BillingControl) GetTableName() string {
	return "billing_controls"
}

func (bc *BillingControl) GetOrganizationID() pulid.ID {
	return bc.OrganizationID
}

func (bc *BillingControl) GetBusinessUnitID() pulid.ID {
	return bc.BusinessUnitID
}
