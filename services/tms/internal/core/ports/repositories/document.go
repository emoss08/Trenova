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
	GetByResourceID(
		ctx context.Context,
		req *GetDocumentsByResourceRequest,
	) ([]*document.Document, error)
	Create(ctx context.Context, entity *document.Document) (*document.Document, error)
	Update(ctx context.Context, entity *document.Document) (*document.Document, error)
	Delete(ctx context.Context, req DeleteDocumentRequest) error
	BulkDelete(ctx context.Context, req BulkDeleteDocumentRequest) error
}
