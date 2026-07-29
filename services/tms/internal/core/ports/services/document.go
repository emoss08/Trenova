package services

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type DocumentUploadResult struct {
	Document *document.Document `json:"document"`
}

type DocumentContent struct {
	Document           *document.Document `json:"document"`
	Body               io.ReadCloser      `json:"body"`
	ContentType        string             `json:"contentType"`
	ContentLength      int64              `json:"contentLength"`
	ContentDisposition string             `json:"contentDisposition"`
}

type InvoiceDocumentService interface {
	Get(
		ctx context.Context,
		req repositories.GetDocumentByIDRequest,
	) (*document.Document, error)
	GetDownloadContent(
		ctx context.Context,
		req repositories.GetDocumentByIDRequest,
	) (*DocumentContent, error)
}

type UploadRequest struct {
	TenantInfo        pagination.TenantInfo
	Actor             RequestActor
	File              *multipart.FileHeader
	ResourceID        string
	ResourceType      string
	ProcessingProfile string
	Description       string
	Tags              []string
	DocumentTypeID    string
	LineageID         string
}

type BulkUploadRequest struct {
	TenantInfo   pagination.TenantInfo
	Actor        RequestActor
	Files        []*multipart.FileHeader
	ResourceID   string
	ResourceType string
	LineageID    string
}

type BulkUploadResult struct {
	Documents []*document.Document
	Errors    []error
}
