package documentupload

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DocumentUploadSession)(nil)

type DocumentUploadSession struct {
	bun.BaseModel `bun:"table:document_upload_sessions,alias:dus" json:"-"`

	ID                      pulid.ID                   `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID          pulid.ID                   `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID          pulid.ID                   `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	DocumentID              *pulid.ID                  `json:"documentId"              bun:"document_id,type:VARCHAR(100),nullzero"`
	LineageID               *pulid.ID                  `json:"lineageId"               bun:"lineage_id,type:VARCHAR(100),nullzero"`
	ResourceID              string                     `json:"resourceId"              bun:"resource_id,type:VARCHAR(100),notnull"`
	ResourceType            string                     `json:"resourceType"            bun:"resource_type,type:VARCHAR(100),notnull"`
	ProcessingProfile       document.ProcessingProfile `json:"processingProfile"       bun:"processing_profile,type:VARCHAR(64),notnull,default:'none'"`
	DocumentTypeID          *pulid.ID                  `json:"documentTypeId"          bun:"document_type_id,type:VARCHAR(100),nullzero"`
	OriginalName            string                     `json:"originalName"            bun:"original_name,type:VARCHAR(255),notnull"`
	ContentType             string                     `json:"contentType"             bun:"content_type,type:VARCHAR(255),notnull"`
	FileSize                int64                      `json:"fileSize"                bun:"file_size,type:BIGINT,notnull"`
	StoragePath             string                     `json:"storagePath"             bun:"storage_path,type:VARCHAR(500),notnull"`
	StorageProviderUploadID string                     `json:"storageProviderUploadId" bun:"storage_provider_upload_id,type:VARCHAR(255),nullzero"`
	ChecksumSHA256          string                     `json:"checksumSha256"          bun:"checksum_sha256,type:VARCHAR(64),nullzero"`
	CryptoMode              string                     `json:"cryptoMode"              bun:"crypto_mode,type:VARCHAR(32),notnull,default:'envelope_v1'"`
	CryptoVersion           int16                      `json:"cryptoVersion"           bun:"crypto_version,type:SMALLINT,notnull,default:1"`
	Strategy                Strategy                   `json:"strategy"                bun:"strategy,type:VARCHAR(20),notnull"`
	Status                  Status                     `json:"status"                  bun:"status,type:document_upload_session_status_enum,notnull,default:'Initiated'"`
	Description             string                     `json:"description"             bun:"description,type:TEXT,nullzero"`
	Tags                    []string                   `json:"tags"                    bun:"tags,type:VARCHAR(100)[],default:'{}'"`
	UploadedParts           []storage.UploadedPart     `json:"uploadedParts"           bun:"uploaded_parts,type:JSONB,notnull,default:'[]'"`
	PartSize                int64                      `json:"partSize"                bun:"part_size,type:BIGINT,notnull,default:0"`
	FailureCode             string                     `json:"failureCode"             bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage          string                     `json:"failureMessage"          bun:"failure_message,type:TEXT,nullzero"`
	ExpiresAt               int64                      `json:"expiresAt"               bun:"expires_at,type:BIGINT,notnull"`
	LastActivityAt          int64                      `json:"lastActivityAt"          bun:"last_activity_at,type:BIGINT,notnull"`
	Version                 int64                      `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64                      `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64                      `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *DocumentUploadSession) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("dus_")
		}
		if s.ProcessingProfile == "" {
			s.ProcessingProfile = document.ProcessingProfileNone
		}
		if s.CryptoMode == "" {
			s.CryptoMode = "envelope_v1"
		}
		if s.CryptoVersion == 0 {
			s.CryptoVersion = 1
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

func (s *DocumentUploadSession) GetTableName() string {
	return "document_upload_sessions"
}

func (s *DocumentUploadSession) MarkSuperseded(now int64) {
	s.Status = StatusCanceled
	s.FailureCode = FailureCodeSuspendedByNewerSession.String()
	s.FailureMessage = "Superseded by a newer upload session"
	s.LastActivityAt = now
}

func (s *DocumentUploadSession) IsSupersededByNewerArtifacts(
	activeSessions []*DocumentUploadSession,
	versions []*document.Document,
) bool {
	if s == nil || s.LineageID == nil || s.LineageID.IsNil() {
		return false
	}

	for _, candidate := range activeSessions {
		if candidate == nil || candidate.ID == s.ID || candidate.LineageID == nil ||
			*candidate.LineageID != *s.LineageID {
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

func (s *DocumentUploadSession) IsNewerThan(other *DocumentUploadSession) bool {
	if s == nil || other == nil {
		return false
	}

	if s.CreatedAt != other.CreatedAt {
		return s.CreatedAt > other.CreatedAt
	}

	return s.ID.String() > other.ID.String()
}

func (s *DocumentUploadSession) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.ResourceID,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxSessionCodeLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		validation.Field(&s.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxSessionCodeLength).
				Error("Resource type cannot be longer than 100 characters"),
		),
		validation.Field(&s.ProcessingProfile,
			validation.Required.Error("Processing profile is required"),
			domainvalidation.ValidEnum[document.ProcessingProfile](
				"Processing profile is invalid",
			),
		),
		validation.Field(&s.OriginalName,
			validation.Required.Error("Original name is required"),
			validation.Length(1, maxSessionNameLength).
				Error("Original name cannot be longer than 255 characters"),
		),
		validation.Field(&s.ContentType,
			validation.Required.Error("Content type is required"),
			validation.Length(1, maxSessionNameLength).
				Error("Content type cannot be longer than 255 characters"),
		),
		validation.Field(&s.FileSize,
			validation.Required.Error("File size is required"),
			validation.Min(int64(1)).Error("File size must be greater than zero"),
		),
		validation.Field(&s.StoragePath,
			validation.Required.Error("Storage path is required"),
			validation.Length(1, maxSessionPathLength).
				Error("Storage path cannot be longer than 500 characters"),
		),
		validation.Field(&s.ChecksumSHA256,
			validation.Length(sessionChecksumLength, sessionChecksumLength).
				Error("Checksum must be a 64 character digest"),
		),
		validation.Field(&s.CryptoMode,
			validation.Required.Error("Crypto mode is required"),
			validation.Length(1, maxSessionCryptoModeLength).
				Error("Crypto mode cannot be longer than 32 characters"),
		),
		validation.Field(&s.CryptoVersion,
			validation.Required.Error("Crypto version is required"),
			validation.Min(int16(1)).Error("Crypto version must be at least one"),
		),
		validation.Field(&s.Strategy,
			validation.Required.Error("Strategy is required"),
			domainvalidation.ValidEnum[Strategy]("Strategy is invalid"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		// A multipart upload is resumed from its provider id; without one the
		// parts already uploaded are unreachable and the session is stuck.
		validation.Field(&s.StorageProviderUploadID,
			validation.When(
				s.Strategy == StrategyMultipart,
				validation.Required.Error("A multipart upload must record its provider id"),
			),
			validation.Length(0, maxSessionNameLength).
				Error("Provider upload id cannot be longer than 255 characters"),
		),
		validation.Field(&s.PartSize,
			validation.Min(int64(0)).Error("Part size cannot be negative"),
		),
		validation.Field(&s.Tags,
			validation.Each(validation.Length(1, maxSessionCodeLength).
				Error("A tag must be between 1 and 100 characters")),
		),
		validation.Field(&s.FailureCode,
			validation.Length(0, maxSessionCodeLength).
				Error("Failure code cannot be longer than 100 characters"),
		),
		// A session with no expiry would hold its storage reservation forever.
		validation.Field(&s.ExpiresAt,
			validation.Required.Error("Expiry is required"),
			validation.Min(int64(1)).Error("Expiry must be a valid timestamp"),
		),
	))
}

const (
	maxSessionCodeLength       = 100
	maxSessionNameLength       = 255
	maxSessionPathLength       = 500
	maxSessionCryptoModeLength = 32
	sessionChecksumLength      = 64
)
