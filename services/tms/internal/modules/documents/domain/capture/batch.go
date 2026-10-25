package capture

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"capture_batches"`

	ID                pulid.ID    `json:"id"                keel:"id,pk,autoid:cbat_"`
	BusinessUnitID    pulid.ID    `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID    `json:"organizationId"    keel:"organization_id,pk,tenant"`
	UserID            pulid.ID    `json:"userId"            keel:"user_id"`
	DeviceID          pulid.ID    `json:"deviceId"          keel:"device_id"`
	RequestID         *pulid.ID   `json:"requestId"         keel:"request_id"`
	ProfileID         *pulid.ID   `json:"profileId"         keel:"profile_id"`
	ClientKey         string      `json:"-"                 keel:"client_key"`
	Source            Source      `json:"source"            keel:"source"`
	Status            BatchStatus `json:"status"            keel:"status"`
	SourceName        string      `json:"sourceName"        keel:"source_name,nullzero"`
	JobName           string      `json:"jobName"           keel:"job_name,nullzero"`
	Settings          Settings    `json:"settings"          keel:"settings,jsonb"`
	TargetType        string      `json:"targetType"        keel:"target_type,nullzero"`
	TargetID          *pulid.ID   `json:"targetId"          keel:"target_id"`
	DocumentTypeID    *pulid.ID   `json:"documentTypeId"    keel:"document_type_id"`
	ExpectedPageCount int         `json:"expectedPageCount" keel:"expected_page_count"`
	ReceivedPageCount int         `json:"receivedPageCount" keel:"received_page_count"`
	ItemCount         int         `json:"itemCount"         keel:"item_count"`
	FiledItemCount    int         `json:"filedItemCount"    keel:"filed_item_count"`
	ManifestDigest    string      `json:"-"                 keel:"manifest_digest,nullzero"`
	FailureMessage    string      `json:"failureMessage"    keel:"failure_message,nullzero"`
	SealedAt          *int64      `json:"sealedAt"          keel:"sealed_at"`
	ProcessedAt       *int64      `json:"processedAt"       keel:"processed_at"`
	RetainUntil       int64       `json:"retainUntil"       keel:"retain_until"`
	// RetentionRemindedAt is when the owner was told the unfiled pages are
	// about to go. It is set once, by whichever sweep claims the reminder.
	RetentionRemindedAt *int64         `json:"-"               keel:"retention_reminded_at"`
	Version             int64          `json:"version"         keel:"version,version"`
	CreatedAt           int64          `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt           int64          `json:"updatedAt"       keel:"updated_at,autoupdate"`
	Pages               []*CapturePage `json:"pages,omitempty" keel:"many:batch_id"`
	Items               []*CaptureItem `json:"items,omitempty" keel:"many:batch_id"`

	CaptureDevice *CaptureDevice       `json:"device,omitempty"       keel:"belongs:device_id"`
	Organization  *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit  *tenant.BusinessUnit `json:"businessUnit,omitempty"`
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

func (b *CaptureBatch) BeforeCreate(ctx context.Context) error {
	if b.Status == "" {
		b.Status = BatchReceiving
	}

	return nil
}
