package bankreceiptbatch

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Batch struct {
	bun.BaseModel `bun:"table:bank_receipt_import_batches,alias:brib" json:"-"`

	ID                   pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Source               string   `json:"source" bun:"source,type:VARCHAR(100),notnull"`
	Reference            string   `json:"reference" bun:"reference,type:VARCHAR(100),nullzero"`
	Status               Status   `json:"status" bun:"status,type:VARCHAR(50),notnull"`
	ImportedCount        int64    `json:"importedCount" bun:"imported_count,type:BIGINT,notnull,default:0"`
	MatchedCount         int64    `json:"matchedCount" bun:"matched_count,type:BIGINT,notnull,default:0"`
	ExceptionCount       int64    `json:"exceptionCount" bun:"exception_count,type:BIGINT,notnull,default:0"`
	ImportedAmountMinor  int64    `json:"importedAmountMinor" bun:"imported_amount_minor,type:BIGINT,notnull,default:0"`
	MatchedAmountMinor   int64    `json:"matchedAmountMinor" bun:"matched_amount_minor,type:BIGINT,notnull,default:0"`
	ExceptionAmountMinor int64    `json:"exceptionAmountMinor" bun:"exception_amount_minor,type:BIGINT,notnull,default:0"`
	CreatedByID          pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID          pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version              int64    `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type SourceOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (b *Batch) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.Source, validation.Required),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (b *Batch) BeforeAppendModel(_ context.Context, query bun.Query) error {
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
