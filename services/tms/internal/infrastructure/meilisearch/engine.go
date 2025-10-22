package meilisearch

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"go.uber.org/fx"
)

type EngineParams struct {
	fx.In

	Indexer  *Indexer
	Searcher *Searcher
}

type Engine struct {
	indexer  *Indexer
	searcher *Searcher
}

func NewEngine(p EngineParams) ports.SearchEngine {
	return &Engine{
		indexer:  p.Indexer,
		searcher: p.Searcher,
	}
}

func (e *Engine) Index(ctx context.Context, document *meilisearchtype.SearchDocument) error {
	return e.indexer.Index(ctx, document)
}

func (e *Engine) Delete(
	ctx context.Context,
	req meilisearchtype.DeleteOperationRequest,
) error {
	return e.indexer.Delete(ctx, req)
}

func (e *Engine) BatchIndex(
	ctx context.Context,
	documents []*meilisearchtype.SearchDocument,
) error {
	return e.indexer.BatchIndex(ctx, documents)
}

func (e *Engine) BatchDelete(
	ctx context.Context,
	operations []meilisearchtype.DeleteOperationRequest,
) error {
	return e.indexer.BatchDelete(ctx, operations)
}

func (e *Engine) Search(
	request *meilisearchtype.SearchRequest,
) (*meilisearchtype.SearchResponse, error) {
	return e.searcher.Search(request)
}

func (e *Engine) SearchByEntityType(
	request *meilisearchtype.SearchRequest,
	entityType meilisearchtype.EntityType,
) (*meilisearchtype.SearchResponse, error) {
	return e.searcher.SearchByEntityType(request, entityType)
}
