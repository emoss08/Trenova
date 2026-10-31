package documentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentContentService interface {
	GetContent(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
	GetShipmentDraft(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentshipmentdraft.DocumentShipmentDraft, error)
	SearchDocuments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		resourceType, resourceID, query string,
	) ([]*document.Document, error)
	Reextract(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) error
	EnqueueExtraction(ctx context.Context, doc *document.Document, userID pulid.ID) error
}

type ExtractionStatus struct {
	ModelProviderAvailable bool   `json:"modelProviderAvailable"`
	ModelProviderReason    string `json:"modelProviderReason"`
	ConverterAvailable     bool   `json:"converterAvailable"`
	ExtractionAvailable    bool   `json:"extractionAvailable"`
}

type ExtractionStatusReader interface {
	ExtractionStatus(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ExtractionStatus, error)
}

// DocumentReader is a document as it was read: each value with the page and box
// it was found in, in the extraction card's payload, which the Desk and the
// billing queue's document review both draw. The agent module implements it.
type DocumentReader interface {
	ReadDocument(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (map[string]any, error)
}

// PaperworkReadRequester asks Nova to read a shipment document the moment it
// arrives, alongside its conversion. The agent module implements it.
type PaperworkReadRequester interface {
	RequestRead(ctx context.Context, doc *document.Document, userID pulid.ID)
}
