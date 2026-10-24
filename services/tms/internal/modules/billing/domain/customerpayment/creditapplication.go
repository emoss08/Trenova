package customerpayment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

// CreditMemoApplication records part of a posted credit memo being used to
// settle an invoice. Both documents are already on the ledger, so applying one
// to the other moves no money: it only decides which open item the credit pays.
type CreditMemoApplication struct {
	keel.Table `keel:"credit_memo_applications"`

	ID                  pulid.ID                `json:"id" keel:"id,pk,autoid:cma_"`
	OrganizationID      pulid.ID                `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID                `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CreditMemoInvoiceID pulid.ID                `json:"creditMemoInvoiceId" keel:"credit_memo_invoice_id"`
	InvoiceID           pulid.ID                `json:"invoiceId" keel:"invoice_id"`
	AppliedAmountMinor  int64                   `json:"appliedAmountMinor" keel:"applied_amount_minor"`
	AccountingDate      int64                   `json:"accountingDate" keel:"accounting_date"`
	LineNumber          int                     `json:"lineNumber" keel:"line_number"`
	Status              CreditApplicationStatus `json:"status" keel:"status"`
	UnappliedAt         *int64                  `json:"unappliedAt" keel:"unapplied_at"`
	UnappliedByID       pulid.ID                `json:"unappliedById" keel:"unapplied_by_id,nullzero"`
	UnappliedReason     string                  `json:"unappliedReason" keel:"unapplied_reason,nullzero"`
	CreatedByID         pulid.ID                `json:"createdById" keel:"created_by_id"`
	CreatedAt           int64                   `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64                   `json:"updatedAt" keel:"updated_at,autoupdate"`

	CreditMemo *invoice.Invoice `json:"creditMemo,omitempty" keel:"belongs:credit_memo_invoice_id"`
	Invoice    *invoice.Invoice `json:"invoice,omitempty" keel:"belongs:invoice_id"`
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

func (a *CreditMemoApplication) BeforeCreate(ctx context.Context) error {
	if a.Status == "" {
		a.Status = CreditApplicationStatusApplied
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
