package bankreceiptbatch

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type BankReceiptBatch struct {
	bun.BaseModel `bun:"table:bank_receipt_import_batches,alias:brib" json:"-"`
	keel.Table    `keel:"bank_receipt_import_batches"`

	ID                   pulid.ID `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:brib_"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	Source               string   `json:"source"               bun:"source,type:VARCHAR(100),notnull" keel:"source"`
	Reference            string   `json:"reference"            bun:"reference,type:VARCHAR(100),nullzero" keel:"reference,nullzero"`
	Status               Status   `json:"status"               bun:"status,type:VARCHAR(50),notnull" keel:"status"`
	ImportedCount        int64    `json:"importedCount"        bun:"imported_count,type:BIGINT,notnull" keel:"imported_count"`
	MatchedCount         int64    `json:"matchedCount"         bun:"matched_count,type:BIGINT,notnull" keel:"matched_count"`
	ExceptionCount       int64    `json:"exceptionCount"       bun:"exception_count,type:BIGINT,notnull" keel:"exception_count"`
	ImportedAmountMinor  int64    `json:"importedAmountMinor"  bun:"imported_amount_minor,type:BIGINT,notnull" keel:"imported_amount_minor"`
	MatchedAmountMinor   int64    `json:"matchedAmountMinor"   bun:"matched_amount_minor,type:BIGINT,notnull" keel:"matched_amount_minor"`
	ExceptionAmountMinor int64    `json:"exceptionAmountMinor" bun:"exception_amount_minor,type:BIGINT,notnull" keel:"exception_amount_minor"`
	CreatedByID          pulid.ID `json:"createdById"          bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	UpdatedByID          pulid.ID `json:"updatedById"          bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	Version              int64    `json:"version"              bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64    `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (b *BankReceiptBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.Source, validation.Required),
	))
}

func (b *BankReceiptBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("brib_")
		}
		b.CreatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}
	return nil
}
