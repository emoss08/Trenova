package customerpayment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type CreditApplicationStatus string

const (
	CreditApplicationStatusApplied   = CreditApplicationStatus("Applied")
	CreditApplicationStatusUnapplied = CreditApplicationStatus("Unapplied")
)

func (s CreditApplicationStatus) IsValid() bool {
	switch s {
	case CreditApplicationStatusApplied, CreditApplicationStatusUnapplied:
		return true
	default:
		return false
	}
}

var _ bun.BeforeAppendModelHook = (*CreditMemoApplication)(nil)

// CreditMemoApplication records part of a posted credit memo being used to
// settle an invoice. Both documents are already on the ledger, so applying one
// to the other moves no money: it only decides which open item the credit pays.
type CreditMemoApplication struct {
	bun.BaseModel `bun:"table:credit_memo_applications,alias:cma" json:"-"`

	ID                  pulid.ID                `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID                `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID                `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CreditMemoInvoiceID pulid.ID                `json:"creditMemoInvoiceId" bun:"credit_memo_invoice_id,type:VARCHAR(100),notnull"`
	InvoiceID           pulid.ID                `json:"invoiceId"           bun:"invoice_id,type:VARCHAR(100),notnull"`
	AppliedAmountMinor  int64                   `json:"appliedAmountMinor"  bun:"applied_amount_minor,type:BIGINT,notnull"`
	AccountingDate      int64                   `json:"accountingDate"      bun:"accounting_date,type:BIGINT,notnull"`
	LineNumber          int                     `json:"lineNumber"          bun:"line_number,type:INTEGER,notnull"`
	Status              CreditApplicationStatus `json:"status"              bun:"status,type:VARCHAR(20),notnull,default:'Applied'"`
	UnappliedAt         *int64                  `json:"unappliedAt"         bun:"unapplied_at,type:BIGINT,nullzero"`
	UnappliedByID       pulid.ID                `json:"unappliedById"       bun:"unapplied_by_id,type:VARCHAR(100),nullzero"`
	UnappliedReason     string                  `json:"unappliedReason"     bun:"unapplied_reason,type:TEXT,nullzero"`
	CreatedByID         pulid.ID                `json:"createdById"         bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt           int64                   `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64                   `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CreditMemo *invoice.Invoice `json:"creditMemo,omitempty" bun:"rel:belongs-to,join:credit_memo_invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Invoice    *invoice.Invoice `json:"invoice,omitempty"    bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *CreditMemoApplication) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OrganizationID, validation.Required),
		validation.Field(&a.BusinessUnitID, validation.Required),
		validation.Field(
			&a.CreditMemoInvoiceID,
			validation.Required.Error("Credit memo is required"),
		),
		validation.Field(&a.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(
			&a.AppliedAmountMinor,
			validation.Min(int64(1)).Error("Applied amount must be greater than zero"),
		),
		validation.Field(
			&a.AccountingDate,
			validation.Required.Error("Accounting date is required"),
		),
	))
	if a.CreditMemoInvoiceID == a.InvoiceID {
		multiErr.Add(
			"invoiceId",
			errortypes.ErrInvalid,
			"A credit memo cannot be applied to itself",
		)
	}
}

func (a *CreditMemoApplication) IsApplied() bool {
	return a != nil && a.Status == CreditApplicationStatusApplied
}

func (a *CreditMemoApplication) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("cma_")
		}
		if a.Status == "" {
			a.Status = CreditApplicationStatusApplied
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *CreditMemoApplication) GetID() pulid.ID {
	return a.ID
}

func (a *CreditMemoApplication) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *CreditMemoApplication) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (a *CreditMemoApplication) GetTableName() string {
	return "credit_memo_applications"
}
