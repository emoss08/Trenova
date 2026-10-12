package documentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StoreDocumentRequest struct {
	TenantInfo     pagination.TenantInfo
	Actor          permissionports.RequestActor
	Content        []byte
	FileName       string
	ContentType    string
	ResourceType   string
	ResourceID     string
	DocumentTypeID pulid.ID
	Description    string
	Tags           []string
}

type DocumentStore interface {
	Store(ctx context.Context, req *StoreDocumentRequest) (*document.Document, error)
}
