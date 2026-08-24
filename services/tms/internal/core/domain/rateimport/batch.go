// Package rateimport stages a rate sheet so it can be read before it is
// applied.
//
// A sheet is never written straight into a contract. It is parsed into rows,
// each row is validated on its own, and the whole thing is diffed against what
// the contract already says. Only then can somebody commit it — and committing
// goes through the same close-out-and-insert primitive a general rate increase
// uses, so history is never mutated.
package rateimport

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateImportBatch)(nil)
	_ validationframework.TenantedEntity = (*RateImportBatch)(nil)
)

const maxFileNameLength = 255

// RateImportBatch is one uploaded rate sheet.
type RateImportBatch struct {
	bun.BaseModel             `bun:"table:rate_import_batches,alias:rib" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull"`

	FileName     string       `json:"fileName"     bun:"file_name,type:VARCHAR(255),notnull"`
	SourceFormat SourceFormat `json:"sourceFormat" bun:"source_format,type:rate_import_format_enum,notnull"`
	Status       Status       `json:"status"       bun:"status,type:rate_import_status_enum,notnull,default:'Pending'"`

	// EffectiveFrom is the day the imported rules start pricing. It is the
	// caller's decision rather than the sheet's: a tariff is negotiated to take
	// effect on a date, and the day somebody happened to upload it is not it.
	EffectiveFrom int64 `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`

	// Mapping is which column supplied which field. It is stored so a committed
	// import can explain itself, and so the next sheet from the same source can
	// start from what worked last time.
	Mapping map[string]int `json:"mapping" bun:"mapping,type:JSONB,nullzero"`

	// UnmappedHeaders are the columns nothing was read from. They are kept
	// because a column silently ignored is how a sheet imports looking complete
	// while a discount nobody noticed never made it in.
	UnmappedHeaders []string `json:"unmappedHeaders" bun:"unmapped_headers,type:JSONB,nullzero"`

	// Changes is the dry run: what committing this sheet would do to each lane.
	Changes []pkgrateimport.Change `json:"changes" bun:"changes,type:JSONB,nullzero"`

	Summary *pkgrateimport.Summary `json:"summary" bun:"summary,type:JSONB,nullzero"`

	RowCount   int `json:"rowCount"   bun:"row_count,type:INTEGER,notnull,default:0"`
	ErrorCount int `json:"errorCount" bun:"error_count,type:INTEGER,notnull,default:0"`

	Error string `json:"error" bun:"error,type:TEXT,nullzero"`

	UploadedByID *pulid.ID `json:"uploadedById" bun:"uploaded_by_id,type:VARCHAR(100),nullzero"`
	CommittedAt  *int64    `json:"committedAt"  bun:"committed_at,type:BIGINT,nullzero"`
	CommittedBy  *pulid.ID `json:"committedBy"  bun:"committed_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization         `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" bun:"rel:belongs-to,join:rate_agreement_id=id"`
	Rows         []*RateImportRow             `json:"rows,omitempty"      bun:"rel:has-many,join:id=rate_import_batch_id"`
}

func (b *RateImportBatch) applyDefaults() {
	if b.Status == "" {
		b.Status = StatusPending
	}
}

func (b *RateImportBatch) Validate(multiErr *errortypes.MultiError) {
	b.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.RateAgreementID,
			validation.Required.Error("An agreement to import into is required"),
		),
		validation.Field(&b.FileName,
			validation.Required.Error("A file name is required"),
			validation.Length(1, maxFileNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&b.EffectiveFrom,
			validation.Required.Error("An effective date is required"),
		),
	))
}

// HasBlockingErrors reports whether any row failed to read.
//
// A sheet with unreadable rows can still be committed — the rows that read fine
// are usually most of it — but somebody has to see the count first, because the
// missing lanes are the ones that will stop pricing.
func (b *RateImportBatch) HasBlockingErrors() bool {
	return b.ErrorCount > 0
}

func (b *RateImportBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("rib_")
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

func (b *RateImportBatch) GetID() pulid.ID { return b.ID }

func (b *RateImportBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *RateImportBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *RateImportBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *RateImportBatch) GetTableName() string { return "rate_import_batches" }
