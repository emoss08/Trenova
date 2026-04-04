package document

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Document)(nil)
	_ domaintypes.PostgresSearchable = (*Document)(nil)
)

type Status string
type PreviewStatus string
type ContentStatus string
type ShipmentDraftStatus string
type ProcessingProfile string

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

func (p ProcessingProfile) IsValid() bool {
	switch p {
	case ProcessingProfileNone, ProcessingProfileRateConfirmationImport:
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
	return p == ProcessingProfileRateConfirmationImport
}

func SupportsPreview(fileType string) bool {
	fileType = strings.ToLower(fileType)
	return strings.HasPrefix(fileType, "image/") || fileType == "application/pdf"
}

type Document struct {
	bun.BaseModel `bun:"table:documents,alias:doc" json:"-"`

	ID                    pulid.ID            `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID            `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID        pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	LineageID             pulid.ID            `json:"lineageId"          bun:"lineage_id,type:VARCHAR(100),notnull"`
	VersionNumber         int64               `json:"versionNumber"      bun:"version_number,type:BIGINT,notnull,default:1"`
	IsCurrentVersion      bool                `json:"isCurrentVersion"   bun:"is_current_version,type:BOOLEAN,notnull,default:true"`
	FileName              string              `json:"fileName"           bun:"file_name,type:VARCHAR(255),notnull"`
	OriginalName          string              `json:"originalName"       bun:"original_name,type:VARCHAR(255),notnull"`
	FileSize              int64               `json:"fileSize"           bun:"file_size,type:BIGINT,notnull"`
	FileType              string              `json:"fileType"           bun:"file_type,type:VARCHAR(100),notnull"`
	StoragePath           string              `json:"storagePath"        bun:"storage_path,type:VARCHAR(500),notnull"`
	ChecksumSHA256        string              `json:"checksumSha256"     bun:"checksum_sha256,type:VARCHAR(64),nullzero"`
	StorageVersionID      string              `json:"storageVersionId"   bun:"storage_version_id,type:VARCHAR(255),nullzero"`
	StorageRetentionMode  string              `json:"storageRetentionMode" bun:"storage_retention_mode,type:VARCHAR(50),nullzero"`
	StorageRetentionUntil *int64              `json:"storageRetentionUntil" bun:"storage_retention_until,type:BIGINT,nullzero"`
	StorageLegalHold      bool                `json:"storageLegalHold"   bun:"storage_legal_hold,type:BOOLEAN,notnull,default:false"`
	Status                Status              `json:"status"             bun:"status,type:document_status_enum,notnull,default:'Active'"`
	Description           string              `json:"description"        bun:"description,type:TEXT,nullzero"`
	ResourceID            string              `json:"resourceId"         bun:"resource_id,type:VARCHAR(100),notnull"`
	ResourceType          string              `json:"resourceType"       bun:"resource_type,type:VARCHAR(100),notnull"`
	ProcessingProfile     ProcessingProfile   `json:"processingProfile"  bun:"processing_profile,type:VARCHAR(64),notnull,default:'none'"`
	ExpirationDate        *int64              `json:"expirationDate"     bun:"expiration_date,type:BIGINT,nullzero"`
	Tags                  []string            `json:"tags"               bun:"tags,type:VARCHAR(100)[],default:'{}'"`
	IsPublic              bool                `json:"isPublic"           bun:"is_public,type:BOOLEAN,notnull,default:false"`
	UploadedByID          pulid.ID            `json:"uploadedById"       bun:"uploaded_by_id,type:VARCHAR(100),notnull"`
	ApprovedByID          pulid.ID            `json:"approvedById"       bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt            *int64              `json:"approvedAt"         bun:"approved_at,type:BIGINT,nullzero"`
	PreviewStoragePath    string              `json:"previewStoragePath" bun:"preview_storage_path,type:VARCHAR(500),nullzero"`
	PreviewStatus         PreviewStatus       `json:"previewStatus"      bun:"preview_status,type:document_preview_status_enum,notnull,nullzero,default:'Unsupported'"`
	ContentStatus         ContentStatus       `json:"contentStatus"      bun:"content_status,type:document_content_status_enum,notnull,nullzero,default:'Pending'"`
	ContentError          string              `json:"contentError"       bun:"content_error,type:TEXT,nullzero"`
	DetectedKind          string              `json:"detectedKind"       bun:"detected_kind,type:VARCHAR(100),nullzero"`
	HasExtractedText      bool                `json:"hasExtractedText"   bun:"has_extracted_text,type:BOOLEAN,notnull,default:false"`
	ShipmentDraftStatus   ShipmentDraftStatus `json:"shipmentDraftStatus" bun:"shipment_draft_status,type:document_shipment_draft_status_enum,notnull,nullzero,default:'Unavailable'"`
	DocumentTypeID        *pulid.ID           `json:"documentTypeId"     bun:"document_type_id,type:VARCHAR(100),nullzero"`
	SearchVector          string              `json:"-"                  bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                  string              `json:"-"                  bun:"rank,type:VARCHAR(100),scanonly"`
	Version               int64               `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt             int64               `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64               `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (d *Document) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("doc_")
		}
		if d.LineageID.IsNil() {
			d.LineageID = d.ID
		}
		if d.VersionNumber == 0 {
			d.VersionNumber = 1
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
		if d.ContentStatus == "" {
			d.ContentStatus = ContentStatusPending
		}
		if d.ShipmentDraftStatus == "" {
			d.ShipmentDraftStatus = ShipmentDraftStatusUnavailable
		}
		if d.ProcessingProfile == "" {
			d.ProcessingProfile = ProcessingProfileNone
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
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
