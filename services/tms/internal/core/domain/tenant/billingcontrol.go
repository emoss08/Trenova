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
	_ bun.BeforeAppendModelHook          = (*BillingControl)(nil)
	_ validationframework.TenantedEntity = (*BillingControl)(nil)
)

type BillingControl struct {
	bun.BaseModel `json:"-" bun:"table:billing_controls,alias:bc"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	DefaultPaymentTerm      PaymentTerm `json:"defaultPaymentTerm"      bun:"default_payment_term,type:payment_term_enum,notnull,default:'Net30'"`
	DefaultInvoiceTerms     string      `json:"defaultInvoiceTerms"     bun:"default_invoice_terms,type:TEXT,nullzero"`
	DefaultInvoiceFooter    string      `json:"defaultInvoiceFooter"    bun:"default_invoice_footer,type:TEXT,nullzero"`
	ShowDueDateOnInvoice    bool        `json:"showDueDateOnInvoice"    bun:"show_due_date_on_invoice,type:BOOLEAN,notnull,default:true"`
	ShowBalanceDueOnInvoice bool        `json:"showBalanceDueOnInvoice" bun:"show_balance_due_on_invoice,type:BOOLEAN,notnull,default:true"`

	ReadyToBillAssignmentMode     ReadyToBillAssignmentMode `json:"readyToBillAssignmentMode"     bun:"ready_to_bill_assignment_mode,type:ready_to_bill_assignment_mode_enum,notnull,default:'ManualOnly'"`
	BillingQueueTransferMode      BillingQueueTransferMode  `json:"billingQueueTransferMode"      bun:"billing_queue_transfer_mode,type:billing_queue_transfer_mode_enum,notnull,default:'ManualOnly'"`
	BillingQueueTransferSchedule  TransferSchedule          `json:"billingQueueTransferSchedule"  bun:"billing_queue_transfer_schedule,type:transfer_schedule_enum,nullzero"`
	BillingQueueTransferBatchSize int                       `json:"billingQueueTransferBatchSize" bun:"billing_queue_transfer_batch_size,type:INTEGER,nullzero"`

	InvoiceDraftCreationMode    InvoiceDraftCreationMode `json:"invoiceDraftCreationMode"    bun:"invoice_draft_creation_mode,type:invoice_draft_creation_mode_enum,notnull,default:'ManualOnly'"`
	InvoicePostingMode          InvoicePostingMode       `json:"invoicePostingMode"          bun:"invoice_posting_mode,type:invoice_posting_mode_enum,notnull,default:'ManualReviewRequired'"`
	AutoInvoiceBatchSize        int                      `json:"autoInvoiceBatchSize"        bun:"auto_invoice_batch_size,type:INTEGER,nullzero"`
	NotifyOnAutoInvoiceCreation bool                     `json:"notifyOnAutoInvoiceCreation" bun:"notify_on_auto_invoice_creation,type:BOOLEAN,notnull,default:false"`

	ShipmentBillingRequirementEnforcement EnforcementLevel            `json:"shipmentBillingRequirementEnforcement" bun:"shipment_billing_requirement_enforcement,type:enforcement_level_enum,notnull,default:'Block'"`
	RateValidationEnforcement             EnforcementLevel            `json:"rateValidationEnforcement"             bun:"rate_validation_enforcement,type:enforcement_level_enum,notnull,default:'RequireReview'"`
	BillingExceptionDisposition           BillingExceptionDisposition `json:"billingExceptionDisposition"           bun:"billing_exception_disposition,type:billing_exception_disposition_enum,notnull,default:'RouteToBillingReview'"`
	NotifyOnBillingExceptions             bool                        `json:"notifyOnBillingExceptions"             bun:"notify_on_billing_exceptions,type:BOOLEAN,notnull,default:true"`

	RateVarianceTolerancePercent   decimal.Decimal                `json:"rateVarianceTolerancePercent"   bun:"rate_variance_tolerance_percent,type:NUMERIC(9,6),notnull,default:0.000000"`
	RateVarianceAutoResolutionMode RateVarianceAutoResolutionMode `json:"rateVarianceAutoResolutionMode" bun:"rate_variance_auto_resolution_mode,type:rate_variance_auto_resolution_mode_enum,notnull,default:'Disabled'"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (bc *BillingControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		bc,
		validation.Field(&bc.DefaultPaymentTerm, validation.Required),
		validation.Field(&bc.ReadyToBillAssignmentMode, validation.Required),
		validation.Field(&bc.BillingQueueTransferMode, validation.Required),
		validation.Field(&bc.InvoiceDraftCreationMode, validation.Required),
		validation.Field(&bc.InvoicePostingMode, validation.Required),
		validation.Field(&bc.ShipmentBillingRequirementEnforcement, validation.Required),
		validation.Field(&bc.RateValidationEnforcement, validation.Required),
		validation.Field(&bc.BillingExceptionDisposition, validation.Required),
		validation.Field(&bc.RateVarianceAutoResolutionMode, validation.Required),
		validation.Field(
			&bc.RateVarianceTolerancePercent,
			validation.Required,
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (bc *BillingControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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
