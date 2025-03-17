package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetDocumentCountByResourceResponse struct {
	ResourceType permission.Resource `json:"resourceType"`
	// Count is the number of `sub-folders` aka unique resource ids
	Count        int   `json:"count"`
	TotalSize    int64 `json:"totalSize"`
	LastModified int64 `json:"lastModified"`
}

type GetResourceSubFoldersRequest struct {
	ResourceType permission.Resource `json:"resourceType"`
	ports.TenantOptions
}

type GetDocumentsByResourceIDRequest struct {
	Filter       *ports.LimitOffsetQueryOptions
	ResourceType permission.Resource `json:"resourceType"`
	ResourceID   string              `json:"resourceId"`
	ports.TenantOptions
}

type GetResourceSubFoldersResponse struct {
	// This value will change depending on the resource type
	FolderName string `json:"folderName"`
	// The number of documents in the sub-folder
	Count int `json:"count"`
	// The total size of the documents in the sub-folder
	TotalSize int64 `json:"totalSize"`
	// The last modified date of the documents in the sub-folder
	LastModified int64 `json:"lastModified"`
	// ResourceID is the ID of the resource that the sub-folder belongs to
	ResourceID string `json:"resourceId"`
}

// DeleteDocumentRequest contains options for deleting a document
type DeleteDocumentRequest struct {
	ID    pulid.ID
	OrgID pulid.ID
	BuID  pulid.ID
}

// DocumentRepository defines the interface for document data access
type DocumentRepository interface {
	// CRUD operations
	Create(ctx context.Context, doc *document.Document) (*document.Document, error)
	Update(ctx context.Context, doc *document.Document) (*document.Document, error)
	Delete(ctx context.Context, req DeleteDocumentRequest) error

	// List operations
	GetDocumentsByResourceID(ctx context.Context, req *GetDocumentsByResourceIDRequest) (*ports.ListResult[*document.Document], error)

	// Aggregation operations
	GetDocumentCountByResource(ctx context.Context, req *ports.TenantOptions) ([]*GetDocumentCountByResourceResponse, error)
	GetResourceSubFolders(ctx context.Context, req GetResourceSubFoldersRequest) ([]*GetResourceSubFoldersResponse, error)
}
