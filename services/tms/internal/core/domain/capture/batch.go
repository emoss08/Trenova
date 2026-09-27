package capture

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	// MaxBatchPages bounds one acquisition. A feeder stack of a thousand
	// sheets is several boxes of paper; anything larger is a stuck loop on the
	// device, not a person scanning.
	MaxBatchPages        = 1000
	maxClientKeyLength   = 100
	maxJobNameLength     = 255
	maxManifestDigestLen = 64
)

// CaptureBatch is one acquisition: every page from one press of Scan, or one print
// job. The pages are kept in the order they arrived; how they are divided
// into documents lives on the items.
type CaptureBatch struct {
	bun.BaseModel             `bun:"table:capture_batches,alias:cbat" json:"-"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                pulid.ID    `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID    pulid.ID    `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID    `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	UserID            pulid.ID    `json:"userId"            bun:"user_id,type:VARCHAR(100),notnull"`
	DeviceID          pulid.ID    `json:"deviceId"          bun:"device_id,type:VARCHAR(100),notnull"`
	RequestID         *pulid.ID   `json:"requestId"         bun:"request_id,type:VARCHAR(100),nullzero"`
	ProfileID         *pulid.ID   `json:"profileId"         bun:"profile_id,type:VARCHAR(100),nullzero"`
	ClientKey         string      `json:"-"                 bun:"client_key,type:VARCHAR(100),notnull"`
	Source            Source      `json:"source"            bun:"source,type:VARCHAR(10),notnull"`
	Status            BatchStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull"`
	SourceName        string      `json:"sourceName"        bun:"source_name,type:VARCHAR(255),nullzero"`
	JobName           string      `json:"jobName"           bun:"job_name,type:VARCHAR(255),nullzero"`
	Settings          Settings    `json:"settings"          bun:"settings,type:JSONB,notnull,default:'{}'"`
	TargetType        string      `json:"targetType"        bun:"target_type,type:VARCHAR(50),nullzero"`
	TargetID          *pulid.ID   `json:"targetId"          bun:"target_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID    *pulid.ID   `json:"documentTypeId"    bun:"document_type_id,type:VARCHAR(100),nullzero"`
	ExpectedPageCount int         `json:"expectedPageCount" bun:"expected_page_count,type:INTEGER,notnull,default:0"`
	ReceivedPageCount int         `json:"receivedPageCount" bun:"received_page_count,type:INTEGER,notnull,default:0"`
	ItemCount         int         `json:"itemCount"         bun:"item_count,type:INTEGER,notnull,default:0"`
	FiledItemCount    int         `json:"filedItemCount"    bun:"filed_item_count,type:INTEGER,notnull,default:0"`
	ManifestDigest    string      `json:"-"                 bun:"manifest_digest,type:VARCHAR(64),nullzero"`
	FailureMessage    string      `json:"failureMessage"    bun:"failure_message,type:VARCHAR(500),nullzero"`
	SealedAt          *int64      `json:"sealedAt"          bun:"sealed_at,type:BIGINT,nullzero"`
	ProcessedAt       *int64      `json:"processedAt"       bun:"processed_at,type:BIGINT,nullzero"`
	RetainUntil       int64       `json:"retainUntil"       bun:"retain_until,type:BIGINT,notnull"`
	// RetentionRemindedAt is when the owner was told the unfiled pages are
	// about to go. It is set once, by whichever sweep claims the reminder.
	RetentionRemindedAt *int64         `json:"-"               bun:"retention_reminded_at,type:BIGINT,nullzero"`
	Version             int64          `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt           int64          `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64          `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Pages               []*CapturePage `json:"pages,omitempty" bun:"rel:has-many,join:id=batch_id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	Items               []*CaptureItem `json:"items,omitempty" bun:"rel:has-many,join:id=batch_id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`

	CaptureDevice *CaptureDevice       `json:"device,omitempty"       bun:"rel:belongs-to,join:device_id=id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	Organization  *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit  *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

// Settings are what the device actually used, which is not always what the
// profile asked for: a source that cannot scan duplex scans one side, and the
// record of that is what explains a batch with half its pages missing.
type Settings struct {
	Protocol      SourceProtocol `json:"protocol,omitempty"`
	Bitness       int            `json:"bitness,omitempty"`
	DPI           int            `json:"dpi,omitempty"`
	PixelType     PixelType      `json:"pixelType,omitempty"`
	Duplex        bool           `json:"duplex,omitempty"`
	Feeder        bool           `json:"feeder,omitempty"`
	BlankDiscard  bool           `json:"blankDiscard,omitempty"`
	ShowDriverUI  bool           `json:"showDriverUi,omitempty"`
	DriverVersion string         `json:"driverVersion,omitempty"`
	Application   string         `json:"application,omitempty"`
	// Refused lists the capabilities the source declined, by TWAIN name.
	Refused []string `json:"refused,omitempty"`
}

func (b *CaptureBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.UserID, validation.Required.Error("User is required")),
		validation.Field(&b.DeviceID, validation.Required.Error("Device is required")),
		validation.Field(&b.ClientKey,
			validation.Required.Error("Client key is required"),
			validation.Length(1, maxClientKeyLength),
		),
		validation.Field(&b.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[Source]("Source must be Scan or Print"),
		),
		validation.Field(&b.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[BatchStatus]("Status is not a batch status"),
		),
		validation.Field(&b.SourceName, validation.Length(0, maxSourceNameLength)),
		validation.Field(&b.JobName, validation.Length(0, maxJobNameLength)),
		validation.Field(&b.ExpectedPageCount, validation.Min(0), validation.Max(MaxBatchPages)),
		validation.Field(&b.ManifestDigest, validation.Length(0, maxManifestDigestLen)),
		validation.Field(&b.FailureMessage, validation.Length(0, maxFailureMessageLength)),
	))

	b.Target().Validate(multiErr, targetFields, false)
}

// Target is where the batch was asked to go, if anywhere.
func (b *CaptureBatch) Target() Target {
	return Target{
		ResourceType:   b.TargetType,
		ResourceID:     b.TargetID,
		DocumentTypeID: b.DocumentTypeID,
	}
}

// AcceptsPages reports whether the device may still add pages.
func (b *CaptureBatch) AcceptsPages() bool {
	return b.Status == BatchReceiving
}

// OpenItemCount is how many items still wait on a person. A batch read with
// its items counts them; a list row, which carries only the counts, reads the
// items not yet filed, which includes any being filed at that moment.
func (b *CaptureBatch) OpenItemCount() int {
	if b.Items == nil {
		return max(b.ItemCount-b.FiledItemCount, 0)
	}

	open := 0
	for _, item := range b.Items {
		if item.Status.Open() {
			open++
		}
	}

	return open
}

// SettleFiling rolls the batch forward from its items' counts. It is the one
// place the counts are read, so the list and the batch cannot disagree about
// what "filed" means.
func (b *CaptureBatch) SettleFiling() {
	if b.Status.Terminal() || b.ItemCount == 0 {
		return
	}

	switch {
	case b.FiledItemCount >= b.ItemCount:
		b.Status = BatchFiled
	case b.FiledItemCount > 0:
		b.Status = BatchPartiallyFiled
	}
}

func (b *CaptureBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("cbat_")
		}
		if b.Status == "" {
			b.Status = BatchReceiving
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

func (b *CaptureBatch) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cbat",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "job_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "source_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (b *CaptureBatch) GetID() pulid.ID      { return b.ID }
func (b *CaptureBatch) GetCreatedAt() int64  { return b.CreatedAt }
func (b *CaptureBatch) GetTableName() string { return "capture_batches" }
