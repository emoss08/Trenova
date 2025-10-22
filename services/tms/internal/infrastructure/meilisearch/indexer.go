package meilisearch

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
)

type Indexer struct {
	client     *Client
	conn       *Connection
	indexCache map[string]*meilisearch.IndexResult
	indexMu    sync.RWMutex
}

func NewIndexer(conn *Connection) *Indexer {
	return &Indexer{
		client:     NewClient(conn),
		conn:       conn,
		indexCache: make(map[string]*meilisearch.IndexResult),
	}
}

func (i *Indexer) Index(ctx context.Context, document *meilisearchtype.SearchDocument) error {
	if err := document.Validate(); err != nil {
		return err
	}

	index, err := i.getIndex(
		ctx,
		document.EntityType,
		document.OrganizationID,
		document.BusinessUnitID,
	)
	if err != nil {
		return err
	}

	if _, err = i.client.AddDocuments(index, []*meilisearchtype.SearchDocument{document}); err != nil {
		return err
	}

	return nil
}

func (i *Indexer) Delete(
	ctx context.Context,
	req meilisearchtype.DeleteOperationRequest,
) error {
	if !req.EntityType.IsValid() {
		return ErrInvalidEntityType
	}

	index, err := i.getIndex(ctx, req.EntityType, req.OrgID, req.BuID)
	if err != nil {
		return err
	}

	if _, err = i.client.DeleteDocument(index, req.DocumentID); err != nil {
		return err
	}

	return nil
}

func (i *Indexer) BatchIndex(
	ctx context.Context,
	documents []*meilisearchtype.SearchDocument,
) error {
	if len(documents) == 0 {
		return nil
	}

	grouped := i.groupDocumentsByIndex(documents)

	for key, docs := range grouped {
		if len(docs) == 0 {
			continue
		}

		for _, doc := range docs {
			if err := doc.Validate(); err != nil {
				i.conn.logger.Warn("Skipping invalid document in batch",
					zap.String("documentID", doc.ID),
					zap.Error(err),
				)
				continue
			}
		}

		firstDoc := docs[0]
		index, err := i.getIndex(
			ctx,
			firstDoc.EntityType,
			firstDoc.OrganizationID,
			firstDoc.BusinessUnitID,
		)
		if err != nil {
			i.conn.logger.Error("Failed to get index for batch",
				zap.String("key", key),
				zap.Error(err),
			)
			continue
		}

		if _, err = i.client.AddDocuments(index, docs); err != nil {
			i.conn.logger.Error("Failed to index batch",
				zap.String("index", index.UID),
				zap.Int("count", len(docs)),
				zap.Error(err),
			)
			continue
		}

		i.conn.logger.Info("Batch indexed documents",
			zap.String("index", index.UID),
			zap.Int("count", len(docs)),
		)
	}

	return nil
}

func (i *Indexer) BatchDelete(
	ctx context.Context,
	operations []meilisearchtype.DeleteOperationRequest,
) error {
	if len(operations) == 0 {
		return nil
	}

	type indexKey struct {
		EntityType meilisearchtype.EntityType
		OrgID      string
		BuID       string
	}
	grouped := make(map[indexKey][]string)

	for _, op := range operations {
		key := indexKey{
			EntityType: op.EntityType,
			OrgID:      op.OrgID,
			BuID:       op.BuID,
		}
		grouped[key] = append(grouped[key], op.DocumentID)
	}

	for key, docIDs := range grouped {
		if len(docIDs) == 0 {
			continue
		}

		index, err := i.getIndex(ctx, key.EntityType, key.OrgID, key.BuID)
		if err != nil {
			i.conn.logger.Error("Failed to get index for batch delete",
				zap.String("entityType", key.EntityType.String()),
				zap.Error(err),
			)
			continue
		}

		_, err = i.client.DeleteDocuments(index, docIDs)
		if err != nil {
			i.conn.logger.Error("Failed to delete batch",
				zap.String("index", index.UID),
				zap.Int("count", len(docIDs)),
				zap.Error(err),
			)
			continue
		}

		i.conn.logger.Info("Batch deleted documents",
			zap.String("index", index.UID),
			zap.Int("count", len(docIDs)),
		)
	}

	return nil
}

func (i *Indexer) getIndex(
	ctx context.Context,
	entityType meilisearchtype.EntityType,
	orgID, buID string,
) (*meilisearch.IndexResult, error) {
	if !entityType.IsValid() {
		return nil, ErrInvalidEntityType
	}

	indexName := GetIndexName(i.conn.indexPrefix, orgID, buID, entityType)

	i.indexMu.RLock()
	if index, exists := i.indexCache[indexName]; exists {
		i.indexMu.RUnlock()
		return index, nil
	}
	i.indexMu.RUnlock()

	index, err := i.client.GetOrCreateIndex(ctx, indexName, "id")
	if err != nil {
		return nil, err
	}

	config := GetIndexConfig(entityType)
	if err = i.client.UpdateIndexSettings(ctx, index, &config); err != nil {
		i.conn.logger.Warn("Failed to update index settings",
			zap.String("index", indexName),
			zap.Error(err),
		)
	}

	i.indexMu.Lock()
	i.indexCache[indexName] = index
	i.indexMu.Unlock()

	return index, nil
}

func (i *Indexer) groupDocumentsByIndex(
	documents []*meilisearchtype.SearchDocument,
) map[string][]*meilisearchtype.SearchDocument {
	grouped := make(map[string][]*meilisearchtype.SearchDocument)

	for _, doc := range documents {
		key := fmt.Sprintf("%s_%s_%s", doc.OrganizationID, doc.BusinessUnitID, doc.EntityType)
		grouped[key] = append(grouped[key], doc)
	}

	return grouped
}

func (i *Indexer) ClearCache() {
	i.indexMu.Lock()
	defer i.indexMu.Unlock()
	i.indexCache = make(map[string]*meilisearch.IndexResult)
}
