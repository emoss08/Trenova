package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// UploadDocumentRequest contains the details needed to upload a document
type UploadDocumentRequest struct {
	// Required fields
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UploadedByID   pulid.ID
	ResourceID     pulid.ID
	ResourceType   permission.Resource
	DocumentType   document.DocumentType
	File           []byte
	FileName       string
	OriginalName   string

	// Optional fields
	Description     string
	Tags            []string
	ExpirationDate  *int64
	IsPublic        bool
	Status          document.DocumentStatus
	RequireApproval bool
}

// UploadDocumentResponse contains the result of a document upload
type UploadDocumentResponse struct {
	Document  *document.Document
	Location  string
	Checksum  string
	Size      int64
	VersionID string
}

// BulkUploadDocumentRequest contains multiple document upload requests
type BulkUploadDocumentRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UploadedByID   pulid.ID
	ResourceID     pulid.ID
	ResourceType   permission.Resource
	Documents      []BulkDocumentInfo
}

// BulkDocumentInfo contains information for a single document in a bulk upload
type BulkDocumentInfo struct {
	DocumentType   document.DocumentType
	File           []byte
	FileName       string
	OriginalName   string
	Description    string
	Tags           []string
	ExpirationDate *int64
	IsPublic       bool
}

// BulkUploadResponse contains the results of a bulk document upload
type BulkUploadDocumentResponse struct {
	Successful []UploadDocumentResponse
	Failed     []FailedUpload
}

// FailedUpload contains information about a failed document upload
type FailedUpload struct {
	FileName string
	Error    error
}

// DocumentService defines the interface for document management operations
type DocumentService interface {
	// Upload operations
	UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*UploadDocumentResponse, error)
	BulkUploadDocuments(ctx context.Context, req *BulkUploadDocumentRequest) (*BulkUploadDocumentResponse, error)

	// Retrieval operations
	List(ctx context.Context, req *repositories.ListDocumentsRequest) (*ports.ListResult[*document.Document], error)
	GetDocumentByID(ctx context.Context, orgID, buID, docID pulid.ID) (*document.Document, error)
	GetDocumentContent(ctx context.Context, doc *document.Document) ([]byte, error)
	GetDocumentDownloadURL(ctx context.Context, doc *document.Document, expiryDuration time.Duration) (string, error)
	ListEntityDocuments(ctx context.Context, req *repositories.ListDocumentsRequest) (*ports.ListResult[*document.Document], error)

	// Document workflow operations
	ApproveDocument(ctx context.Context, orgID, buID, docID, approverID pulid.ID) (*document.Document, error)
	RejectDocument(ctx context.Context, orgID, buID, docID, rejectorID pulid.ID, reason string) (*document.Document, error)
	ArchiveDocument(ctx context.Context, orgID, buID, docID pulid.ID) (*document.Document, error)
	DeleteDocument(ctx context.Context, orgID, buID, docID pulid.ID) error

	// Versioning operations
	GetDocumentVersions(ctx context.Context, doc *document.Document) ([]VersionInfo, error)
	RestoreDocumentVersion(ctx context.Context, doc *document.Document, versionID string) (*document.Document, error)

	// Compliance operation
	CheckExpiringDocuments(ctx context.Context, daysToExpiration int) ([]*document.Document, error)
}
