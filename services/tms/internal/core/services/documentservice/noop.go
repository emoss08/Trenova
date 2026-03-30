package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type noopDocumentContentService struct{}

func (noopDocumentContentService) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return nil, nil
}

func (noopDocumentContentService) GetShipmentDraft(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentshipmentdraft.Draft, error) {
	return nil, nil
}

func (noopDocumentContentService) SearchDocuments(
	context.Context,
	pagination.TenantInfo,
	string,
	string,
	string,
) ([]*document.Document, error) {
	return nil, nil
}

func (noopDocumentContentService) Reextract(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) error {
	return nil
}

func (noopDocumentContentService) EnqueueExtraction(
	context.Context,
	*document.Document,
	pulid.ID,
) error {
	return nil
}

type noopDocumentSearchProjectionService struct{}

func (noopDocumentSearchProjectionService) Upsert(
	context.Context,
	*document.Document,
	string,
) error {
	return nil
}

func (noopDocumentSearchProjectionService) Delete(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) error {
	return nil
}

var (
	_ serviceports.DocumentContentService          = noopDocumentContentService{}
	_ serviceports.DocumentSearchProjectionService = noopDocumentSearchProjectionService{}
)
