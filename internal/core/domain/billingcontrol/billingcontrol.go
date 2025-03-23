package billingcontrol

import (
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
	InvoiceDueAfterDays int64  `json:"invoiceDueAfterDays" bun:"invoice_due_after_days,type:INTEGER,notnull,default:30"`
	ShowInvoiceDueDate  bool   `json:"showInvoiceDueDate" bun:"show_invoice_due_date,type:BOOLEAN,notnull,default:true"`
	InvoiceTerms        string `json:"invoiceTerms" bun:"invoice_terms,type:TEXT,nullzero"`
	InvoiceFooter       string `json:"invoiceFooter" bun:"invoice_footer,type:TEXT,nullzero"`
	ShowAmountDue       bool   `json:"showAmountDue" bun:"show_amount_due,type:BOOLEAN,notnull,default:true"`

	// Controls for the billing process
	TransferCriteria          TransferCriteria `json:"transferCriteria" bun:"transfer_criteria,type:transfer_criteria_enum,notnull,default:'ReadyAndCompleted'"`
	EnforceCustomerBillingReq bool             `json:"enforceCustomerBillingReq" bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"` // * Enforce customer billing requirements before billing
	ValidateCustomerRates     bool             `json:"validateCustomerRates" bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`          // * Validate customer rates before billing
	AutoMarkReadyToBill       bool             `json:"autoMarkReadyToBill" bun:"auto_mark_ready_to_bill,type:BOOLEAN,notnull,default:true"`            // * Automatically mark shipment as ready to bill if it meets billing requirements

	// Automated billing controls
	AutoBill         bool             `json:"autoBill" bun:"auto_bill,type:BOOLEAN,notnull,default:true"` // * Automatically bill shipment if it meets billing requirements
	AutoBillCriteria AutoBillCriteria `json:"autoBillCriteria" bun:"auto_bill_criteria,type:auto_bill_criteria_enum,notnull,default:'Delivered'"`

	// Exception handling
	BillingExceptionHandling      BillingExceptionHandling `json:"billingExceptionHandling" bun:"billing_exception_handling,type:billing_exception_handling_enum,notnull,default:'Queue'"`
	RateDiscrepancyThreshold      float64                  `json:"rateDiscrepancyThreshold" bun:"rate_discrepancy_threshold,type:DECIMAL(10,2),notnull,default:5.00"` // * Percentage threshold for rate discrepancies
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
