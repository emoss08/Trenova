package bankreceipt

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Deposits read off the bank statement and their match state
type BankReceipt struct {
	keel.Table `                                            keel:"bank_receipts,report=Accounting"`

	ID                       pulid.ID `json:"id" keel:"id,pk,autoid:brcpt_"`
	OrganizationID           pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID           pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ReceiptDate              int64    `json:"receiptDate" keel:"receipt_date"`
	AmountMinor              int64    `json:"amountMinor" keel:"amount_minor,label=Amount (Minor)"`
	ReferenceNumber          string   `json:"referenceNumber" keel:"reference_number,nullzero"`
	Memo                     string   `json:"memo" keel:"memo,nullzero,label=Bank Memo"`
	Status                   Status   `json:"status" keel:"status"`
	ImportBatchID            pulid.ID `json:"importBatchId" keel:"import_batch_id,nullzero"`
	MatchedCustomerPaymentID pulid.ID `json:"matchedCustomerPaymentId" keel:"matched_customer_payment_id,nullzero"`
	MatchedAt                *int64   `json:"matchedAt" keel:"matched_at"`
	MatchedByID              pulid.ID `json:"matchedById" keel:"matched_by_id,nullzero"`
	ExceptionReason          string   `json:"exceptionReason" keel:"exception_reason,nullzero"`
	CreatedByID              pulid.ID `json:"createdById" keel:"created_by_id"`
	UpdatedByID              pulid.ID `json:"updatedById" keel:"updated_by_id,nullzero"`
	Version                  int64    `json:"version" keel:"version,version"`
	CreatedAt                int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	MatchedCustomerPayment *customerpayment.Payment `json:"matchedCustomerPayment,omitempty" keel:"belongs:matched_customer_payment_id,label=Matched Payment"`
}

func (r *BankReceipt) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(&r.ReceiptDate, validation.Required),
	))
	if r.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Receipt amount must be greater than zero",
		)
	}
}
