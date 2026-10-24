package customerledger

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// The AR subledger — every document that moved a customer's balance
type CustomerLedgerEntry struct {
	keel.Table `                                                       keel:"customer_ledger_entries,report=Accounting"`

	ID               pulid.ID `json:"id" keel:"id,pk,autoid:cle_"`
	OrganizationID   pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CustomerID       pulid.ID `json:"customerId" keel:"customer_id"`
	SourceObjectType string   `json:"sourceObjectType" keel:"source_object_type,label=Source Object"`
	SourceObjectID   string   `json:"sourceObjectId" keel:"source_object_id"`
	SourceEventType  string   `json:"sourceEventType" keel:"source_event_type,label=Source Event"`
	RelatedInvoiceID pulid.ID `json:"relatedInvoiceId" keel:"related_invoice_id,nullzero"`
	DocumentNumber   string   `json:"documentNumber" keel:"document_number,nullzero"`
	TransactionDate  int64    `json:"transactionDate" keel:"transaction_date"`
	LineNumber       int      `json:"lineNumber" keel:"line_number,label=Line,dimension"`
	AmountMinor      int64    `json:"amountMinor" keel:"amount_minor,label=Amount (Minor)"`
	CreatedByID      pulid.ID `json:"createdById" keel:"created_by_id"`
	CreatedAt        int64    `json:"createdAt" keel:"created_at,autocreate"`

	Customer       *customer.Customer `json:"customer,omitempty" keel:"belongs:customer_id"`
	RelatedInvoice *invoice.Invoice   `json:"relatedInvoice,omitempty" keel:"belongs:related_invoice_id,label=Invoice"`
}

func (e *CustomerLedgerEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required),
		validation.Field(&e.BusinessUnitID, validation.Required),
		validation.Field(&e.CustomerID, validation.Required),
		validation.Field(&e.SourceObjectType, validation.Required, validation.Length(1, 50)),
		validation.Field(&e.SourceObjectID, validation.Required, validation.Length(1, 100)),
		validation.Field(&e.SourceEventType, validation.Required, validation.Length(1, 100)),
		validation.Field(&e.TransactionDate, validation.Required),
		validation.Field(&e.CreatedByID, validation.Required),
	))

	if e.LineNumber <= 0 {
		multiErr.Add("lineNumber", errortypes.ErrInvalid, "Line number must be positive")
	}
}

func (e *CustomerLedgerEntry) GetTableName() string { return "customer_ledger_entries" }

func (e *CustomerLedgerEntry) GetID() pulid.ID { return e.ID }

func (e *CustomerLedgerEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *CustomerLedgerEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *CustomerLedgerEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }
