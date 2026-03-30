package documentupload

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusInitiated   Status = "Initiated"
	StatusUploading   Status = "Uploading"
	StatusUploaded    Status = "Uploaded"
	StatusVerifying   Status = "Verifying"
	StatusFinalizing  Status = "Finalizing"
	StatusPaused      Status = "Paused"
	StatusCompleting  Status = "Completing"
	StatusCompleted   Status = "Completed"
	StatusAvailable   Status = "Available"
	StatusQuarantined Status = "Quarantined"
	StatusFailed      Status = "Failed"
	StatusCanceled    Status = "Canceled"
	StatusExpired     Status = "Expired"
)

type Strategy string

const (
	StrategySingle    Strategy = "single"
	StrategyMultipart Strategy = "multipart"
)

type UploadedPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

type Session struct {
	bun.BaseModel `bun:"table:document_upload_sessions,alias:dus" json:"-"`

	ID                      pulid.ID       `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID          pulid.ID       `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID          pulid.ID       `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	DocumentID              *pulid.ID      `json:"documentId"              bun:"document_id,type:VARCHAR(100),nullzero"`
	LineageID               *pulid.ID      `json:"lineageId"               bun:"lineage_id,type:VARCHAR(100),nullzero"`
	ResourceID              string         `json:"resourceId"              bun:"resource_id,type:VARCHAR(100),notnull"`
	ResourceType            string         `json:"resourceType"            bun:"resource_type,type:VARCHAR(100),notnull"`
	DocumentTypeID          *pulid.ID      `json:"documentTypeId"          bun:"document_type_id,type:VARCHAR(100),nullzero"`
	OriginalName            string         `json:"originalName"            bun:"original_name,type:VARCHAR(255),notnull"`
	ContentType             string         `json:"contentType"             bun:"content_type,type:VARCHAR(255),notnull"`
	FileSize                int64          `json:"fileSize"                bun:"file_size,type:BIGINT,notnull"`
	StoragePath             string         `json:"storagePath"             bun:"storage_path,type:VARCHAR(500),notnull"`
	StorageProviderUploadID string         `json:"storageProviderUploadId" bun:"storage_provider_upload_id,type:VARCHAR(255),nullzero"`
	Strategy                Strategy       `json:"strategy"                bun:"strategy,type:VARCHAR(20),notnull"`
	Status                  Status         `json:"status"                  bun:"status,type:document_upload_session_status_enum,notnull,default:'Initiated'"`
	Description             string         `json:"description"             bun:"description,type:TEXT,nullzero"`
	Tags                    []string       `json:"tags"                    bun:"tags,type:VARCHAR(100)[],default:'{}'"`
	UploadedParts           []UploadedPart `json:"uploadedParts"           bun:"uploaded_parts,type:JSONB,notnull,default:'[]'::jsonb"`
	PartSize                int64          `json:"partSize"                bun:"part_size,type:BIGINT,notnull,default:0"`
	FailureCode             string         `json:"failureCode"             bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage          string         `json:"failureMessage"          bun:"failure_message,type:TEXT,nullzero"`
	ExpiresAt               int64          `json:"expiresAt"               bun:"expires_at,type:BIGINT,notnull"`
	LastActivityAt          int64          `json:"lastActivityAt"          bun:"last_activity_at,type:BIGINT,notnull"`
	Version                 int64          `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64          `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64          `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *Session) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("dus_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
		if s.LastActivityAt == 0 {
			s.LastActivityAt = now
		}
	case *bun.UpdateQuery:
		s.UpdatedAt = now
		if s.LastActivityAt == 0 {
			s.LastActivityAt = now
		}
	}

	return nil
}

func (s *Session) GetTableName() string {
	return "document_upload_sessions"
}

func (s Status) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusAvailable, StatusQuarantined, StatusFailed, StatusCanceled, StatusExpired:
		return true
	default:
		return false
	}
}

func (s *Session) MarkSuperseded(now int64) {
	s.Status = StatusCanceled
	s.FailureCode = "SUPERSEDED_BY_NEWER_SESSION"
	s.FailureMessage = "Superseded by a newer upload session"
	s.LastActivityAt = now
}

func (s *Session) IsSupersededByNewerArtifacts(
	activeSessions []*Session,
	versions []*document.Document,
) bool {
	if s == nil || s.LineageID == nil || s.LineageID.IsNil() {
		return false
	}

	for _, candidate := range activeSessions {
		if candidate == nil || candidate.ID == s.ID || candidate.LineageID == nil || *candidate.LineageID != *s.LineageID {
			continue
		}
		if candidate.IsNewerThan(s) {
			return true
		}
	}

	for _, version := range versions {
		if version == nil || version.StoragePath == s.StoragePath {
			continue
		}
		if version.CreatedAt > s.CreatedAt {
			return true
		}
	}

	return false
}

func (s *Session) IsNewerThan(other *Session) bool {
	if s == nil || other == nil {
		return false
	}

	if s.CreatedAt != other.CreatedAt {
		return s.CreatedAt > other.CreatedAt
	}

	return s.ID.String() > other.ID.String()
}
