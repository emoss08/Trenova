package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentsRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	ResourceID   string                   `json:"resourceId"`
	ResourceType string                   `json:"resourceType"`
	Status       string                   `json:"status"`
}

type GetDocumentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetDocumentsByResourceRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceID   string                `json:"resourceId"`
	ResourceType string                `json:"resourceType"`
}

type DeleteDocumentRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type UpdateDocumentPreviewRequest struct {
	ID                 pulid.ID               `json:"id"`
	TenantInfo         pagination.TenantInfo  `json:"tenantInfo"`
	PreviewStatus      document.PreviewStatus `json:"previewStatus"`
	PreviewStoragePath string                 `json:"previewStoragePath"`
}

type UpdateDocumentIntelligenceRequest struct {
	ID                  pulid.ID                     `json:"id"`
	TenantInfo          pagination.TenantInfo        `json:"tenantInfo"`
	ContentStatus       document.ContentStatus       `json:"contentStatus"`
	ContentError        string                       `json:"contentError"`
	DetectedKind        string                       `json:"detectedKind"`
	HasExtractedText    bool                         `json:"hasExtractedText"`
	ShipmentDraftStatus document.ShipmentDraftStatus `json:"shipmentDraftStatus"`
	DocumentTypeID      *pulid.ID                    `json:"documentTypeId"`
}

type ListDocumentVersionsRequest struct {
	LineageID   pulid.ID              `json:"lineageId"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
}

type PromoteDocumentVersionRequest struct {
	LineageID        pulid.ID              `json:"lineageId"`
	CurrentDocumentID pulid.ID             `json:"currentDocumentId"`
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteDocumentLineageRequest struct {
	LineageIDs  []pulid.ID            `json:"lineageIds"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
}

type BulkDeleteDocumentRequest struct {
	IDs        []pulid.ID            `json:"ids"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DocumentRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentsRequest,
	) (*pagination.ListResult[*document.Document], error)
	GetByID(ctx context.Context, req GetDocumentByIDRequest) (*document.Document, error)
	GetByIDs(ctx context.Context, req BulkDeleteDocumentRequest) ([]*document.Document, error)
	ListVersions(ctx context.Context, req ListDocumentVersionsRequest) ([]*document.Document, error)
	GetByResourceID(
		ctx context.Context,
		req *GetDocumentsByResourceRequest,
	) ([]*document.Document, error)
	ListPendingPreviewReconciliation(
		ctx context.Context,
		olderThan int64,
		limit int,
	) ([]*document.Document, error)
	Create(ctx context.Context, entity *document.Document) (*document.Document, error)
	Update(ctx context.Context, entity *document.Document) (*document.Document, error)
	UpdatePreview(ctx context.Context, req *UpdateDocumentPreviewRequest) error
	UpdateIntelligence(ctx context.Context, req *UpdateDocumentIntelligenceRequest) error
	PromoteVersion(ctx context.Context, req *PromoteDocumentVersionRequest) error
	Delete(ctx context.Context, req DeleteDocumentRequest) error
	BulkDelete(ctx context.Context, req BulkDeleteDocumentRequest) error
	DeleteByLineageIDs(ctx context.Context, req DeleteDocumentLineageRequest) error
}
