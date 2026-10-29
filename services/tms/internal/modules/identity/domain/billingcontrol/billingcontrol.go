package billingcontrol

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*BillingControl)(nil)
)

type BillingControl struct {
	keel.Table `keel:"billing_controls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:bc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	DefaultPaymentTerm                    PaymentTerm                    `json:"defaultPaymentTerm"                    keel:"default_payment_term"`
	DefaultInvoiceTerms                   string                         `json:"defaultInvoiceTerms"                   keel:"default_invoice_terms,nullzero"`
	DefaultInvoiceFooter                  string                         `json:"defaultInvoiceFooter"                  keel:"default_invoice_footer,nullzero"`
	ShowDueDateOnInvoice                  bool                           `json:"showDueDateOnInvoice"                  keel:"show_due_date_on_invoice"`
	ShowBalanceDueOnInvoice               bool                           `json:"showBalanceDueOnInvoice"               keel:"show_balance_due_on_invoice"`
	ReadyToBillAssignmentMode             ReadyToBillAssignmentMode      `json:"readyToBillAssignmentMode"             keel:"ready_to_bill_assignment_mode"`
	BillingQueueTransferMode              BillingQueueTransferMode       `json:"billingQueueTransferMode"              keel:"billing_queue_transfer_mode"`
	BillingQueueTransferSchedule          TransferSchedule               `json:"billingQueueTransferSchedule"          keel:"billing_queue_transfer_schedule,nullzero"`
	BillingQueueTransferBatchSize         int                            `json:"billingQueueTransferBatchSize"         keel:"billing_queue_transfer_batch_size,nullzero"`
	InvoiceDraftCreationMode              InvoiceDraftCreationMode       `json:"invoiceDraftCreationMode"              keel:"invoice_draft_creation_mode"`
	InvoicePostingMode                    InvoicePostingMode             `json:"invoicePostingMode"                    keel:"invoice_posting_mode"`
	AutoInvoiceBatchSize                  int                            `json:"autoInvoiceBatchSize"                  keel:"auto_invoice_batch_size,nullzero"`
	NotifyOnAutoInvoiceCreation           bool                           `json:"notifyOnAutoInvoiceCreation"           keel:"notify_on_auto_invoice_creation"`
	ShipmentBillingRequirementEnforcement tenant.EnforcementLevel        `json:"shipmentBillingRequirementEnforcement" keel:"shipment_billing_requirement_enforcement"`
	RateValidationEnforcement             tenant.EnforcementLevel        `json:"rateValidationEnforcement"             keel:"rate_validation_enforcement"`
	BillingExceptionDisposition           BillingExceptionDisposition    `json:"billingExceptionDisposition"           keel:"billing_exception_disposition"`
	NotifyOnBillingExceptions             bool                           `json:"notifyOnBillingExceptions"             keel:"notify_on_billing_exceptions"`
	RateVarianceTolerancePercent          decimal.Decimal                `json:"rateVarianceTolerancePercent"          keel:"rate_variance_tolerance_percent"`
	RateVarianceAutoResolutionMode        RateVarianceAutoResolutionMode `json:"rateVarianceAutoResolutionMode"        keel:"rate_variance_auto_resolution_mode"`

	// UnratedShipmentDisposition decides what happens when no rate agreement
	// covers a shipment's lane. It defaults to falling back to the formula
	// template on the shipment, which is exactly how rating worked before
	// agreements existed, so an organization that has not written a contract
	// yet sees no change at all.
	UnratedShipmentDisposition UnratedShipmentDisposition `json:"unratedShipmentDisposition" keel:"unrated_shipment_disposition"`
	FallbackFormulaTemplateID  *pulid.ID                  `json:"fallbackFormulaTemplateId"  keel:"fallback_formula_template_id"`
	RequireRateOverrideReason  bool                       `json:"requireRateOverrideReason"  keel:"require_rate_override_reason"`
	EnforceMarginFloor         bool                       `json:"enforceMarginFloor"         keel:"enforce_margin_floor"`

	// LateChargeAssessmentMode says whether the nightly run previews or raises
	// late-charge debit memos; the rate and grace live on each customer.
	LateChargeAssessmentMode LateChargeAssessmentMode `json:"lateChargeAssessmentMode" keel:"late_charge_assessment_mode"`
	// LateChargeMinimumAmount is the smallest total a run will bill a customer.
	LateChargeMinimumAmount decimal.Decimal `json:"lateChargeMinimumAmount"  keel:"late_charge_minimum_amount"`

	// PaymentRemindersEnabled emails customers about their overdue invoices on
	// the days past due in PaymentReminderDays, one reminder per step.
	PaymentRemindersEnabled bool    `json:"paymentRemindersEnabled" keel:"payment_reminders_enabled"`
	PaymentReminderDays     []int32 `json:"paymentReminderDays"     keel:"payment_reminder_days"`

	// MonthlyStatementsEnabled emails every customer with a balance or
	// activity a statement of the month just closed, on MonthlyStatementDay.
	MonthlyStatementsEnabled bool  `json:"monthlyStatementsEnabled" keel:"monthly_statements_enabled"`
	MonthlyStatementDay      int16 `json:"monthlyStatementDay"      keel:"monthly_statement_day"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (bc *BillingControl) BeforeUpdate(_ context.Context) error {
	if len(bc.PaymentReminderDays) == 0 {
		bc.PaymentReminderDays = slices.Clone(DefaultPaymentReminderDays)
	}
	if bc.MonthlyStatementDay == 0 {
		bc.MonthlyStatementDay = 1
	}

	return nil
}

func (bc *BillingControl) BeforeCreate(_ context.Context) error {
	if len(bc.PaymentReminderDays) == 0 {
		bc.PaymentReminderDays = slices.Clone(DefaultPaymentReminderDays)
	}
	if bc.MonthlyStatementDay == 0 {
		bc.MonthlyStatementDay = 1
	}
	if bc.DefaultPaymentTerm == "" {
		bc.DefaultPaymentTerm = PaymentTermNet30
	}
	if bc.ReadyToBillAssignmentMode == "" {
		bc.ReadyToBillAssignmentMode = ReadyToBillAssignmentModeManualOnly
	}
	if bc.BillingQueueTransferMode == "" {
		bc.BillingQueueTransferMode = BillingQueueTransferModeManualOnly
	}
	if bc.BillingQueueTransferSchedule == "" {
		bc.BillingQueueTransferSchedule = TransferScheduleContinuous
	}
	if bc.InvoiceDraftCreationMode == "" {
		bc.InvoiceDraftCreationMode = InvoiceDraftCreationModeManualOnly
	}
	if bc.InvoicePostingMode == "" {
		bc.InvoicePostingMode = InvoicePostingModeManualReviewRequired
	}
	if bc.ShipmentBillingRequirementEnforcement == "" {
		bc.ShipmentBillingRequirementEnforcement = tenant.EnforcementLevelBlock
	}
	if bc.RateValidationEnforcement == "" {
		bc.RateValidationEnforcement = tenant.EnforcementLevelRequireReview
	}
	if bc.BillingExceptionDisposition == "" {
		bc.BillingExceptionDisposition = BillingExceptionDispositionRouteToBillingReview
	}
	if bc.RateVarianceAutoResolutionMode == "" {
		bc.RateVarianceAutoResolutionMode = RateVarianceAutoResolutionModeDisabled
	}
	if bc.UnratedShipmentDisposition == "" {
		bc.UnratedShipmentDisposition = UnratedShipmentDispositionFallbackFormulaTemplate
	}
	if bc.LateChargeAssessmentMode == "" {
		bc.LateChargeAssessmentMode = LateChargeAssessmentModeDisabled
	}

	return nil
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

func (bc *BillingControl) BeforeSave(_ context.Context) error {
	if len(bc.PaymentReminderDays) == 0 {
		bc.PaymentReminderDays = slices.Clone(DefaultPaymentReminderDays)
	}
	if bc.MonthlyStatementDay == 0 {
		bc.MonthlyStatementDay = 1
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
