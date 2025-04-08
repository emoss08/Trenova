package meilisearch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/meilisearch/meilisearch-go"
	"github.com/mitchellh/mapstructure"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
}

type client struct {
	meili         meilisearch.ServiceManager
	l             *zerolog.Logger
	cfg           *config.SearchConfig
	indexMutex    sync.Mutex // Protects index operations
	indexInitOnce sync.Once  // Ensures index initialization happens only once
	indexInitErr  error      // Stores error from index initialization
}

func NewClient(p ClientParams) infra.SearchClient {
	log := p.Logger.With().Str("client", "meilisearch").Logger()

	cfg := p.Config.Search()
	if cfg == nil {
		log.Error().Msg("search configuration is nil")
		return nil
	}

	// Validate configuration
	if cfg.Host == "" {
		log.Error().Msg("meilisearch host is required")
		return nil
	}

	// Log configuration
	log.Debug().
		Str("host", cfg.Host).
		Int("maxBatchSize", cfg.MaxBatchSize).
		Int("batchInterval", cfg.BatchInterval).
		Str("indexPrefix", cfg.IndexPrefix).
		Msg("initializing meilisearch client")

	c := &client{
		meili: meilisearch.New(cfg.Host, meilisearch.WithAPIKey(cfg.APIKey)),
		l:     &log,
		cfg:   cfg,
	}

	// Asynchronously initialize indexes to not block startup
	go func() {
		err := c.InitializeIndexes()
		if err != nil {
			log.Error().Err(err).Msg("failed to initialize search indexes")
		}
	}()

	return c
}

// InitializeIndexes sets up the required search indexes with appropriate settings.
func (c *client) InitializeIndexes() error {
	// Use sync.Once to ensure initialization runs only once
	c.indexInitOnce.Do(func() {
		idxName := c.GetIndexName()
		c.l.Info().Str("index", idxName).Msg("initializing search index")

		// Lock to prevent concurrent index operations
		c.indexMutex.Lock()
		defer c.indexMutex.Unlock()

		// Check if index exists
		c.createIndexIfNotExists(idxName)
		if c.indexInitErr != nil {
			return
		}

		// Configure index settings
		c.configureIndexSettings(idxName)
	})

	return c.indexInitErr
}

// createIndexIfNotExists creates the search index if it doesn't already exist
func (c *client) createIndexIfNotExists(idxName string) {
	_, err := c.meili.GetIndex(idxName)
	if err != nil {
		// Create index if it doesn't exist
		c.l.Debug().Str("index", idxName).Msg("index not found, creating...")

		createTask, cErr := c.meili.CreateIndex(&meilisearch.IndexConfig{
			Uid:        idxName,
			PrimaryKey: "id",
		})
		if cErr != nil {
			c.indexInitErr = eris.Wrapf(cErr, "failed to create index %s", idxName)
			return
		}

		// Wait for index creation to complete
		_, cErr = c.meili.WaitForTask(createTask.TaskUID, 30*time.Second)
		if cErr != nil {
			c.indexInitErr = eris.Wrapf(cErr, "failed to wait for index creation task %d", createTask.TaskUID)
			return
		}
	}
}

// configureIndexSettings applies the search configuration settings to the index
func (c *client) configureIndexSettings(idxName string) {
	settings := c.buildIndexSettings()

	// Update index settings
	settingsTask, err := c.meili.Index(idxName).UpdateSettings(settings)
	if err != nil {
		c.indexInitErr = eris.Wrapf(err, "failed to update index settings for %s", idxName)
		return
	}

	// Wait for settings update to complete
	_, err = c.meili.WaitForTask(settingsTask.TaskUID, 30*time.Second)
	if err != nil {
		c.indexInitErr = eris.Wrapf(err, "failed to wait for settings update task %d", settingsTask.TaskUID)
		return
	}

	c.l.Info().Str("index", idxName).Msg("search index initialized successfully")
}

// buildIndexSettings creates the Meilisearch index settings configuration
func (c *client) buildIndexSettings() *meilisearch.Settings {
	return &meilisearch.Settings{
		// Attributes that can be searched
		SearchableAttributes: []string{
			"title",
			"description",
			"searchableText",
			"metadata.routeNumber",
			"metadata.customerName",
			"metadata.shipmentNumbers",
			"metadata.equipmentNumber",
			"metadata.driverName",
			"metadata.tags",
		},

		// Attributes that can be filtered
		FilterableAttributes: []string{
			"type",
			"organizationId",
			"businessUnitId",
			"createdAt",
			"updatedAt",
			"metadata.status",
			"metadata.priority",
			"metadata.assignedTo",
			"metadata.tags",
			"metadata.customerType",
			"metadata.equipmentType",
			"metadata.departmentCode",
		},

		// Attributes that can be sorted
		SortableAttributes: []string{
			"createdAt",
			"updatedAt",
			"metadata.priority",
			"metadata.dueDate",
			"metadata.pickupTime",
			"metadata.deliveryTime",
		},

		// Ranking rules in order of importance
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},

		// Attributes for faceting (filtering and aggregations)
		Faceting: &meilisearch.Faceting{
			MaxValuesPerFacet: 100,
			SortFacetValuesBy: map[string]meilisearch.SortFacetType{
				"count": meilisearch.SortFacetTypeCount,
			},
		},

		// Synonyms to improve search relevance
		Synonyms: map[string][]string{
			"shipment":  {"load", "freight", "cargo"},
			"customer":  {"client", "shipper", "consignee"},
			"driver":    {"operator", "trucker"},
			"equipment": {"truck", "vehicle", "trailer", "asset"},
			"route":     {"trip", "journey", "path"},
		},

		// Typo tolerance settings
		TypoTolerance: &meilisearch.TypoTolerance{
			Enabled: true,
			MinWordSizeForTypos: meilisearch.MinWordSizeForTypos{
				OneTypo:  4,
				TwoTypos: 8,
			},
			DisableOnWords:      []string{},
			DisableOnAttributes: []string{},
		},
	}
}

// IndexDocuments indexes a batch of documents.
func (c *client) IndexDocuments(indexName string, docs []*infra.SearchDocument) (*infra.SearchTaskInfo, error) {
	if len(docs) == 0 {
		return nil, eris.New("no documents to index")
	}

	// Add timestamps if missing
	now := time.Now().Unix()
	for _, doc := range docs {
		if doc.CreatedAt == 0 {
			doc.CreatedAt = now
		}
		if doc.UpdatedAt == 0 {
			doc.UpdatedAt = now
		}
	}

	// Execute indexing operation
	tInfo, err := c.meili.Index(indexName).AddDocuments(docs)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to index %d documents", len(docs))
	}

	return &infra.SearchTaskInfo{
		UID:        tInfo.TaskUID,
		IndexUID:   tInfo.IndexUID,
		Type:       string(tInfo.Type),
		EnqueuedAt: tInfo.EnqueuedAt.Format(time.RFC3339),
		TaskUID:    tInfo.TaskUID,
		Status:     string(tInfo.Status),
	}, nil
}

// WaitForTask waits for a Meilisearch task to complete.
func (c *client) WaitForTask(taskUID int64, timeout time.Duration) (*infra.SearchTask, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second // Default timeout
	}

	task, err := c.meili.WaitForTask(taskUID, timeout)
	if err != nil {
		return nil, eris.Wrapf(err, "wait for task %d timed out after %v", taskUID, timeout)
	}

	// Convert to our task structure
	return &infra.SearchTask{
		Status:     infra.TaskStatus(task.Status),
		TaskUID:    task.TaskUID,
		IndexUID:   task.IndexUID,
		Type:       string(task.Type),
		EnqueuedAt: task.EnqueuedAt,
		Duration:   task.Duration,
		StartedAt:  task.StartedAt,
		FinishedAt: task.FinishedAt,
		CanceledBy: task.CanceledBy,
		Error: infra.SearchTaskError{
			Message: task.Error.Message,
			Code:    task.Error.Code,
			Type:    task.Error.Type,
			Link:    task.Error.Link,
		},
		Details: task.Details,
	}, nil
}

// Search performs a search operation with the provided options.
func (c *client) Search(ctx context.Context, opts *infra.SearchOptions) ([]*infra.SearchDocument, error) {
	if opts == nil {
		return nil, eris.New("search options cannot be nil")
	}

	if opts.Query == "" {
		return nil, eris.New("search query cannot be empty")
	}

	// Apply default values
	c.normalizeSearchOptions(opts)

	// Build the search request
	searchRequest := c.buildSearchRequest(opts)

	// Log the search request (in debug mode only)
	c.l.Debug().
		Str("query", opts.Query).
		Int("limit", opts.Limit).
		Int("offset", opts.Offset).
		Strs("types", opts.Types).
		Interface("filter", searchRequest.Filter).
		Msg("executing search request")

	// Execute the search with context
	res, err := c.meili.Index(c.GetIndexName()).SearchWithContext(ctx, opts.Query, searchRequest)
	if err != nil {
		return nil, eris.Wrap(err, "failed to execute search request")
	}

	// Parse the results
	results := c.processSearchResults(res)

	c.l.Debug().
		Int64("totalHits", res.EstimatedTotalHits).
		Int("processedHits", len(results)).
		Int("offset", opts.Offset).
		Int64("processingTimeMs", res.ProcessingTimeMs).
		Msg("search completed")

	return results, nil
}

// normalizeSearchOptions applies default values to search options
func (c *client) normalizeSearchOptions(opts *infra.SearchOptions) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100 // Prevent too many results at once
	}
}

// buildSearchRequest creates a Meilisearch search request from our options
func (c *client) buildSearchRequest(opts *infra.SearchOptions) *meilisearch.SearchRequest {
	searchRequest := &meilisearch.SearchRequest{
		Query:                opts.Query,
		Limit:                int64(opts.Limit),
		Offset:               int64(opts.Offset),
		AttributesToRetrieve: []string{"*"}, // Retrieve all attributes
		Sort:                 opts.SortBy,
	}

	// Add filters if present
	filter := c.buildSearchFilters(opts)
	if filter != "" {
		searchRequest.Filter = filter
	}

	return searchRequest
}

// buildSearchFilters constructs filter expressions from search options
func (c *client) buildSearchFilters(opts *infra.SearchOptions) string {
	filters := []string{}

	// Add organization and business unit filters
	if opts.OrgID != "" {
		filters = append(filters, fmt.Sprintf("organizationId = %s", opts.OrgID))
	}

	if opts.BuID != "" {
		filters = append(filters, fmt.Sprintf("businessUnitId = %s", opts.BuID))
	}

	// Add type filters if specified
	if len(opts.Types) > 0 {
		typeFilters := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			typeFilters[i] = fmt.Sprintf("type = %s", t)
		}
		filters = append(filters, fmt.Sprintf("(%s)", strings.Join(typeFilters, " OR ")))
	}

	// Add any additional custom filters
	if len(opts.Filters) > 0 {
		filters = append(filters, opts.Filters...)
	}

	// Combine all filters with AND
	if len(filters) > 0 {
		return strings.Join(filters, " AND ")
	}

	return ""
}

// processSearchResults converts Meilisearch hits to our document format
func (c *client) processSearchResults(res *meilisearch.SearchResponse) []*infra.SearchDocument {
	results := make([]*infra.SearchDocument, 0, len(res.Hits))

	// Process hits
	for _, hit := range res.Hits {
		doc, ok := hit.(map[string]any)
		if !ok {
			continue
		}

		var searchDoc infra.SearchDocument

		// Use mapstructure to convert the map to our document structure
		decoderConfig := &mapstructure.DecoderConfig{
			Result:           &searchDoc,
			TagName:          "json",
			WeaklyTypedInput: true,
		}

		decoder, dErr := mapstructure.NewDecoder(decoderConfig)
		if dErr != nil {
			c.l.Error().Err(dErr).Interface("doc", doc).Msg("failed to create decoder for search result")
			continue
		}

		if dErr = decoder.Decode(doc); dErr != nil {
			c.l.Error().Err(dErr).Interface("doc", doc).Msg("failed to decode search result")
			continue
		}

		// Add to results
		results = append(results, &searchDoc)
	}

	return results
}

// GetStats returns index statistics for monitoring.
func (c *client) GetStats() (map[string]any, error) {
	indexName := c.GetIndexName()
	stats, err := c.meili.Index(indexName).GetStats()
	if err != nil {
		return nil, eris.Wrapf(err, "failed to get stats for index %s", indexName)
	}

	result := map[string]any{
		"numberOfDocuments": stats.NumberOfDocuments,
		"isIndexing":        stats.IsIndexing,
		"fieldDistribution": stats.FieldDistribution,
	}

	return result, nil
}

// DeleteDocument removes a document from the search index.
func (c *client) DeleteDocument(id string) (*infra.SearchTaskInfo, error) {
	if id == "" {
		return nil, eris.New("document ID cannot be empty")
	}

	task, err := c.meili.Index(c.GetIndexName()).DeleteDocument(id)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to delete document with ID %s", id)
	}

	return &infra.SearchTaskInfo{
		UID:        task.TaskUID,
		IndexUID:   task.IndexUID,
		Type:       string(task.Type),
		EnqueuedAt: task.EnqueuedAt.Format(time.RFC3339),
		TaskUID:    task.TaskUID,
	}, nil
}

// DeleteDocuments removes multiple documents from the search index.
func (c *client) DeleteDocuments(ids []string) (*infra.SearchTaskInfo, error) {
	if len(ids) == 0 {
		return nil, eris.New("no document IDs provided")
	}

	task, err := c.meili.Index(c.GetIndexName()).DeleteDocuments(ids)
	if err != nil {
		return nil, eris.Wrap(err, "failed to delete documents")
	}

	return &infra.SearchTaskInfo{
		UID:        task.TaskUID,
		IndexUID:   task.IndexUID,
		Type:       string(task.Type),
		EnqueuedAt: task.EnqueuedAt.Format(time.RFC3339),
		TaskUID:    task.TaskUID,
	}, nil
}

func (c *client) GetIndexName() string {
	if c.cfg.IndexPrefix != "" {
		return c.cfg.IndexPrefix + "_global"
	}
	return "global"
}

// SuggestCompletions returns query suggestions for autocomplete.
func (c *client) SuggestCompletions(ctx context.Context, prefix string, limit int, types []string) ([]string, error) {
	if prefix == "" {
		return nil, eris.New("prefix cannot be empty")
	}

	if limit <= 0 {
		limit = 5 // Default limit for suggestions
	}
	if limit > 50 {
		limit = 50 // Cap suggestions at 50
	}

	// Build search options for suggestions
	searchOpts := &infra.SearchOptions{
		Query:  prefix,
		Limit:  limit,
		Offset: 0,
	}

	// Add type filter if specified
	if len(types) > 0 {
		searchOpts.Types = types
	}

	// Use the Search method for prefix search
	docs, err := c.Search(ctx, searchOpts)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to get suggestions for prefix '%s'", prefix)
	}

	// Extract suggestions from search results
	suggestions := make([]string, 0, len(docs))
	for _, doc := range docs {
		// Use title as the suggestion
		suggestions = append(suggestions, doc.Title)
	}

	return suggestions, nil
}
