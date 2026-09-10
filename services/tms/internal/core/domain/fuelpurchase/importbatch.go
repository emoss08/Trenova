package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const maxImportFileNameLength = 255

var (
	_ bun.BeforeAppendModelHook          = (*ImportBatch)(nil)
	_ pagination.CursorEntity            = (*ImportBatch)(nil)
	_ validationframework.TenantedEntity = (*ImportBatch)(nil)
)

type ImportSummary struct {
	RowCount             int               `json:"rowCount"`
	NewCount             int               `json:"newCount"`
	DuplicateInFileCount int               `json:"duplicateInFileCount"`
	AlreadyImportedCount int               `json:"alreadyImportedCount"`
	ErrorCount           int               `json:"errorCount"`
	TotalGallons         string            `json:"totalGallons"`
	TotalAmountMinor     int64             `json:"totalAmountMinor"`
	ByFuelType           map[string]string `json:"byFuelType"`
	ByJurisdiction       map[string]string `json:"byJurisdiction"`
	EarliestPurchasedAt  int64             `json:"earliestPurchasedAt"`
	LatestPurchasedAt    int64             `json:"latestPurchasedAt"`
}

type ImportBatch struct {
	bun.BaseModel             `bun:"table:fuel_purchase_import_batches,alias:fpib" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                        json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Provider      CardProvider `json:"provider"      bun:"provider,type:fuel_card_provider_enum,notnull"`
	Origin        ImportOrigin `json:"origin"        bun:"origin,type:fuel_import_origin_enum,notnull"`
	FeedReference string       `json:"feedReference" bun:"feed_reference,type:VARCHAR(255),nullzero"`
	DocumentID    *pulid.ID    `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`
	FileName      string       `json:"fileName"      bun:"file_name,type:VARCHAR(255),nullzero"`
	SourceFormat  SourceFormat `json:"sourceFormat"  bun:"source_format,type:fuel_import_format_enum,nullzero"`
	Status        ImportStatus `json:"status"        bun:"status,type:fuel_import_status_enum,notnull,default:'Pending'"`

	DefaultFuelType   domaintypes.IFTAFuelType `json:"defaultFuelType"   bun:"default_fuel_type,type:ifta_fuel_type_enum,nullzero"`
	DefaultFuelCardID *pulid.ID                `json:"defaultFuelCardId" bun:"default_fuel_card_id,type:VARCHAR(100),nullzero"`
	DefaultCurrency   string                   `json:"defaultCurrency"   bun:"default_currency,type:VARCHAR(3),notnull,default:'USD'"`

	Mapping         map[string]int `json:"mapping"         bun:"mapping,type:JSONB,nullzero"`
	UnmappedHeaders []string       `json:"unmappedHeaders" bun:"unmapped_headers,type:JSONB,nullzero"`
	Summary         *ImportSummary `json:"summary"         bun:"summary,type:JSONB,nullzero"`

	RowCount       int `json:"rowCount"       bun:"row_count,type:INTEGER,notnull,default:0"`
	ErrorCount     int `json:"errorCount"     bun:"error_count,type:INTEGER,notnull,default:0"`
	CommittedCount int `json:"committedCount" bun:"committed_count,type:INTEGER,notnull,default:0"`

	Error string `json:"error" bun:"error,type:TEXT,nullzero"`

	UploadedByID  pulid.ID `json:"uploadedById"  bun:"uploaded_by_id,type:VARCHAR(100),nullzero"`
	StagedAt      *int64   `json:"stagedAt"      bun:"staged_at,type:BIGINT,nullzero"`
	CommittedAt   *int64   `json:"committedAt"   bun:"committed_at,type:BIGINT,nullzero"`
	CommittedByID pulid.ID `json:"committedById" bun:"committed_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Document        *document.Document `json:"document,omitempty"        bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DefaultFuelCard *FuelCard          `json:"defaultFuelCard,omitempty" bun:"rel:belongs-to,join:default_fuel_card_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	UploadedBy      *tenant.User       `json:"uploadedBy,omitempty"      bun:"rel:belongs-to,join:uploaded_by_id=id"`
	CommittedBy     *tenant.User       `json:"committedBy,omitempty"     bun:"rel:belongs-to,join:committed_by_id=id"`
	Rows            []*ImportRow       `json:"rows,omitempty"            bun:"rel:has-many,join:id=import_batch_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (b *ImportBatch) Normalize() {
	b.FileName = strings.TrimSpace(b.FileName)
	b.FeedReference = strings.TrimSpace(b.FeedReference)
	if b.Origin == "" {
		b.Origin = ImportOriginUpload
	}
	b.DefaultCurrency = strings.ToUpper(strings.TrimSpace(b.DefaultCurrency))
	if b.DefaultCurrency == "" {
		b.DefaultCurrency = money.DefaultCurrencyCode
	}
	if b.Status == "" {
		b.Status = ImportStatusPending
	}
}

func (b *ImportBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[CardProvider]("Provider is not valid"),
		),
		validation.Field(&b.FileName,
			validation.Length(0, maxImportFileNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&b.SourceFormat,
			domainvalidation.ValidEnum[SourceFormat]("Source format is not valid"),
		),
		validation.Field(&b.Origin,
			validation.Required.Error("Origin is required"),
			domainvalidation.ValidEnum[ImportOrigin]("Origin is not valid"),
		),
		validation.Field(&b.FeedReference,
			validation.Length(0, maxImportFileNameLength).
				Error("Feed reference cannot be longer than 255 characters"),
		),
		validation.Field(&b.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ImportStatus]("Status is not valid"),
		),
		validation.Field(&b.DefaultFuelType,
			domainvalidation.ValidEnum[domaintypes.IFTAFuelType]("Default fuel type is not valid"),
		),
		validation.Field(&b.DefaultCurrency,
			validation.Required.Error("Default currency is required"),
			validation.Match(domaintypes.CurrencyCodeRegex).
				Error("Default currency must be a three-letter ISO code"),
		),
	))

	b.validateLifecycle(multiErr)
}

func (b *ImportBatch) validateLifecycle(multiErr *errortypes.MultiError) {
	if b.RowCount < 0 || b.ErrorCount < 0 || b.CommittedCount < 0 {
		multiErr.Add("rowCount", errortypes.ErrInvalid, "Row counts cannot be negative")
	}

	if b.Status == ImportStatusParsed && !b.Origin.IsFeed() &&
		(b.DocumentID == nil || b.DocumentID.IsNil()) {
		multiErr.Add(
			"documentId",
			errortypes.ErrRequired,
			"A parsed import must reference the uploaded file",
		)
	}

	if b.Status == ImportStatusCommitted {
		if b.CommittedAt == nil || *b.CommittedAt <= 0 {
			multiErr.Add(
				"committedAt",
				errortypes.ErrRequired,
				"A committed import must record when it was committed",
			)
		}
		if b.CommittedByID.IsNil() && !b.Origin.IsFeed() {
			multiErr.Add(
				"committedById",
				errortypes.ErrRequired,
				"A committed import must record who committed it",
			)
		}
		return
	}

	if b.CommittedAt != nil || !b.CommittedByID.IsNil() {
		multiErr.Add(
			"committedAt",
			errortypes.ErrInvalid,
			"Only a committed import carries a commit stamp",
		)
	}
}

func (b *ImportBatch) CanStage() bool { return b.Status.CanStage() }

func (b *ImportBatch) CanCommit() bool { return b.Status.CanCommit() }

func (b *ImportBatch) CanDiscard() bool { return b.Status.CanDiscard() }

func (b *ImportBatch) IsTerminal() bool { return b.Status.IsTerminal() }

func (b *ImportBatch) HasBlockingErrors() bool { return b.ErrorCount > 0 }

func (b *ImportBatch) NewRowCount() int {
	if b.Summary == nil {
		return 0
	}
	return b.Summary.NewCount
}

func (b *ImportBatch) HasRowsToCommit() bool { return b.NewRowCount() > 0 }

func (b *ImportBatch) IsFeed() bool { return b.Origin.IsFeed() }

func (b *ImportBatch) GetID() pulid.ID { return b.ID }

func (b *ImportBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *ImportBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *ImportBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *ImportBatch) GetTableName() string { return "fuel_purchase_import_batches" }

func (b *ImportBatch) GetResourceType() string { return "fuel_purchase_import" }

func (b *ImportBatch) GetResourceID() string { return b.ID.String() }

func (b *ImportBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("fpib_")
		}
		if b.Status == "" {
			b.Status = ImportStatusPending
		}
		if b.DefaultCurrency == "" {
			b.DefaultCurrency = money.DefaultCurrencyCode
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}
