package documentupload

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type DocumentUploadSession struct {
	keel.Table `keel:"document_upload_sessions"`

	ID                      pulid.ID                   `json:"id"                      keel:"id,pk,autoid:dus_"`
	OrganizationID          pulid.ID                   `json:"organizationId"          keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID                   `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	DocumentID              *pulid.ID                  `json:"documentId"              keel:"document_id"`
	LineageID               *pulid.ID                  `json:"lineageId"               keel:"lineage_id"`
	ResourceID              string                     `json:"resourceId"              keel:"resource_id"`
	ResourceType            string                     `json:"resourceType"            keel:"resource_type"`
	ProcessingProfile       document.ProcessingProfile `json:"processingProfile"       keel:"processing_profile"`
	DocumentTypeID          *pulid.ID                  `json:"documentTypeId"          keel:"document_type_id"`
	OriginalName            string                     `json:"originalName"            keel:"original_name"`
	ContentType             string                     `json:"contentType"             keel:"content_type"`
	FileSize                int64                      `json:"fileSize"                keel:"file_size"`
	StoragePath             string                     `json:"storagePath"             keel:"storage_path"`
	StorageProviderUploadID string                     `json:"storageProviderUploadId" keel:"storage_provider_upload_id,nullzero"`
	ChecksumSHA256          string                     `json:"checksumSha256"          keel:"checksum_sha256,nullzero"`
	CryptoMode              string                     `json:"cryptoMode"              keel:"crypto_mode"`
	CryptoVersion           int16                      `json:"cryptoVersion"           keel:"crypto_version"`
	MalwareScan             bool                       `json:"malwareScan"             keel:"malware_scan"`
	Strategy                Strategy                   `json:"strategy"                keel:"strategy"`
	Status                  Status                     `json:"status"                  keel:"status"`
	Description             string                     `json:"description"             keel:"description,nullzero"`
	Tags                    []string                   `json:"tags"                    keel:"tags"`
	UploadedParts           []kernelports.UploadedPart `json:"uploadedParts"           keel:"uploaded_parts,jsonb"`
	PartSize                int64                      `json:"partSize"                keel:"part_size"`
	FailureCode             string                     `json:"failureCode"             keel:"failure_code,nullzero"`
	FailureMessage          string                     `json:"failureMessage"          keel:"failure_message,nullzero"`
	ThreatName              string                     `json:"threatName"              keel:"threat_name,nullzero"`
	ExpiresAt               int64                      `json:"expiresAt"               keel:"expires_at"`
	LastActivityAt          int64                      `json:"lastActivityAt"          keel:"last_activity_at"`
	Version                 int64                      `json:"version"                 keel:"version,version"`
	CreatedAt               int64                      `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64                      `json:"updatedAt"               keel:"updated_at,autoupdate"`
}

func (s *DocumentUploadSession) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	s.applyDefaults(now)
	if s.ProcessingProfile == "" {
		s.ProcessingProfile = document.ProcessingProfileNone
	}
	if s.CryptoMode == "" {
		s.CryptoMode = "blob_v1"
	}
	if s.CryptoVersion == 0 {
		s.CryptoVersion = 2
	}
	if s.Status == "" {
		s.Status = StatusInitiated
	}
	s.applyDefaults(timeutils.NowUnix())

	return nil
}

func (s *DocumentUploadSession) BeforeUpdate(_ context.Context) error {
	now := timeutils.NowUnix()
	s.touch(now)
	s.touch(timeutils.NowUnix())

	return nil
}

func (s *DocumentUploadSession) applyDefaults(now int64) {
	if s.ProcessingProfile == "" {
		s.ProcessingProfile = document.ProcessingProfileNone
	}
	if s.CryptoMode == "" {
		s.CryptoMode = "blob_v1"
	}
	if s.CryptoVersion == 0 {
		s.CryptoVersion = 2
	}
	if s.Status == "" {
		s.Status = StatusInitiated
	}
	s.touch(now)
}

func (s *DocumentUploadSession) touch(now int64) {
	if s.LastActivityAt == 0 {
		s.LastActivityAt = now
	}
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
