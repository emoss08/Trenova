package search

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/config"
	apperrors "github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

// ServiceParams are the dependencies required by the search service
type ServiceParams struct {
	fx.In

	LC     fx.Lifecycle
	Client infra.SearchClient
	Config *config.Manager
	Logger *logger.Logger
}

// Service handles search operations and document indexing
type Service struct {
	l      *zerolog.Logger
	client infra.SearchClient
	config *config.SearchConfig

	batchQueue     chan batchOpt
	isRunning      atomic.Bool
	processingLock sync.Mutex

	// Control channels for goroutines
	shutdownCh    chan struct{}
	processorDone chan struct{}
	monitorDone   chan struct{}
}

// NewService creates a new search service with the provided dependencies
func NewService(p ServiceParams) (*Service, error) {
	log := p.Logger.With().Str("service", "search").Logger()

	cfg := p.Config.Search()
	if cfg == nil {
		return nil, eris.New("search configuration is nil")
	}

	// Set sensible defaults for configuration
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 100
		log.Warn().Int("maxBatchSize", cfg.MaxBatchSize).Msg("using default max batch size")
	}

	if cfg.BatchInterval <= 0 {
		cfg.BatchInterval = 5
		log.Warn().Int("batchInterval", cfg.BatchInterval).Msg("using default batch interval")
	}

	service := &Service{
		client:        p.Client,
		l:             &log,
		config:        cfg,
		batchQueue:    make(chan batchOpt, cfg.MaxBatchSize*2), // Buffer twice the batch size
		shutdownCh:    make(chan struct{}),
		processorDone: make(chan struct{}),
		monitorDone:   make(chan struct{}),
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},
	})

	return service, nil
}

// Start initializes and starts the search service
func (s *Service) Start(ctx context.Context) error {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.l.Warn().Msg("search service is already running")
		return nil
	}

	// Perform any initialization checks
	if s.client == nil {
		return eris.New("search client is nil")
	}

	// Start the batch processor
	go func() {
		defer close(s.processorDone)
		s.startBatchProcessor()
	}()

	// Start the health monitor
	go func() {
		defer close(s.monitorDone)
		s.monitorHealth()
	}()

	s.l.Info().
		Int("maxBatchSize", s.config.MaxBatchSize).
		Int("batchInterval", s.config.BatchInterval).
		Str("indexName", s.client.GetIndexName()).
		Msg("🚀 Search service started successfully")

	return nil
}

// Stop gracefully shuts down the search service
func (s *Service) Stop(ctx context.Context) error {
	if !s.isRunning.CompareAndSwap(true, false) {
		s.l.Warn().Msg("search service is already stopped")
		return nil
	}

	s.l.Info().Msg("stopping search service")

	// Signal all goroutines to stop
	close(s.shutdownCh)

	// Wait for processor and monitor with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Wait group to track both goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Wait for processor to finish
	go func() {
		defer wg.Done()
		select {
		case <-shutdownCtx.Done():
			s.l.Warn().Msg("timeout waiting for processor to stop")
		case <-s.processorDone:
			s.l.Debug().Msg("processor stopped successfully")
		}
	}()

	// Wait for monitor to finish
	go func() {
		defer wg.Done()
		select {
		case <-shutdownCtx.Done():
			s.l.Warn().Msg("timeout waiting for monitor to stop")
		case <-s.monitorDone:
			s.l.Debug().Msg("monitor stopped successfully")
		}
	}()

	// Wait for both goroutines or timeout
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-shutdownCtx.Done():
		return eris.Wrap(shutdownCtx.Err(), "timeout waiting for search service to stop")
	case <-doneCh:
		// Both goroutines are stopped, now flush any remaining documents
		if err := s.flushBatch(ctx); err != nil {
			s.l.Error().Err(err).Msg("failed to flush final batch")
			return eris.Wrap(err, "final batch flush")
		}
		s.l.Info().Msg("search service stopped successfully")
	}

	return nil
}

// Index adds a searchable entity to the search index.
// The operation is asynchronous and will be batched with other indexing operations.
func (s *Service) Index(ctx context.Context, entity infra.SearchableEntity) error {
	if !s.isRunning.Load() {
		return ErrServiceStopped
	}

	if entity == nil {
		return apperrors.NewValidationError("entity", apperrors.ErrRequired, "entity is required")
	}

	doc := entity.ToDocument()

	// Validate document
	if doc.ID == "" {
		return apperrors.NewValidationError("id", apperrors.ErrRequired, "document ID is required")
	}

	if doc.Type == "" {
		return apperrors.NewValidationError("type", apperrors.ErrRequired, "document type is required")
	}

	// Add to batch queue with timeout context
	select {
	case s.batchQueue <- batchOpt{
		documents: []*infra.SearchDocument{&doc},
		callback: func(err error) {
			if err != nil {
				s.l.Error().
					Err(err).
					Str("id", doc.ID).
					Str("type", doc.Type).
					Msg("failed to index document")
			} else {
				s.l.Debug().
					Str("id", doc.ID).
					Str("type", doc.Type).
					Msg("document indexed successfully")
			}
		},
	}:
		s.l.Debug().
			Str("id", doc.ID).
			Str("type", doc.Type).
			Msg("document queued for indexing")
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// IndexBatch adds multiple searchable entities to the search index in a single batch.
func (s *Service) IndexBatch(ctx context.Context, entities []infra.SearchableEntity) error {
	if !s.isRunning.Load() {
		return ErrServiceStopped
	}

	if len(entities) == 0 {
		return nil
	}

	docs := make([]*infra.SearchDocument, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		doc := entity.ToDocument()
		docs = append(docs, &doc)
	}

	if len(docs) == 0 {
		return nil
	}

	// Add to batch queue with timeout context
	select {
	case s.batchQueue <- batchOpt{
		documents: docs,
		callback: func(err error) {
			if err != nil {
				s.l.Error().
					Err(err).
					Int("count", len(docs)).
					Msg("failed to index document batch")
			} else {
				s.l.Debug().
					Int("count", len(docs)).
					Msg("document batch indexed successfully")
			}
		},
	}:
		s.l.Debug().
			Int("count", len(docs)).
			Msg("document batch queued for indexing")
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Search performs a search operation with the provided parameters.
func (s *Service) Search(ctx context.Context, params *SearchRequest) (*SearchResponse, error) {
	if !s.isRunning.Load() {
		return nil, ErrServiceStopped
	}

	start := time.Now()
	log := s.l.With().
		Str("operation", "Search").
		Str("query", params.Query).
		Int("limit", params.Limit).
		Interface("types", params.Types).
		Logger()

	// Validate request parameters
	if params.Query == "" {
		return nil, apperrors.NewValidationError("query", apperrors.ErrRequired, "query is required")
	}

	// Apply defaults
	if params.Limit <= 0 {
		params.Limit = 20
	}

	if params.Limit > 100 {
		params.Limit = 100 // Prevent large result sets
	}

	// Prepare search options
	searchOpts := &infra.SearchOptions{
		Query:  params.Query,
		Types:  params.Types,
		Limit:  params.Limit,
		Offset: params.Offset,
		OrgID:  params.OrgID,
		BuID:   params.BuID,
		SortBy: params.SortBy,
		Facets: params.Facets,
	}

	// Add filter if specified
	if params.Filter != "" {
		searchOpts.Filters = []string{params.Filter}
	}

	// Add highlighting if requested
	if params.Highlight {
		searchOpts.Highlight = []string{"title", "description", "searchableText"}
	}

	// If no sort options provided, use defaults
	if len(searchOpts.SortBy) == 0 {
		searchOpts.SortBy = []string{"createdAt:desc", "updatedAt:desc"}
	}

	// Execute search
	results, err := s.client.Search(ctx, searchOpts)
	if err != nil {
		log.Error().Err(err).Msg("search failed")
		return nil, eris.Wrap(err, "search execution")
	}

	duration := time.Since(start)
	log.Debug().
		Int("resultCount", len(results)).
		Dur("duration", duration).
		Msg("search completed")

	// Build response
	response := &SearchResponse{
		Results:     results,
		Total:       len(results),
		Query:       params.Query,
		ProcessedIn: duration,
	}

	// TODO: Process facets when Meilisearch client implements them
	// This is a placeholder for processing facet information

	return response, nil
}

// SearchByType performs a search operation for a specific entity type.
func (s *Service) SearchByType(ctx context.Context, entityType string, query string, limit int, offset int) (*SearchResponse, error) {
	if !s.isRunning.Load() {
		return nil, ErrServiceStopped
	}

	if entityType == "" {
		return nil, apperrors.NewValidationError("entityType", apperrors.ErrRequired, "entity type is required")
	}

	// Get request info from context
	// Note: In a real implementation, you'd extract this from the context or other auth mechanism
	var orgID, buID string
	if reqCtx, ok := ctx.Value("requestContext").(struct{ OrgID, BuID string }); ok {
		orgID = reqCtx.OrgID
		buID = reqCtx.BuID
	}

	// Create and execute search request
	searchReq := &SearchRequest{
		Query:  query,
		Types:  []string{entityType},
		Limit:  limit,
		Offset: offset,
		OrgID:  orgID,
		BuID:   buID,
	}

	return s.Search(ctx, searchReq)
}

// DeleteDocument removes a document from the search index.
func (s *Service) DeleteDocument(ctx context.Context, id string, entityType string) error {
	if !s.isRunning.Load() {
		return ErrServiceStopped
	}

	if id == "" {
		return apperrors.NewValidationError("id", apperrors.ErrRequired, "document ID is required")
	}

	s.l.Debug().
		Str("id", id).
		Str("type", entityType).
		Msg("deleting document from search index")

	// Delete from index
	taskInfo, err := s.client.DeleteDocument(id)
	if err != nil {
		s.l.Error().
			Err(err).
			Str("id", id).
			Str("type", entityType).
			Msg("failed to delete document")
		return eris.Wrapf(err, "failed to delete document with ID %s", id)
	}

	// Wait for task completion
	task, err := s.client.WaitForTask(taskInfo.TaskUID, 10*time.Second)
	if err != nil {
		s.l.Error().
			Err(err).
			Str("id", id).
			Int64("taskUid", taskInfo.TaskUID).
			Msg("failed to wait for delete task")
		return eris.Wrap(err, "waiting for delete task")
	}

	if task.Status != "succeeded" {
		errMsg := fmt.Sprintf("delete task failed with status: %s", task.Status)
		if task.Error.Message != "" {
			errMsg = fmt.Sprintf("%s - %s", errMsg, task.Error.Message)
		}
		err = eris.New(errMsg)
		s.l.Error().
			Err(err).
			Str("id", id).
			Int64("taskUid", task.TaskUID).
			Str("status", task.Status).
			Interface("error", task.Error).
			Msg("document deletion failed")
		return err
	}

	s.l.Debug().
		Str("id", id).
		Str("type", entityType).
		Int64("taskUid", task.TaskUID).
		Msg("document deleted successfully")

	return nil
}

// GetSuggestions returns query completion suggestions.
func (s *Service) GetSuggestions(ctx context.Context, prefix string, limit int, types []string) ([]string, error) {
	if !s.isRunning.Load() {
		return nil, ErrServiceStopped
	}

	if prefix == "" {
		return nil, apperrors.NewValidationError("prefix", apperrors.ErrRequired, "prefix is required")
	}

	s.l.Debug().
		Str("prefix", prefix).
		Int("limit", limit).
		Interface("types", types).
		Msg("getting search suggestions")

	// Use client's suggestion method
	return s.client.SuggestCompletions(ctx, prefix, limit, types)
}

// GetHealth returns the current health status of the search service.
func (s *Service) GetHealth(ctx context.Context) (map[string]interface{}, error) {
	if !s.isRunning.Load() {
		return map[string]interface{}{
			"status":  "stopped",
			"healthy": false,
		}, nil
	}

	// Get stats from the client
	stats, err := s.client.GetStats()
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get search stats")
		return map[string]interface{}{
			"status":  "degraded",
			"healthy": false,
			"error":   err.Error(),
		}, err
	}

	// Get queue stats
	queueSize := len(s.batchQueue)
	queueCapacity := cap(s.batchQueue)
	queueUtilization := float64(queueSize) / float64(queueCapacity) * 100.0

	// Build health response
	health := map[string]interface{}{
		"status":           "running",
		"healthy":          true,
		"queueSize":        queueSize,
		"queueCapacity":    queueCapacity,
		"queueUtilization": queueUtilization,
		"statistics":       stats,
	}

	return health, nil
}

// monitorHealth periodically checks the health of the search service.
func (s *Service) monitorHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownCh:
			s.l.Debug().Msg("health monitor stopped due to service shutdown")
			return
		case <-ticker.C:
			s.checkServiceHealth()
		}
	}
}

// checkServiceHealth logs the current state of the search service.
func (s *Service) checkServiceHealth() {
	queueSize := len(s.batchQueue)
	queueCapacity := cap(s.batchQueue)
	queueUtilization := float64(queueSize) / float64(queueCapacity) * 100.0

	// Log based on queue utilization
	if queueUtilization > 80.0 {
		s.l.Warn().
			Int("queueSize", queueSize).
			Int("queueCapacity", queueCapacity).
			Float64("utilizationPercent", queueUtilization).
			Bool("isRunning", s.isRunning.Load()).
			Msg("search queue near capacity")
	} else if queueSize > 0 {
		s.l.Info().
			Int("queueSize", queueSize).
			Int("queueCapacity", queueCapacity).
			Float64("utilizationPercent", queueUtilization).
			Bool("isRunning", s.isRunning.Load()).
			Msg("search health check - documents pending")
	} else {
		s.l.Debug().
			Int("queueSize", queueSize).
			Int("queueCapacity", queueCapacity).
			Bool("isRunning", s.isRunning.Load()).
			Msg("search health check - queue empty")
	}
}

// startBatchProcessor handles the batching and processing of document indexing operations.
func (s *Service) startBatchProcessor() {
	s.l.Debug().
		Int("maxBatchSize", s.config.MaxBatchSize).
		Int("batchInterval", s.config.BatchInterval).
		Msg("starting batch processor")

	batch := make([]*infra.SearchDocument, 0, s.config.MaxBatchSize)
	callbacks := make([]func(error), 0, s.config.MaxBatchSize)

	// Convert seconds to duration
	batchInterval := time.Duration(s.config.BatchInterval) * time.Second

	// Create and start the ticker
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	s.l.Debug().Str("interval", batchInterval.String()).Msg("batch processor started")

	// Function to process the current batch
	processBatch := func(reason string) {
		if len(batch) == 0 {
			s.l.Debug().Str("reason", reason).Msg("no documents to process")
			return
		}

		s.l.Debug().
			Int("batchSize", len(batch)).
			Str("reason", reason).
			Msg("processing batch")

		// Process batch with mutex to prevent concurrent processing
		s.processingLock.Lock()
		s.processBatch(batch, callbacks)
		s.processingLock.Unlock()

		// Clear the batch and callbacks
		batch = make([]*infra.SearchDocument, 0, s.config.MaxBatchSize)
		callbacks = make([]func(error), 0, s.config.MaxBatchSize)
	}

	// Main processing loop
	for {
		select {
		case <-s.shutdownCh:
			s.l.Info().
				Int("remainingBatch", len(batch)).
				Msg("batch processor received shutdown signal")
			processBatch("shutdown")
			return

		case op := <-s.batchQueue:
			s.l.Debug().
				Int("currentBatchSize", len(batch)).
				Int("newDocuments", len(op.documents)).
				Msg("received new documents")

			batch = append(batch, op.documents...)
			for range op.documents {
				callbacks = append(callbacks, op.callback)
			}

			// Process batch if it reaches max size
			if len(batch) >= s.config.MaxBatchSize {
				processBatch("max size reached")
			}

		case <-ticker.C:
			// Process batch on ticker
			if !s.isRunning.Load() {
				s.l.Info().Msg("service stopping, exiting batch processor")
				return
			}
			processBatch("scheduled")
		}
	}
}

// processBatch sends a batch of documents to the search index.
func (s *Service) processBatch(docs []*infra.SearchDocument, callbacks []func(error)) {
	if len(docs) == 0 {
		return
	}

	start := time.Now()
	s.l.Debug().Int("batchSize", len(docs)).Msg("processing batch")

	// Index the documents
	taskInfo, err := s.client.IndexDocuments(s.client.GetIndexName(), docs)
	if err != nil {
		s.l.Error().
			Err(err).
			Int("batchSize", len(docs)).
			Msg("failed to index documents")
		for _, cb := range callbacks {
			if cb != nil {
				cb(eris.Wrap(err, "indexing documents"))
			}
		}
		return
	}

	// Wait for the task to complete
	taskTimeout := 15 * time.Second
	task, err := s.client.WaitForTask(taskInfo.TaskUID, taskTimeout)
	if err != nil {
		s.l.Error().
			Err(err).
			Int64("taskUid", taskInfo.TaskUID).
			Msg("failed waiting for task")
		for _, cb := range callbacks {
			if cb != nil {
				cb(eris.Wrap(err, "waiting for task"))
			}
		}
		return
	}

	// Check task status
	if task.Status != "succeeded" {
		errMsg := fmt.Sprintf("task failed with status: %s", task.Status)
		if task.Error.Message != "" {
			errMsg = fmt.Sprintf("%s - %s", errMsg, task.Error.Message)
		}
		err = eris.New(errMsg)
		s.l.Error().
			Err(err).
			Int64("taskUid", task.TaskUID).
			Str("status", task.Status).
			Interface("error", task.Error).
			Msg("indexing task failed")
		for _, cb := range callbacks {
			if cb != nil {
				cb(err)
			}
		}
		return
	}

	// Log success
	s.l.Debug().
		Int("batchSize", len(docs)).
		Int64("taskUid", task.TaskUID).
		Str("status", task.Status).
		Str("duration", task.Duration).
		Float64("processingTimeMs", float64(time.Since(start).Milliseconds())).
		Msg("batch processed successfully")

	// Call callbacks with success
	for _, cb := range callbacks {
		if cb != nil {
			cb(nil)
		}
	}
}

// flushBatch processes any remaining documents in the queue before shutdown.
func (s *Service) flushBatch(ctx context.Context) error {
	s.l.Debug().Msg("performing final batch flush")

	// Try to drain the batch queue
	documents := make([]*infra.SearchDocument, 0)
	var callbacks []func(error)

	// Use a timeout for draining
	drainCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Collect all pending documents
drainLoop:
	for {
		select {
		case <-drainCtx.Done():
			s.l.Warn().Msg("timeout while draining batch queue")
			break drainLoop
		case op, ok := <-s.batchQueue:
			if !ok {
				s.l.Debug().Msg("batch queue closed")
				break drainLoop
			}
			s.l.Debug().Int("documentsAdded", len(op.documents)).Msg("drained documents from queue")
			documents = append(documents, op.documents...)
			for range op.documents {
				callbacks = append(callbacks, op.callback)
			}
		default:
			// No more items in queue
			s.l.Debug().Msg("no more documents in queue")
			break drainLoop
		}
	}

	if len(documents) == 0 {
		s.l.Debug().Msg("no documents to flush")
		return nil
	}

	s.l.Info().Int("batchSize", len(documents)).Msg("flushing remaining documents")

	// Process the final batch
	indexName := s.client.GetIndexName()
	taskInfo, err := s.client.IndexDocuments(indexName, documents)
	if err != nil {
		s.l.Error().
			Err(err).
			Int("batchSize", len(documents)).
			Msg("failed to flush final batch")

		// Notify callbacks of failure
		for _, cb := range callbacks {
			if cb != nil {
				cb(eris.Wrap(err, "final batch indexing"))
			}
		}
		return eris.Wrap(err, "flush final batch")
	}

	// Wait with a shorter timeout since we're shutting down
	task, err := s.client.WaitForTask(taskInfo.TaskUID, 5*time.Second)
	if err != nil {
		s.l.Error().
			Err(err).
			Int64("taskUid", taskInfo.TaskUID).
			Msg("failed waiting for final batch task")

		// Notify callbacks of failure
		for _, cb := range callbacks {
			if cb != nil {
				cb(eris.Wrap(err, "waiting for final task"))
			}
		}
		return eris.Wrap(err, "wait for final task")
	}

	if task.Status != "succeeded" {
		errMsg := fmt.Sprintf("final task failed with status: %s", task.Status)
		if task.Error.Message != "" {
			errMsg = fmt.Sprintf("%s - %s", errMsg, task.Error.Message)
		}
		err = eris.New(errMsg)
		s.l.Error().
			Err(err).
			Int64("taskUid", task.TaskUID).
			Str("status", task.Status).
			Interface("error", task.Error).
			Msg("final indexing task failed")

		// Notify callbacks of failure
		for _, cb := range callbacks {
			if cb != nil {
				cb(err)
			}
		}
		return eris.Wrap(err, "final task failed")
	}

	// Notify callbacks of success
	for _, cb := range callbacks {
		if cb != nil {
			cb(nil)
		}
	}

	s.l.Info().
		Int("batchSize", len(documents)).
		Int64("taskUid", task.TaskUID).
		Msg("final batch flushed successfully")

	return nil
}
