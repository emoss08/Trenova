package services

import (
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateSessionRequest struct {
	TenantInfo        pagination.TenantInfo
	ResourceID        string
	ResourceType      string
	ProcessingProfile string
	FileName          string
	FileSize          int64
	ContentType       string
	Description       string
	Tags              []string
	DocumentTypeID    string
	LineageID         string
}

type PartRequest struct {
	TenantInfo   pagination.TenantInfo
	SessionID    pulid.ID
	PartNumbers  []int
	ResourceID   string
	ResourceType string
}

type CompletionRequest struct {
	TenantInfo pagination.TenantInfo
	SessionID  pulid.ID
}

type CancelRequest struct {
	TenantInfo pagination.TenantInfo
	SessionID  pulid.ID
}

type PartUploadTarget struct {
	PartNumber int    `json:"partNumber"`
	URL        string `json:"url"`
}

type SessionState struct {
	Session *documentupload.DocumentUploadSession `json:"session"`
	Parts   []storage.UploadedPart                `json:"parts"`
}
