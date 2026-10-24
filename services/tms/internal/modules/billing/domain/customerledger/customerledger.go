package customerledger

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomerLedgerEntry)(nil)

// CustomerLedgerEntry is the AR subledger: one signed row per document that
// moved a customer's balance. Aging is built from these rather than from
// invoices, because credits and unapplied cash never appear on an invoice.
type CustomerLedgerEntry struct {
	bun.BaseModel `bun:"table:customer_ledger_entries,alias:cle" json:"-"`
	keel.Table    `keel:"customer_ledger_entries"`

	ID               pulid.ID `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:cle_"`
	OrganizationID   pulid.ID `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	CustomerID       pulid.ID `json:"customerId"       bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	SourceObjectType string   `json:"sourceObjectType" bun:"source_object_type,type:VARCHAR(50),notnull" keel:"source_object_type"`
	SourceObjectID   string   `json:"sourceObjectId"   bun:"source_object_id,type:VARCHAR(100),notnull" keel:"source_object_id"`
	SourceEventType  string   `json:"sourceEventType"  bun:"source_event_type,type:VARCHAR(100),notnull" keel:"source_event_type"`
	RelatedInvoiceID pulid.ID `json:"relatedInvoiceId" bun:"related_invoice_id,type:VARCHAR(100),nullzero" keel:"related_invoice_id,nullzero"`
	DocumentNumber   string   `json:"documentNumber"   bun:"document_number,type:VARCHAR(100),nullzero" keel:"document_number,nullzero"`
	TransactionDate  int64    `json:"transactionDate"  bun:"transaction_date,type:BIGINT,notnull" keel:"transaction_date"`
	LineNumber       int      `json:"lineNumber"       bun:"line_number,type:INTEGER,notnull" keel:"line_number"`
	AmountMinor      int64    `json:"amountMinor"      bun:"amount_minor,type:BIGINT,notnull" keel:"amount_minor"`
	CreatedByID      pulid.ID `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	CreatedAt        int64    `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	Customer       *customer.Customer `json:"customer,omitempty"       bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:customer_id"`
	RelatedInvoice *invoice.Invoice   `json:"relatedInvoice,omitempty" bun:"rel:belongs-to,join:related_invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:related_invoice_id"`
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

func (e *CustomerLedgerEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("cle_")
		}
		e.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
