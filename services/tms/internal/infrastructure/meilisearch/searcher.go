package meilisearch

import (
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/meilisearch/meilisearch-go"
	"github.com/sourcegraph/conc"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearcherParams struct {
	fx.In

	Connection *Connection
	Client     *Client
	Indexer    *Indexer
}

type Searcher struct {
	conn    *Connection
	client  *Client
	indexer *Indexer
}

func NewSearcher(p SearcherParams) *Searcher {
	return &Searcher{
		conn:    p.Connection,
		client:  p.Client,
		indexer: p.Indexer,
	}
}

func (s *Searcher) Search(
	request *meilisearchtype.SearchRequest,
) (*meilisearchtype.SearchResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	entityTypes := request.EntityTypes
	if len(entityTypes) == 0 {
		entityTypes = []meilisearchtype.EntityType{
			meilisearchtype.EntityTypeShipment,
			meilisearchtype.EntityTypeCustomer,
		}
	}

	limit := request.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := request.Offset
	if offset < 0 {
		offset = 0
	}

	results := s.searchAcrossTypes(request, entityTypes, limit, offset)
	response := s.mergeSearchResults(results, request.Query, limit, offset)

	return response, nil
}

func (s *Searcher) searchAcrossTypes(
	request *meilisearchtype.SearchRequest,
	entityTypes []meilisearchtype.EntityType,
	limit, offset int,
) []*meilisearch.SearchResponse {
	var wg conc.WaitGroup
	var mu sync.Mutex
	results := make([]*meilisearch.SearchResponse, 0, len(entityTypes))

	for _, entityType := range entityTypes {
		wg.Go(func() {
			result, err := s.searchSingleType(request, entityType, limit, offset)
			if err != nil {
				s.conn.logger.Warn("Failed to search entity type",
					zap.String("entityType", entityType.String()),
					zap.Error(err),
				)
				return
			}

			if result != nil && len(result.Hits) > 0 {
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	return results
}

func (s *Searcher) searchSingleType(
	request *meilisearchtype.SearchRequest,
	entityType meilisearchtype.EntityType,
	limit, offset int,
) (*meilisearch.SearchResponse, error) {
	indexName := GetIndexName(
		s.conn.indexPrefix,
		request.OrganizationID.String(),
		request.BusinessUnitID.String(),
		entityType,
	)

	index := s.conn.Manager().Index(indexName)
	indexResult, err := index.FetchInfo()
	if err != nil {
		return nil, err
	}

	searchReq := &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
		Filter: s.buildFilter(request),
	}

	result, err := s.client.Search(indexResult, request.Query, searchReq)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Searcher) buildFilter(request *meilisearchtype.SearchRequest) string {
	filters := []string{
		fmt.Sprintf("organizationId = %s", request.OrganizationID.String()),
		fmt.Sprintf("businessUnitId = %s", request.BusinessUnitID.String()),
	}

	for key, value := range request.Filters {
		switch v := value.(type) {
		case string:
			filters = append(filters, fmt.Sprintf("%s = %s", key, v))
		case int, int64:
			filters = append(filters, fmt.Sprintf("%s = %v", key, v))
		case bool:
			filters = append(filters, fmt.Sprintf("%s = %v", key, v))
		}
	}

	if len(filters) == 0 {
		return ""
	}

	filterStr := filters[0]
	for i := 1; i < len(filters); i++ {
		filterStr += " AND " + filters[i]
	}

	return filterStr
}

func (s *Searcher) mergeSearchResults(
	results []*meilisearch.SearchResponse,
	query string,
	limit, offset int,
) *meilisearchtype.SearchResponse {
	response := &meilisearchtype.SearchResponse{
		Hits:   make([]meilisearchtype.SearchHit, 0),
		Total:  0,
		Offset: offset,
		Limit:  limit,
		Query:  query,
	}

	allHits := make([]meilisearchtype.SearchHit, 0)
	totalProcessingTime := int64(0)

	for _, result := range results {
		response.Total += result.EstimatedTotalHits
		totalProcessingTime += result.ProcessingTimeMs

		for _, hit := range result.Hits {
			hitBytes, err := sonic.Marshal(hit)
			if err != nil {
				s.conn.logger.Warn("Failed to marshal hit", zap.Error(err))
				continue
			}

			var hitMap map[string]any
			if err = sonic.Unmarshal(hitBytes, &hitMap); err != nil {
				s.conn.logger.Warn("Failed to unmarshal hit", zap.Error(err))
				continue
			}

			searchHit := meilisearchtype.SearchHit{
				ID:         getString(hitMap, "id"),
				EntityType: meilisearchtype.EntityType(getString(hitMap, "entityType")),
				Title:      getString(hitMap, "title"),
				Subtitle:   getString(hitMap, "subtitle"),
				Metadata:   getMap(hitMap, "metadata"),
			}

			if formatted, ok := hitMap["_formatted"].(map[string]any); ok {
				searchHit.HighlightedContent = make(map[string]string)
				if title, titleIsString := formatted["title"].(string); titleIsString {
					searchHit.HighlightedContent["title"] = title
				}
				if content, contentIsString := formatted["content"].(string); contentIsString {
					searchHit.HighlightedContent["content"] = content
				}
			}

			allHits = append(allHits, searchHit)
		}
	}

	// Sort by relevance (Meilisearch already sorts within each index)
	// For cross-index sorting, we'd need to implement custom scoring
	// For now, just combine the results as-is

	start := offset
	if start > len(allHits) {
		start = len(allHits)
	}

	end := start + limit
	if end > len(allHits) {
		end = len(allHits)
	}

	response.Hits = allHits[start:end]
	response.ProcessingTimeMs = totalProcessingTime

	return response
}

func (s *Searcher) SearchByEntityType(
	request *meilisearchtype.SearchRequest,
	entityType meilisearchtype.EntityType,
) (*meilisearchtype.SearchResponse, error) {
	if !entityType.IsValid() {
		return nil, ErrInvalidEntityType
	}

	if err := request.Validate(); err != nil {
		return nil, err
	}

	limit := request.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := request.Offset
	if offset < 0 {
		offset = 0
	}

	result, err := s.searchSingleType(request, entityType, limit, offset)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return &meilisearchtype.SearchResponse{
			Hits:             []meilisearchtype.SearchHit{},
			Total:            0,
			Offset:           offset,
			Limit:            limit,
			ProcessingTimeMs: 0,
			Query:            request.Query,
		}, nil
	}

	response := s.mergeSearchResults(
		[]*meilisearch.SearchResponse{result},
		request.Query,
		limit,
		offset,
	)

	return response, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if str, isString := v.(string); isString {
			return str
		}
	}
	return ""
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mapVal, isMap := v.(map[string]any); isMap {
			return mapVal
		}
	}
	return nil
}
