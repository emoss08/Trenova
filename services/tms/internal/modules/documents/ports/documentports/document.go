package documentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentsRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	ResourceID   string                   `json:"resourceId"`
	ResourceType string                   `json:"resourceType"`
	Status       string                   `json:"status"`
	Viewer       pulid.ID                 `json:"-"`
}

type DocumentSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	ResourceID         string                         `json:"resourceId"`
	ResourceType       string                         `json:"resourceType"`
	IDs                []pulid.ID                     `json:"ids"`
	Viewer             pulid.ID                       `json:"-"`
}

type GetDocumentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetDocumentPreviewRequest struct {
	GetDocumentByIDRequest

	IfNoneMatch string `json:"ifNoneMatch"`
}

type GetDocumentPageImageRequest struct {
	GetDocumentByIDRequest

	PageNumber  int    `json:"pageNumber"`
	IfNoneMatch string `json:"ifNoneMatch"`
}

type GetDocumentByStoragePathRequest struct {
	StoragePath string                `json:"storagePath"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
}

type GetDocumentsByResourceRequest struct {
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	ResourceID          string                `json:"resourceId"`
	ResourceType        string                `json:"resourceType"`
	IncludeDocumentType bool                  `json:"includeDocumentType"`
	Viewer              pulid.ID              `json:"-"`
}

type GetDocumentsByIDsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IDs                 []pulid.ID            `json:"ids"`
	IncludeDocumentType bool                  `json:"includeDocumentType"`
}

type GetDocumentsByResourceIDsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	ResourceType        string                `json:"resourceType"`
	ResourceIDs         []string              `json:"resourceIds"`
	IncludeDocumentType bool                  `json:"includeDocumentType"`
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
	// ProcessingProfile moves the document to another profile when set.
	ProcessingProfile *document.ProcessingProfile `json:"processingProfile,omitempty"`
}

type RecordDetectedSignatureRequest struct {
	ID         pulid.ID                 `json:"id"`
	TenantInfo pagination.TenantInfo    `json:"tenantInfo"`
	Status     document.SignatureStatus `json:"status"`
	SignedAt   *int64                   `json:"signedAt"`
	DetectedAt int64                    `json:"detectedAt"`
}

// RecordDeliveryFactsRequest writes a document's delivery facts and nothing
// else, so it never carries a stale copy of the rest of the row back over a
// change the conversion or a person made meanwhile.
type RecordDeliveryFactsRequest struct {
	ID         pulid.ID                `json:"id"`
	TenantInfo pagination.TenantInfo   `json:"tenantInfo"`
	Facts      *document.DeliveryFacts `json:"facts"`
	// Version, when set, is the version the writer read; a document that
	// changed since is left as it is.
	Version *int64 `json:"version,omitempty"`
	// KeepPerson leaves facts a person recorded as they are.
	KeepPerson bool `json:"keepPerson,omitempty"`
}

type ListPendingPreviewReconciliationRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	OlderThan  int64                 `json:"olderThan"`
	Limit      int                   `json:"limit"`
}

type ListDocumentVersionsRequest struct {
	LineageID  pulid.ID              `json:"lineageId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type PromoteDocumentVersionRequest struct {
	LineageID         pulid.ID              `json:"lineageId"`
	CurrentDocumentID pulid.ID              `json:"currentDocumentId"`
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteDocumentLineageRequest struct {
	LineageIDs []pulid.ID            `json:"lineageIds"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type MoveDocumentLineageRequest struct {
	DocumentID   pulid.ID              `json:"documentId"`
	ResourceID   string                `json:"resourceId"`
	ResourceType string                `json:"resourceType"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
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
	SelectOptions(
		ctx context.Context,
		req *DocumentSelectOptionsRequest,
	) (*pagination.ListResult[*document.Document], error)
	GetByID(ctx context.Context, req GetDocumentByIDRequest) (*document.Document, error)
	GetByStoragePath(
		ctx context.Context,
		req GetDocumentByStoragePathRequest,
	) (*document.Document, error)
	GetByIDs(ctx context.Context, req GetDocumentsByIDsRequest) ([]*document.Document, error)
	ListVersions(ctx context.Context, req ListDocumentVersionsRequest) ([]*document.Document, error)
	GetByResourceID(
		ctx context.Context,
		req *GetDocumentsByResourceRequest,
	) ([]*document.Document, error)
	GetByResourceIDs(
		ctx context.Context,
		req *GetDocumentsByResourceIDsRequest,
	) ([]*document.Document, error)
	ListPendingPreviewReconciliation(
		ctx context.Context,
		req *ListPendingPreviewReconciliationRequest,
	) ([]*document.Document, error)
	ListPendingPreviewReconciliationTenants(
		ctx context.Context,
		req *ListPendingPreviewReconciliationRequest,
	) ([]pagination.TenantInfo, error)
	Create(ctx context.Context, entity *document.Document) (*document.Document, error)
	Update(ctx context.Context, entity *document.Document) (*document.Document, error)
	UpdatePreview(ctx context.Context, req *UpdateDocumentPreviewRequest) error
	UpdateIntelligence(ctx context.Context, req *UpdateDocumentIntelligenceRequest) error
	RecordDetectedSignature(ctx context.Context, req *RecordDetectedSignatureRequest) (bool, error)
	// RecordDeliveryFacts returns the document as written, or nil when its
	// guards left it as it was.
	RecordDeliveryFacts(
		ctx context.Context,
		req *RecordDeliveryFactsRequest,
	) (*document.Document, error)
	PromoteVersion(ctx context.Context, req *PromoteDocumentVersionRequest) error
	MoveLineageToResource(ctx context.Context, req *MoveDocumentLineageRequest) error
	Delete(ctx context.Context, req DeleteDocumentRequest) error
	BulkDelete(ctx context.Context, req BulkDeleteDocumentRequest) error
	DeleteByLineageIDs(ctx context.Context, req DeleteDocumentLineageRequest) error
}

const DocumentChangeObserverGroup = `group:"document_change_observers"`

type DocumentChange struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceIDs  []string
	Retired      []RetiredDocument
}

// RetiredDocument is a document that left its resource: deleted, rejected,
// archived, moved to another resource, or superseded by ReplacedBy, a newer
// version of it.
type RetiredDocument struct {
	ID         pulid.ID
	ResourceID string
	ReplacedBy pulid.ID
}

type DocumentChangeObserver interface {
	DocumentsChanged(ctx context.Context, change *DocumentChange)
}
