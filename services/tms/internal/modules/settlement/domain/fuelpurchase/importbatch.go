package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const maxImportFileNameLength = 255

var (
	_ domaintypes.PostgresSearchable     = (*ImportBatch)(nil)
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
	keel.Table `keel:"fuel_purchase_import_batches"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:fpib_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Provider      CardProvider `json:"provider" keel:"provider"`
	Origin        ImportOrigin `json:"origin" keel:"origin"`
	FeedReference string       `json:"feedReference" keel:"feed_reference,nullzero"`
	DocumentID    *pulid.ID    `json:"documentId" keel:"document_id"`
	FileName      string       `json:"fileName" keel:"file_name,nullzero"`
	SourceFormat  SourceFormat `json:"sourceFormat" keel:"source_format,nullzero"`
	Status        ImportStatus `json:"status" keel:"status"`

	DefaultFuelType   domaintypes.IFTAFuelType `json:"defaultFuelType" keel:"default_fuel_type,nullzero"`
	DefaultFuelCardID *pulid.ID                `json:"defaultFuelCardId" keel:"default_fuel_card_id"`
	DefaultCurrency   string                   `json:"defaultCurrency" keel:"default_currency"`

	Mapping         map[string]int `json:"mapping" keel:"mapping,jsonb,nullzero"`
	UnmappedHeaders []string       `json:"unmappedHeaders" keel:"unmapped_headers,jsonb,nullzero"`
	Summary         *ImportSummary `json:"summary" keel:"summary,jsonb"`

	RowCount       int `json:"rowCount" keel:"row_count"`
	ErrorCount     int `json:"errorCount" keel:"error_count"`
	CommittedCount int `json:"committedCount" keel:"committed_count"`

	Error string `json:"error" keel:"error,nullzero"`

	UploadedByID  pulid.ID `json:"uploadedById" keel:"uploaded_by_id,nullzero"`
	StagedAt      *int64   `json:"stagedAt" keel:"staged_at"`
	CommittedAt   *int64   `json:"committedAt" keel:"committed_at"`
	CommittedByID pulid.ID `json:"committedById" keel:"committed_by_id,nullzero"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Document        *document.Document `json:"document,omitempty" keel:"belongs:document_id"`
	DefaultFuelCard *FuelCard          `json:"defaultFuelCard,omitempty" keel:"belongs:default_fuel_card_id"`
	UploadedBy      *tenant.User       `json:"uploadedBy,omitempty" keel:"belongs:uploaded_by_id"`
	CommittedBy     *tenant.User       `json:"committedBy,omitempty" keel:"belongs:committed_by_id"`
	Rows            []*ImportRow       `json:"rows,omitempty" keel:"many:import_batch_id"`
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

// GetPostgresSearchConfig makes a run findable by what it read: the file somebody
// uploaded, or the remote path a feed pulled. Everything else on a batch is an
// enum or a count, which the field filters already handle.
func (b *ImportBatch) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fpib",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "feed_reference",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (b *ImportBatch) BeforeCreate(ctx context.Context) error {
	if b.Status == "" {
		b.Status = ImportStatusPending
	}
	if b.DefaultCurrency == "" {
		b.DefaultCurrency = "USD"
	}
	if b.DefaultCurrency == "" {
		b.DefaultCurrency = money.DefaultCurrencyCode
	}

	return nil
}
