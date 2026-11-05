package documenthandler_test

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type noopDocumentContent struct{}

func (noopDocumentContent) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
}

func (noopDocumentContent) GetShipmentDraft(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
}

func (noopDocumentContent) SearchDocuments(
	context.Context,
	pagination.TenantInfo,
	string,
	string,
	string,
) ([]*document.Document, error) {
	return nil, nil
}

func (noopDocumentContent) Reextract(context.Context, pulid.ID, pagination.TenantInfo) error {
	return nil
}

func (noopDocumentContent) EnqueueExtraction(
	context.Context,
	*document.Document,
	pulid.ID,
) error {
	return nil
}

type noopSearchProjection struct{}

func (noopSearchProjection) Upsert(context.Context, *document.Document, string) error {
	return nil
}

func (noopSearchProjection) Delete(context.Context, pulid.ID, pagination.TenantInfo) error {
	return nil
}

type noopPaperwork struct{}

func (noopPaperwork) RequestRead(context.Context, *document.Document, pulid.ID) {}

var (
	_ documentports.DocumentContentService                          = noopDocumentContent{}
	_ documentports.PaperworkReadRequester                          = noopPaperwork{}
	_ documentsearchprojectionports.DocumentSearchProjectionService = noopSearchProjection{}
)
