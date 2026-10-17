package documentports

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

type DocumentPageImage struct {
	DocumentContent

	ETag        string `json:"etag"`
	NotModified bool   `json:"notModified"`
}

type InvoiceDocumentService interface {
	Get(
		ctx context.Context,
		req GetDocumentByIDRequest,
	) (*document.Document, error)
	GetDownloadContent(
		ctx context.Context,
		req GetDocumentByIDRequest,
	) (*DocumentContent, error)
}

// ConversationFileReader reads a file a person attached to a conversation so
// the system can process it, as document intelligence reads an upload: it is
// not a person viewing the file, so no view is recorded. Any other document is
// not found.
type ConversationFileReader interface {
	ReadConversationFile(
		ctx context.Context,
		req GetDocumentByIDRequest,
	) (*ConversationFile, error)
}

// ConversationFile is an attached file's document and its bytes, decrypted.
type ConversationFile struct {
	Document *document.Document
	Data     []byte
}

type UploadRequest struct {
	TenantInfo        pagination.TenantInfo
	Actor             permissionports.RequestActor
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
	Actor        permissionports.RequestActor
	Files        []*multipart.FileHeader
	ResourceID   string
	ResourceType string
	LineageID    string
}

type BulkUploadResult struct {
	Documents []*document.Document
	Errors    []error
}
