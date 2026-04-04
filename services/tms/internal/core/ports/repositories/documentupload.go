package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDocumentUploadSessionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListActiveDocumentUploadSessionsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceID   string                `json:"resourceId"`
	ResourceType string                `json:"resourceType"`
}

type ListRelatedDocumentUploadSessionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	DocumentID pulid.ID              `json:"documentId"`
	LineageID  pulid.ID              `json:"lineageId"`
}

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
	Session *documentupload.Session `json:"session"`
	Parts   []storage.UploadedPart  `json:"parts"`
}
type DocumentUploadSessionRepository interface {
	Create(ctx context.Context, entity *documentupload.Session) (*documentupload.Session, error)
	Update(ctx context.Context, entity *documentupload.Session) (*documentupload.Session, error)
	GetByID(
		ctx context.Context,
		req GetDocumentUploadSessionByIDRequest,
	) (*documentupload.Session, error)
	ListForReconciliation(
		ctx context.Context,
		staleBefore int64,
		expiresBefore int64,
		limit int,
	) ([]*documentupload.Session, error)
	ClearDocumentReference(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error
	ClearDocumentReferences(
		ctx context.Context,
		documentIDs []pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error
	ListActive(
		ctx context.Context,
		req *ListActiveDocumentUploadSessionsRequest,
	) ([]*documentupload.Session, error)
	ListRelated(
		ctx context.Context,
		req *ListRelatedDocumentUploadSessionsRequest,
	) ([]*documentupload.Session, error)
}
