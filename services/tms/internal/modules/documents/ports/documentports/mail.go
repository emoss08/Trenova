package documentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxMailAttachments = 20

type ResolveMailAttachmentsRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentIDs []pulid.ID
	Field       string
	System      bool
	Recipients  []string
}

type MailAttachment struct {
	Document   *document.Document
	Attachment emailports.EmailAttachment
}

type MailDocumentSource struct {
	ResourceType string
	ResourceID   string
}

type MailableDocumentsRequest struct {
	TenantInfo pagination.TenantInfo
	Sources    []MailDocumentSource
}

type DocumentMailAttachments interface {
	Resolve(ctx context.Context, req *ResolveMailAttachmentsRequest) ([]*MailAttachment, error)
	Mailable(ctx context.Context, req *MailableDocumentsRequest) ([]*document.Document, error)
}
