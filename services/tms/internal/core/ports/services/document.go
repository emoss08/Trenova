package services

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

type DocumentUploadResult struct {
	Document *document.Document
}

type DocumentContent struct {
	Document           *document.Document
	Body               io.ReadCloser
	ContentType        string
	ContentLength      int64
	ContentDisposition string
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
