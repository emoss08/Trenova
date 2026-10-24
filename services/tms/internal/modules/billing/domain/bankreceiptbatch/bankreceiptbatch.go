package bankreceiptbatch

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type BankReceiptBatch struct {
	keel.Table `keel:"bank_receipt_import_batches"`

	ID                   pulid.ID `json:"id" keel:"id,pk,autoid:brib_"`
	OrganizationID       pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Source               string   `json:"source" keel:"source"`
	Reference            string   `json:"reference" keel:"reference,nullzero"`
	Status               Status   `json:"status" keel:"status"`
	ImportedCount        int64    `json:"importedCount" keel:"imported_count"`
	MatchedCount         int64    `json:"matchedCount" keel:"matched_count"`
	ExceptionCount       int64    `json:"exceptionCount" keel:"exception_count"`
	ImportedAmountMinor  int64    `json:"importedAmountMinor" keel:"imported_amount_minor"`
	MatchedAmountMinor   int64    `json:"matchedAmountMinor" keel:"matched_amount_minor"`
	ExceptionAmountMinor int64    `json:"exceptionAmountMinor" keel:"exception_amount_minor"`
	CreatedByID          pulid.ID `json:"createdById" keel:"created_by_id"`
	UpdatedByID          pulid.ID `json:"updatedById" keel:"updated_by_id,nullzero"`
	Version              int64    `json:"version" keel:"version,version"`
	CreatedAt            int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (b *BankReceiptBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.Source, validation.Required),
	))
}
