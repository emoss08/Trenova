package bankreceipt

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type BankReceipt struct {
	bun.BaseModel `bun:"table:bank_receipts,alias:br" json:"-"`
	keel.Table    `keel:"bank_receipts"`

	ID                       pulid.ID `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:brcpt_"`
	OrganizationID           pulid.ID `json:"organizationId"           bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID           pulid.ID `json:"businessUnitId"           bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ReceiptDate              int64    `json:"receiptDate"              bun:"receipt_date,type:BIGINT,notnull" keel:"receipt_date"`
	AmountMinor              int64    `json:"amountMinor"              bun:"amount_minor,type:BIGINT,notnull" keel:"amount_minor"`
	ReferenceNumber          string   `json:"referenceNumber"          bun:"reference_number,type:VARCHAR(100),nullzero" keel:"reference_number,nullzero"`
	Memo                     string   `json:"memo"                     bun:"memo,type:TEXT,nullzero" keel:"memo,nullzero"`
	Status                   Status   `json:"status"                   bun:"status,type:VARCHAR(50),notnull" keel:"status"`
	ImportBatchID            pulid.ID `json:"importBatchId"            bun:"import_batch_id,type:VARCHAR(100),nullzero" keel:"import_batch_id,nullzero"`
	MatchedCustomerPaymentID pulid.ID `json:"matchedCustomerPaymentId" bun:"matched_customer_payment_id,type:VARCHAR(100),nullzero" keel:"matched_customer_payment_id,nullzero"`
	MatchedAt                *int64   `json:"matchedAt"                bun:"matched_at,type:BIGINT,nullzero" keel:"matched_at"`
	MatchedByID              pulid.ID `json:"matchedById"              bun:"matched_by_id,type:VARCHAR(100),nullzero" keel:"matched_by_id,nullzero"`
	ExceptionReason          string   `json:"exceptionReason"          bun:"exception_reason,type:TEXT,nullzero" keel:"exception_reason,nullzero"`
	CreatedByID              pulid.ID `json:"createdById"              bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	UpdatedByID              pulid.ID `json:"updatedById"              bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	Version                  int64    `json:"version"                  bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt                int64    `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                int64    `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	MatchedCustomerPayment *customerpayment.Payment `json:"matchedCustomerPayment,omitempty" bun:"rel:belongs-to,join:matched_customer_payment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:matched_customer_payment_id"`
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

func (r *BankReceipt) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("brcpt_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}
