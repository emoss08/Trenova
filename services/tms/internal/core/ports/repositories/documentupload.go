package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentupload"
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

type ListDocumentUploadReconciliationRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	StaleBefore   int64                 `json:"staleBefore"`
	ExpiresBefore int64                 `json:"expiresBefore"`
	Limit         int                   `json:"limit"`
}

type DocumentUploadSessionRepository interface {
	Create(
		ctx context.Context,
		entity *documentupload.DocumentUploadSession,
	) (*documentupload.DocumentUploadSession, error)
	Update(
		ctx context.Context,
		entity *documentupload.DocumentUploadSession,
	) (*documentupload.DocumentUploadSession, error)
	GetByID(
		ctx context.Context,
		req GetDocumentUploadSessionByIDRequest,
	) (*documentupload.DocumentUploadSession, error)
	ListForReconciliation(
		ctx context.Context,
		req *ListDocumentUploadReconciliationRequest,
	) ([]*documentupload.DocumentUploadSession, error)
	ListReconciliationTenants(
		ctx context.Context,
		req *ListDocumentUploadReconciliationRequest,
	) ([]pagination.TenantInfo, error)
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
	) ([]*documentupload.DocumentUploadSession, error)
	ListRelated(
		ctx context.Context,
		req *ListRelatedDocumentUploadSessionsRequest,
	) ([]*documentupload.DocumentUploadSession, error)
}
