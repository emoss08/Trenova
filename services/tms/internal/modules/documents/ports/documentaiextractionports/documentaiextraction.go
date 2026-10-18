package documentaiextractionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDocumentAIExtractionRequest struct {
	DocumentID  pulid.ID
	ExtractedAt int64
	TenantInfo  pagination.TenantInfo
}

type DocumentAIExtractionRepository interface {
	GetByDocumentExtractedAt(
		ctx context.Context,
		req GetDocumentAIExtractionRequest,
	) (*documentaiextraction.Extraction, error)
	SavePending(
		ctx context.Context,
		entity *documentaiextraction.Extraction,
	) (*documentaiextraction.Extraction, error)
	Update(
		ctx context.Context,
		entity *documentaiextraction.Extraction,
	) (*documentaiextraction.Extraction, error)
}
