package ports

import (
	"context"

	"github.com/emoss08/trenova/pkg/meilisearchtype"
)

type SearchEngine interface {
	Index(ctx context.Context, document *meilisearchtype.SearchDocument) error
	Delete(
		ctx context.Context,
		req meilisearchtype.DeleteOperationRequest,
	) error
	BatchIndex(ctx context.Context, documents []*meilisearchtype.SearchDocument) error
	BatchDelete(ctx context.Context, operations []meilisearchtype.DeleteOperationRequest) error
	Search(request *meilisearchtype.SearchRequest) (*meilisearchtype.SearchResponse, error)
	SearchByEntityType(
		request *meilisearchtype.SearchRequest,
		entityType meilisearchtype.EntityType,
	) (*meilisearchtype.SearchResponse, error)
}
