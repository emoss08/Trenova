package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// DocumentRequest contains shared options for document repository operations
type DocumentRequest struct {
	ExpandDocumentDetails bool
}

// GetDocumentByIDOptions contains options for retrieving a document by ID
type GetDocumentByIDOptions struct {
	ID    pulid.ID
	OrgID pulid.ID
	BuID  pulid.ID
	DocumentRequest
}

// ListDocumentsRequest contains options for listing documents
type ListDocumentsRequest struct {
	Filter              *ports.LimitOffsetQueryOptions
	ResourceType        permission.Resource       `query:"resourceType"`
	ResourceID          *pulid.ID                 `query:"resourceID"`
	DocumentType        document.DocumentType     `query:"documentType"`
	Statuses            []document.DocumentStatus `query:"statuses"`
	Tags                []string                  `query:"tags"`
	SortBy              string                    `query:"sortBy"`
	SortDir             string                    `query:"sortDir"`
	ExpirationDateStart *int64                    `query:"expirationDateStart"`
	ExpirationDateEnd   *int64                    `query:"expirationDateEnd"`
	CreatedAtStart      *int64                    `query:"createdAtStart"`
	CreatedAtEnd        *int64                    `query:"createdAtEnd"`
	DocumentRequest
}

// FindDocumentsByResourceRequest contains options for finding documents by entity
type FindDocumentsByResourceRequest struct {
	ResourceID   pulid.ID
	ResourceType permission.Resource
	OrgID        pulid.ID
	BuID         pulid.ID
	DocumentType document.DocumentType
	Statuses     []document.DocumentStatus
	DocumentRequest
}

// FindDocumentsByTagsRequest contains options for finding documents by tags
type FindDocumentsByTagsRequest struct {
	Tags         []string
	OrgID        pulid.ID
	BuID         pulid.ID
	DocumentType document.DocumentType
	Statuses     []document.DocumentStatus
	DocumentRequest
}

// FindDocumentsByTypeRequest contains options for finding documents by type
type FindDocumentsByTypeRequest struct {
	DocumentType document.DocumentType
	OrgID        pulid.ID
	BuID         pulid.ID
	ResourceType permission.Resource
	Statuses     []document.DocumentStatus
	DocumentRequest
}

// DeleteDocumentRequest contains options for deleting a document
type DeleteDocumentRequest struct {
	ID    pulid.ID
	OrgID pulid.ID
	BuID  pulid.ID
}

// UpdateDocumentStatusRequest contains options for updating a document's status
type UpdateDocumentStatusRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	Status document.DocumentStatus
}

// BulkUpdateDocumentStatusRequest contains options for bulk updating document statuses
type BulkUpdateDocumentStatusRequest struct {
	IDs    []pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	Status document.DocumentStatus
}

// FindExpiringDocumentsRequest contains options for finding documents nearing expiration
type FindExpiringDocumentsRequest struct {
	ExpirationThreshold int64
	OrgID               pulid.ID
	BuID                pulid.ID
}

// CountDocumentsRequest contains options for counting documents by type
type CountDocumentsRequest struct {
	OrgID        pulid.ID
	BuID         pulid.ID
	ResourceType permission.Resource
	ResourceID   pulid.ID
	Statuses     []document.DocumentStatus
}

// DocumentRepository defines the interface for document data access
type DocumentRepository interface {
	// CRUD operations
	Create(ctx context.Context, doc *document.Document) (*document.Document, error)
	GetByID(ctx context.Context, req GetDocumentByIDOptions) (*document.Document, error)
	Update(ctx context.Context, doc *document.Document) (*document.Document, error)
	Delete(ctx context.Context, req DeleteDocumentRequest) error

	// Query operations
	List(ctx context.Context, req *ListDocumentsRequest) (*ports.ListResult[*document.Document], error)
	FindByResourceID(ctx context.Context, req *FindDocumentsByResourceRequest) ([]*document.Document, error)
	FindByTags(ctx context.Context, req *FindDocumentsByTagsRequest) ([]*document.Document, error)
	FindByDocumentType(ctx context.Context, req *FindDocumentsByTypeRequest) ([]*document.Document, error)
	FindExpiringDocuments(ctx context.Context, req *FindExpiringDocumentsRequest) ([]*document.Document, error)

	// Status operations
	UpdateStatus(ctx context.Context, req *UpdateDocumentStatusRequest) (*document.Document, error)
	BulkUpdateStatus(ctx context.Context, req BulkUpdateDocumentStatusRequest) (int, error)

	// Aggregation operations
	CountDocuments(ctx context.Context, req *CountDocumentsRequest) (map[document.DocumentType]int, error)
}
