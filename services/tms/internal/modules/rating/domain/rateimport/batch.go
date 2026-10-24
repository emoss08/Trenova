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

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/errortypes"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*RateImportBatch)(nil)
)

const maxFileNameLength = 255

// RateImportBatch is one uploaded rate sheet.
type RateImportBatch struct {
	keel.Table `keel:"rate_import_batches"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:rib_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RateAgreementID pulid.ID `json:"rateAgreementId" keel:"rate_agreement_id"`

	FileName     string       `json:"fileName" keel:"file_name"`
	SourceFormat SourceFormat `json:"sourceFormat" keel:"source_format"`
	Status       Status       `json:"status" keel:"status"`

	// EffectiveFrom is the day the imported rules start pricing. It is the
	// caller's decision rather than the sheet's: a tariff is negotiated to take
	// effect on a date, and the day somebody happened to upload it is not it.
	EffectiveFrom int64 `json:"effectiveFrom" keel:"effective_from"`

	// Mapping is which column supplied which field. It is stored so a committed
	// import can explain itself, and so the next sheet from the same source can
	// start from what worked last time.
	Mapping map[string]int `json:"mapping" keel:"mapping,jsonb,nullzero"`

	// UnmappedHeaders are the columns nothing was read from. They are kept
	// because a column silently ignored is how a sheet imports looking complete
	// while a discount nobody noticed never made it in.
	UnmappedHeaders []string `json:"unmappedHeaders" keel:"unmapped_headers,jsonb,nullzero"`

	// Changes is the dry run: what committing this sheet would do to each lane.
	Changes []pkgrateimport.Change `json:"changes" keel:"changes,jsonb,nullzero"`

	Summary *pkgrateimport.Summary `json:"summary" keel:"summary,jsonb"`

	RowCount   int `json:"rowCount" keel:"row_count"`
	ErrorCount int `json:"errorCount" keel:"error_count"`

	Error string `json:"error" keel:"error,nullzero"`

	UploadedByID *pulid.ID `json:"uploadedById" keel:"uploaded_by_id"`
	CommittedAt  *int64    `json:"committedAt" keel:"committed_at"`
	CommittedBy  *pulid.ID `json:"committedBy" keel:"committed_by_id"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"`
	Organization *tenant.Organization         `json:"-"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" keel:"belongs:rate_agreement_id"`
	Rows         []*RateImportRow             `json:"rows,omitempty" keel:"many:rate_import_batch_id"`
}

func (b *RateImportBatch) BeforeCreate(_ context.Context) error {
	if b.Status == "" {
		b.Status = StatusPending
	}

	return nil
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

func (b *RateImportBatch) GetID() pulid.ID { return b.ID }

func (b *RateImportBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *RateImportBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *RateImportBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *RateImportBatch) GetTableName() string { return "rate_import_batches" }
