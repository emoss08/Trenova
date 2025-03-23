package billing

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type BillingControl struct {
	bun.BaseModel `json:"-" bun:"table:billing_controls,alias:bc"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Prefixes for invoice and credit memo numbers
	InvoiceNumberPrefix    string `json:"invoiceNumberPrefix" bun:"invoice_number_prefix,type:VARCHAR(10),notnull,default:'INV-'"`
	CreditMemoNumberPrefix string `json:"creditMemoNumberPrefix" bun:"credit_memo_number_prefix,type:VARCHAR(10),notnull,default:'CM-'"`

	// Invoice Terms
	PaymentTerm        PaymentTerm `json:"paymentTerm" bun:"payment_term,type:payment_term_enum,notnull,default:'Net30'"`
	ShowInvoiceDueDate bool        `json:"showInvoiceDueDate" bun:"show_invoice_due_date,type:BOOLEAN,notnull,default:true"`
	InvoiceTerms       string      `json:"invoiceTerms" bun:"invoice_terms,type:TEXT,nullzero"`
	InvoiceFooter      string      `json:"invoiceFooter" bun:"invoice_footer,type:TEXT,nullzero"`
	ShowAmountDue      bool        `json:"showAmountDue" bun:"show_amount_due,type:BOOLEAN,notnull,default:true"`

	// Controls for the billing process
	AutoTransfer        bool             `json:"autoTransfer" bun:"auto_transfer,type:BOOLEAN,notnull,default:true"` // * Automatically transfer shipments if they meet billing requirements
	TransferCriteria    TransferCriteria `json:"transferCriteria" bun:"transfer_criteria,type:transfer_criteria_enum,notnull,default:'ReadyAndCompleted'"`
	TransferSchedule    TransferSchedule `json:"transferSchedule" bun:"transfer_schedule,type:transfer_schedule_enum,notnull,default:'Continuous'"`
	TransferBatchSize   int              `json:"transferBatchSize" bun:"transfer_batch_size,type:INTEGER,notnull,default:100"`        // * Number of shipments to transfer at a time
	AutoMarkReadyToBill bool             `json:"autoMarkReadyToBill" bun:"auto_mark_ready_to_bill,type:BOOLEAN,notnull,default:true"` // * Automatically mark shipment as ready to bill if it meets billing requirements

	// * Enforce customer billing requirements before billing
	EnforceCustomerBillingReq bool `json:"enforceCustomerBillingReq" bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"` // * Enforce customer billing requirements before billing
	ValidateCustomerRates     bool `json:"validateCustomerRates" bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`          // * Validate customer rates before billing

	// Automated billing controls
	AutoBill                  bool             `json:"autoBill" bun:"auto_bill,type:BOOLEAN,notnull,default:true"` // * Automatically bill shipment if it meets billing requirements
	AutoBillCriteria          AutoBillCriteria `json:"autoBillCriteria" bun:"auto_bill_criteria,type:auto_bill_criteria_enum,notnull,default:'Delivered'"`
	SendAutoBillNotifications bool             `json:"sendAutoBillNotifications" bun:"send_auto_bill_notifications,type:BOOLEAN,notnull,default:true"` // * Send notifications when invoices are generated through the automated billing process
	AutoBillBatchSize         int              `json:"autoBillBatchSize" bun:"auto_bill_batch_size,type:INTEGER,notnull,default:100"`                  // * Number of shipments to bill at a time

	// Exception handling
	BillingExceptionHandling      BillingExceptionHandling `json:"billingExceptionHandling" bun:"billing_exception_handling,type:billing_exception_handling_enum,notnull,default:'Queue'"`
	RateDiscrepancyThreshold      float64                  `json:"rateDiscrepancyThreshold" bun:"rate_discrepancy_threshold,type:NUMERIC(10,2),notnull,default:5.00"` // * Percentage threshold for rate discrepancies
	AutoResolveMinorDiscrepancies bool                     `json:"autoResolveMinorDiscrepancies" bun:"auto_resolve_minor_discrepancies,type:BOOLEAN,notnull,default:true"`

	// Consolidation options
	AllowInvoiceConsolidation bool `json:"allowInvoiceConsolidation" bun:"allow_invoice_consolidation,type:BOOLEAN,notnull,default:true"` // * Allow combining multiple shipments in one invoice
	ConsolidationPeriodDays   int  `json:"consolidationPeriodDays" bun:"consolidation_period_days,type:INTEGER,notnull,default:7"`        // * Default number of days to consolidate
	GroupConsolidatedInvoices bool `json:"groupConsolidatedInvoices" bun:"group_consolidated_invoices,type:BOOLEAN,notnull,default:true"` // * Group line items by service type in consolidated invoices

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (bc *BillingControl) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, bc,
		// * Ensure invoice number prefix is populated
		validation.Field(&bc.InvoiceNumberPrefix,
			validation.Required.Error("Invoice number prefix is required"),
			validation.Length(3, 10).Error("Invoice number prefix must be between 3 and 10 characters"),
		),

		// * Ensure credit memo number prefix is populated
		validation.Field(&bc.CreditMemoNumberPrefix,
			validation.Required.Error("Credit memo number prefix is required"),
			validation.Length(3, 10).Error("Credit memo number prefix must be between 3 and 10 characters"),
		),

		// * Ensure payment term is populated
		validation.Field(&bc.PaymentTerm,
			validation.Required.Error("Payment term is required"),
			validation.In(
				PaymentTermNet15,
				PaymentTermNet30,
				PaymentTermNet45,
				PaymentTermNet60,
				PaymentTermNet90,
				PaymentTermDueOnReceipt,
			).Error("Invalid payment term"),
		),

		// * Ensure transfer criteria is populated and a valid value
		validation.Field(&bc.TransferCriteria,
			validation.Required.Error("Transfer criteria is required"),
			validation.In(
				TransferCriteriaReadyAndCompleted,
				TransferCriteriaCompleted,
				TransferCriteriaReadyToBill,
				TransferCriteriaDocumentsAttached,
				TransferCriteriaPODReceived,
			).Error("Invalid transfer criteria"),
		),

		// * Ensure auto bill criteria is populated and a valid value
		validation.Field(&bc.AutoBillCriteria,
			validation.When(bc.AutoBill,
				validation.Required.Error("Auto Billing Criteria is required when auto bill is enabled"),
			),
			validation.In(
				AutoBillCriteriaDelivered,
				AutoBillCriteriaTransferred,
				AutoBillCriteriaMarkedReadyToBill,
				AutoBillCriteriaPODReceived,
				AutoBillCriteriaDocumentsVerified,
			).Error("Invalid auto bill criteria"),
		),

		// * Ensure billing exception handling is populated and a valid value
		validation.Field(&bc.BillingExceptionHandling,
			validation.Required.Error("Billing exception handling is required"),
			validation.In(
				BillingExceptionQueue,
				BillingExceptionNotify,
				BillingExceptionAutoResolve,
				BillingExceptionReject,
			).Error("Invalid billing exception handling"),
		),

		// * Ensure rate discrepancy threshold is populated
		validation.Field(&bc.RateDiscrepancyThreshold,
			validation.Required.Error("Rate discrepancy threshold is required"),
		),

		// * Ensure consolidation period days is populated
		validation.Field(&bc.ConsolidationPeriodDays,
			validation.When(bc.AllowInvoiceConsolidation,
				validation.Required.Error("Consolidation period days is required when invoice consolidation is enabled"),
				validation.Min(1).Error("Consolidation period days must be greater than 0"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (bc *BillingControl) GetID() string {
	return bc.ID.String()
}

func (bc *BillingControl) GetTableName() string {
	return "billing_controls"
}

func (bc *BillingControl) GetVersion() int64 {
	return bc.Version
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
