package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDocumentAIExtractionRequest struct {
	DocumentID  pulid.ID
	ExtractedAt int64
	TenantInfo  pagination.TenantInfo
}

type ListPollableDocumentAIExtractionRequest struct {
	TenantInfo pagination.TenantInfo
	OlderThan  int64
	Limit      int
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
	ListPollable(
		ctx context.Context,
		req *ListPollableDocumentAIExtractionRequest,
	) ([]*documentaiextraction.Extraction, error)
	ListPollableTenants(
		ctx context.Context,
		req *ListPollableDocumentAIExtractionRequest,
	) ([]pagination.TenantInfo, error)
}
