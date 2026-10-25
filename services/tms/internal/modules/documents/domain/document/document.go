package document

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable = (*Document)(nil)
)

type Status string
type PreviewStatus string
type ContentStatus string
type ShipmentDraftStatus string
type ProcessingProfile string

// SignatureStatus is whether a document carries the signature its type needs.
// It stays Unknown until extraction reads the signature line or a person says.
type SignatureStatus string

const (
	SignatureStatusUnknown  SignatureStatus = "Unknown"
	SignatureStatusSigned   SignatureStatus = "Signed"
	SignatureStatusUnsigned SignatureStatus = "Unsigned"
)

const (
	StatusDraft           Status = "Draft"
	StatusActive          Status = "Active"
	StatusArchived        Status = "Archived"
	StatusExpired         Status = "Expired"
	StatusPending         Status = "Pending"
	StatusRejected        Status = "Rejected"
	StatusPendingApproval Status = "PendingApproval"
)

const (
	PreviewStatusPending     PreviewStatus = "Pending"
	PreviewStatusReady       PreviewStatus = "Ready"
	PreviewStatusFailed      PreviewStatus = "Failed"
	PreviewStatusUnsupported PreviewStatus = "Unsupported"
)

const (
	ContentStatusPending    ContentStatus = "Pending"
	ContentStatusExtracting ContentStatus = "Extracting"
	ContentStatusExtracted  ContentStatus = "Extracted"
	ContentStatusIndexed    ContentStatus = "Indexed"
	ContentStatusFailed     ContentStatus = "Failed"
)

const (
	ShipmentDraftStatusUnavailable ShipmentDraftStatus = "Unavailable"
	ShipmentDraftStatusPending     ShipmentDraftStatus = "Pending"
	ShipmentDraftStatusReady       ShipmentDraftStatus = "Ready"
	ShipmentDraftStatusFailed      ShipmentDraftStatus = "Failed"
)

const (
	ProcessingProfileNone                   ProcessingProfile = "none"
	ProcessingProfileRateConfirmationImport ProcessingProfile = "rate_confirmation_import"
	ProcessingProfileAssistantAttachment    ProcessingProfile = "assistant_attachment"
	// ProcessingProfileInboundAttachment is a file that came in on a monitored
	// mailbox. It is its own profile rather than borrowing the rate-confirmation
	// one because a POD, an invoice and a tender all arrive this way, and
	// labelling them all a rate confirmation would be a lie the pipeline then
	// acts on.
	ProcessingProfileInboundAttachment ProcessingProfile = "inbound_attachment"
	// ProcessingProfileCapture is paper scanned or printed into Trenova. It is
	// read like an inbound attachment: what a scan is, and whose it is, is only
	// known once its text has been read.
	ProcessingProfileCapture ProcessingProfile = "capture"
	// ProcessingProfileFactoringRemittance is a factor's PDF remittance. Its
	// text is read and indexed so billing can read its lines, but it is never
	// handed to the shipment extraction model: that model, its shadow runs and
	// its training data are for freight paperwork, not a factor's statement.
	ProcessingProfileFactoringRemittance ProcessingProfile = "factoring_remittance"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusActive,
		StatusArchived,
		StatusExpired,
		StatusPending,
		StatusRejected,
		StatusPendingApproval:
		return true
	}
	return false
}

func (s PreviewStatus) IsValid() bool {
	switch s {
	case PreviewStatusPending,
		PreviewStatusReady,
		PreviewStatusFailed,
		PreviewStatusUnsupported:
		return true
	}
	return false
}

func (s ContentStatus) IsValid() bool {
	switch s {
	case ContentStatusPending,
		ContentStatusExtracting,
		ContentStatusExtracted,
		ContentStatusIndexed,
		ContentStatusFailed:
		return true
	}
	return false
}

func (s ShipmentDraftStatus) IsValid() bool {
	switch s {
	case ShipmentDraftStatusUnavailable,
		ShipmentDraftStatusPending,
		ShipmentDraftStatusReady,
		ShipmentDraftStatusFailed:
		return true
	}
	return false
}

// AllProcessingProfiles is the whole set, so a caller that has to reason about
// which of them do what reads the list rather than repeating it.
func AllProcessingProfiles() []ProcessingProfile {
	return []ProcessingProfile{
		ProcessingProfileNone,
		ProcessingProfileRateConfirmationImport,
		ProcessingProfileAssistantAttachment,
		ProcessingProfileInboundAttachment,
		ProcessingProfileCapture,
		ProcessingProfileFactoringRemittance,
	}
}

// IntelligenceProcessingProfiles are the ones a document is read under. It is
// derived from SupportsIntelligence rather than listed again, so the two cannot
// disagree about which documents get opened.
func IntelligenceProcessingProfiles() []ProcessingProfile {
	profiles := make([]ProcessingProfile, 0, len(AllProcessingProfiles()))
	for _, profile := range AllProcessingProfiles() {
		if profile.SupportsIntelligence() {
			profiles = append(profiles, profile)
		}
	}

	return profiles
}

func (p ProcessingProfile) IsValid() bool {
	switch p {
	case ProcessingProfileNone,
		ProcessingProfileRateConfirmationImport,
		ProcessingProfileAssistantAttachment,
		ProcessingProfileInboundAttachment,
		ProcessingProfileCapture,
		ProcessingProfileFactoringRemittance:
		return true
	}
	return false
}

func NormalizeProcessingProfile(raw string) (ProcessingProfile, error) {
	profile := ProcessingProfile(strings.TrimSpace(raw))
	if profile == "" {
		return ProcessingProfileNone, nil
	}
	if !profile.IsValid() {
		return "", errInvalidProcessingProfile(profile)
	}
	return profile, nil
}

func (p ProcessingProfile) SupportsIntelligence() bool {
	return p == ProcessingProfileRateConfirmationImport ||
		p == ProcessingProfileAssistantAttachment ||
		p == ProcessingProfileInboundAttachment ||
		p == ProcessingProfileCapture ||
		p == ProcessingProfileFactoringRemittance
}

// ExtractsWithAI reports whether a document read under this profile may be
// handed to the shipment extraction model once its text is read.
func (p ProcessingProfile) ExtractsWithAI() bool {
	return p.SupportsIntelligence() && p != ProcessingProfileFactoringRemittance
}

func SupportsPreview(fileType string) bool {
	fileType = strings.ToLower(fileType)
	return strings.HasPrefix(fileType, "image/") || fileType == "application/pdf"
}

type Document struct {
	keel.Table `keel:"documents"`

	ID                    pulid.ID            `json:"id"                    keel:"id,pk,autoid:doc_"`
	OrganizationID        pulid.ID            `json:"organizationId"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID            `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	LineageID             pulid.ID            `json:"lineageId"             keel:"lineage_id"`
	VersionNumber         int64               `json:"versionNumber"         keel:"version_number"`
	IsCurrentVersion      bool                `json:"isCurrentVersion"      keel:"is_current_version"`
	FileName              string              `json:"fileName"              keel:"file_name"`
	OriginalName          string              `json:"originalName"          keel:"original_name"`
	FileSize              int64               `json:"fileSize"              keel:"file_size"`
	FileType              string              `json:"fileType"              keel:"file_type"`
	StoragePath           string              `json:"storagePath"           keel:"storage_path"`
	ChecksumSHA256        string              `json:"checksumSha256"        keel:"checksum_sha256,nullzero"`
	StorageVersionID      string              `json:"storageVersionId"      keel:"storage_version_id,nullzero"`
	StorageRetentionMode  string              `json:"storageRetentionMode"  keel:"storage_retention_mode,nullzero"`
	StorageRetentionUntil *int64              `json:"storageRetentionUntil" keel:"storage_retention_until"`
	StorageLegalHold      bool                `json:"storageLegalHold"      keel:"storage_legal_hold"`
	CryptoMode            string              `json:"cryptoMode"            keel:"crypto_mode"`
	CryptoVersion         int16               `json:"cryptoVersion"         keel:"crypto_version"`
	Status                Status              `json:"status"                keel:"status"`
	Description           string              `json:"description"           keel:"description,nullzero"`
	ResourceID            string              `json:"resourceId"            keel:"resource_id"`
	ResourceType          string              `json:"resourceType"          keel:"resource_type"`
	ProcessingProfile     ProcessingProfile   `json:"processingProfile"     keel:"processing_profile"`
	ExpirationDate        *int64              `json:"expirationDate"        keel:"expiration_date"`
	Tags                  []string            `json:"tags"                  keel:"tags"`
	IsPublic              bool                `json:"isPublic"              keel:"is_public"`
	UploadedByID          pulid.ID            `json:"uploadedById"          keel:"uploaded_by_id"`
	ApprovedByID          pulid.ID            `json:"approvedById"          keel:"approved_by_id,nullzero"`
	ApprovedAt            *int64              `json:"approvedAt"            keel:"approved_at"`
	RejectedByID          pulid.ID            `json:"rejectedById"          keel:"rejected_by_id,nullzero"`
	RejectedAt            *int64              `json:"rejectedAt"            keel:"rejected_at"`
	RejectionReason       string              `json:"rejectionReason"       keel:"rejection_reason,nullzero"`
	PreviewStoragePath    string              `json:"previewStoragePath"    keel:"preview_storage_path,nullzero"`
	PreviewStatus         PreviewStatus       `json:"previewStatus"         keel:"preview_status,nullzero"`
	ContentStatus         ContentStatus       `json:"contentStatus"         keel:"content_status,nullzero"`
	ContentError          string              `json:"contentError"          keel:"content_error,nullzero"`
	DetectedKind          string              `json:"detectedKind"          keel:"detected_kind,nullzero"`
	HasExtractedText      bool                `json:"hasExtractedText"      keel:"has_extracted_text"`
	ShipmentDraftStatus   ShipmentDraftStatus `json:"shipmentDraftStatus"   keel:"shipment_draft_status,nullzero"`
	DocumentTypeID        *pulid.ID           `json:"documentTypeId"        keel:"document_type_id"`
	SignatureStatus       SignatureStatus     `json:"signatureStatus"       keel:"signature_status,nullzero"`
	SignedAt              *int64              `json:"signedAt"              keel:"signed_at"`
	SignatureSource       SignatureSource     `json:"signatureSource"       keel:"signature_source,nullzero"`
	SignatureUpdatedByID  pulid.ID            `json:"signatureUpdatedById"  keel:"signature_updated_by_id,nullzero"`
	SignatureUpdatedAt    *int64              `json:"signatureUpdatedAt"    keel:"signature_updated_at"`
	SearchVector          string              `json:"-"                     keel:"search_vector,tsvector,readonly,noselect"`
	Rank                  string              `json:"-"                     keel:"rank,computed"`
	Version               int64               `json:"version"               keel:"version,version"`
	CreatedAt             int64               `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64               `json:"updatedAt"             keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit       `json:"businessUnit,omitempty"`
	Organization *tenant.Organization       `json:"organization,omitempty"`
	DocumentType *documenttype.DocumentType `json:"documentType,omitempty" keel:"belongs:document_type_id"`
}

func (d *Document) BeforeCreate(ctx context.Context) error {
	if d.VersionNumber == 0 {
		d.VersionNumber = 1
	}
	if d.CryptoMode == "" {
		d.CryptoMode = "blob_v1"
	}
	if d.CryptoVersion == 0 {
		d.CryptoVersion = 2
	}
	if d.Status == "" {
		d.Status = StatusActive
	}
	if d.ProcessingProfile == "" {
		d.ProcessingProfile = ProcessingProfileNone
	}
	if d.ContentStatus == "" {
		d.ContentStatus = ContentStatusPending
	}
	if d.ShipmentDraftStatus == "" {
		d.ShipmentDraftStatus = ShipmentDraftStatusUnavailable
	}
	if d.SignatureStatus == "" {
		d.SignatureStatus = SignatureStatusUnknown
	}
	if d.LineageID.IsNil() {
		d.LineageID = d.ID
	}
	if d.PreviewStatus == "" {
		switch {
		case d.PreviewStoragePath != "":
			d.PreviewStatus = PreviewStatusReady
		case SupportsPreview(d.FileType):
			d.PreviewStatus = PreviewStatusPending
		default:
			d.PreviewStatus = PreviewStatusUnsupported
		}
	}

	return nil
}

func (d *Document) GetID() pulid.ID {
	return d.ID
}

func (d *Document) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *Document) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *Document) GetTableName() string {
	return "documents"
}

func (d *Document) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "doc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "original_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (d *Document) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// Versions of one document share a lineage; without it a new version
		// would present as an unrelated document and the old one would stay
		// current.
		validation.Field(&d.LineageID, validation.Required.Error("Lineage is required")),
		validation.Field(&d.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(int64(1)).Error("Version number must be at least one"),
		),
		validation.Field(&d.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, maxFileNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&d.OriginalName,
			validation.Required.Error("Original name is required"),
			validation.Length(1, maxFileNameLength).
				Error("Original name cannot be longer than 255 characters"),
		),
		validation.Field(&d.FileSize,
			validation.Required.Error("File size is required"),
			validation.Min(int64(1)).Error("File size must be greater than zero"),
		),
		validation.Field(&d.FileType,
			validation.Required.Error("File type is required"),
			validation.Length(1, maxFileTypeLength).
				Error("File type cannot be longer than 100 characters"),
		),
		validation.Field(&d.StoragePath,
			validation.Required.Error("Storage path is required"),
			validation.Length(1, maxStoragePathLength).
				Error("Storage path cannot be longer than 500 characters"),
		),
		validation.Field(&d.ChecksumSHA256,
			validation.Length(checksumLength, checksumLength).
				Error("Checksum must be a 64 character digest"),
		),
		validation.Field(&d.StorageVersionID,
			validation.Length(0, maxStorageVersionLength).
				Error("Storage version cannot be longer than 255 characters"),
		),
		validation.Field(&d.CryptoMode,
			validation.Required.Error("Crypto mode is required"),
			validation.Length(1, maxCryptoModeLength).
				Error("Crypto mode cannot be longer than 32 characters"),
		),
		validation.Field(&d.CryptoVersion,
			validation.Required.Error("Crypto version is required"),
			validation.Min(int16(1)).Error("Crypto version must be at least one"),
		),
		validation.Field(&d.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		// A document with no owning resource is unreachable from every surface
		// that lists documents.
		validation.Field(&d.ResourceID,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxDocumentResourceLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		validation.Field(&d.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxDocumentResourceLength).
				Error("Resource type cannot be longer than 100 characters"),
		),
		validation.Field(&d.ProcessingProfile,
			validation.Required.Error("Processing profile is required"),
			domainvalidation.ValidEnum[ProcessingProfile]("Processing profile is invalid"),
		),
		validation.Field(&d.PreviewStatus,
			domainvalidation.ValidEnum[PreviewStatus]("Preview status is invalid"),
		),
		validation.Field(&d.ContentStatus,
			domainvalidation.ValidEnum[ContentStatus]("Content status is invalid"),
		),
		validation.Field(&d.ShipmentDraftStatus,
			domainvalidation.ValidEnum[ShipmentDraftStatus]("Shipment draft status is invalid"),
		),
		validation.Field(&d.SignatureStatus,
			domainvalidation.ValidEnum[SignatureStatus]("Signature status is invalid"),
		),
		validation.Field(&d.SignatureSource,
			domainvalidation.ValidEnum[SignatureSource]("Signature source is invalid"),
		),
		validation.Field(&d.SignedAt,
			validation.When(
				d.SignatureStatus != SignatureStatusSigned,
				validation.Nil.Error("Only a signed document has a signing date"),
			),
		),
		validation.Field(&d.SignatureUpdatedByID,
			validation.When(
				d.SignatureSource == SignatureSourcePerson,
				validation.Required.Error(
					"A signature a person recorded must name who recorded it",
				),
			),
		),
		validation.Field(&d.PreviewStoragePath,
			validation.Length(0, maxStoragePathLength).
				Error("Preview storage path cannot be longer than 500 characters"),
		),
		validation.Field(&d.Tags,
			validation.Each(validation.Length(1, maxDocumentTagLength).
				Error("A tag must be between 1 and 100 characters")),
		),
		validation.Field(&d.UploadedByID, validation.Required.Error("Uploaded by is required")),
		// An approval is an accountability record: who approved it and when are
		// both part of it or neither is.
		validation.Field(&d.ApprovedAt,
			validation.When(d.ApprovedByID.IsNotNil(), validation.Required.Error(
				"An approved document must record when it was approved",
			)),
		),
		validation.Field(&d.ApprovedByID,
			validation.When(d.ApprovedAt != nil, validation.Required.Error(
				"An approved document must record who approved it",
			)),
		),
		validation.Field(&d.RejectedAt,
			validation.When(d.Status == StatusRejected, validation.Required.Error(
				"A rejected document must record when it was rejected",
			)),
		),
		validation.Field(&d.RejectionReason,
			validation.When(d.Status == StatusRejected, validation.Required.Error(
				"A rejected document must record why it was rejected",
			)),
			validation.Length(0, MaxRejectionReasonLength).
				Error("Rejection reason cannot be longer than 1000 characters"),
		),
	))
}

const (
	maxFileNameLength         = 255
	maxFileTypeLength         = 100
	maxStoragePathLength      = 500
	maxStorageVersionLength   = 255
	maxCryptoModeLength       = 32
	maxDocumentResourceLength = 100
	maxDocumentTagLength      = 100
	checksumLength            = 64
)
