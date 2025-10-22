package providers

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearchHelper struct {
	engine       ports.SearchEngine
	logger       *zap.Logger
	baseProvider *BaseProvider
}

type SearchHelperParams struct {
	fx.In
	Engine ports.SearchEngine
	Logger *zap.Logger
}

func NewSearchHelper(p SearchHelperParams) *SearchHelper {
	return &SearchHelper{
		engine:       p.Engine,
		logger:       p.Logger,
		baseProvider: &BaseProvider{},
	}
}

func (h *SearchHelper) Delete(
	ctx context.Context,
	entityType meilisearchtype.EntityType,
	orgID, buID, documentID string,
) error {
	req := meilisearchtype.DeleteOperationRequest{
		EntityType: entityType,
		OrgID:      orgID,
		BuID:       buID,
		DocumentID: documentID,
	}

	if err := h.engine.Delete(ctx, req); err != nil {
		return err
	}

	return nil
}

func (h *SearchHelper) Search(
	request *meilisearchtype.SearchRequest,
) (*meilisearchtype.SearchResponse, error) {
	return h.engine.Search(request)
}

func (h *SearchHelper) SearchByEntityType(
	request *meilisearchtype.SearchRequest,
	entityType meilisearchtype.EntityType,
) (*meilisearchtype.SearchResponse, error) {
	return h.engine.SearchByEntityType(request, entityType)
}

func (h *SearchHelper) Index(ctx context.Context, entity meilisearchtype.Searchable) error {
	doc, err := h.baseProvider.ToSearchDocument(entity)
	if err != nil {
		return err
	}

	if indexErr := h.engine.Index(ctx, doc); indexErr != nil {
		return indexErr
	}

	return nil
}

func (h *SearchHelper) BatchIndex(
	ctx context.Context,
	entities []meilisearchtype.Searchable,
) error {
	if len(entities) == 0 {
		return nil
	}

	docs, err := h.baseProvider.ToSearchDocuments(entities)
	if err != nil {
		return err
	}

	if indexErr := h.engine.BatchIndex(ctx, docs); indexErr != nil {
		return indexErr
	}

	h.logger.Info("Indexed entities", zap.Int("count", len(docs)))
	return nil
}
