package search

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	LC     fx.Lifecycle
	Client infra.SearchClient
	Config *config.Manager
	Logger *logger.Logger
}

type Service struct {
	l      *zerolog.Logger
	client infra.SearchClient
	config *config.SearchConfig

	batchQueues chan batchOpt
	isRunning   atomic.Bool

	// Channels for goroutine control
	stopProcessor chan struct{}
	processorDone chan struct{}
	stopMonitor   chan struct{}
	monitorDone   chan struct{}
}

func NewService(p ServiceParams) (*Service, error) {
	log := p.Logger.With().Str("service", "search").Logger()

	cfg := p.Config.Search()

	service := &Service{
		client:        p.Client,
		l:             &log,
		config:        cfg,
		batchQueues:   make(chan batchOpt, cfg.MaxBatchSize),
		stopProcessor: make(chan struct{}),
		processorDone: make(chan struct{}),
		stopMonitor:   make(chan struct{}),
		monitorDone:   make(chan struct{}),
	}

	p.LC.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return service.Start()
		},
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},
	})

	return service, nil
}

func (s *Service) Start() error {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.l.Warn().Msg("search service is already running")
		return nil
	}

	go func() {
		defer close(s.processorDone)
		s.startBatchProcessor()
	}()

	go func() {
		defer close(s.monitorDone)
		s.monitor()
	}()

	s.l.Info().
		Int("batchSize", s.config.MaxBatchSize).
		Int("batchInterval", s.config.BatchInterval).
		Msg("🚀 Search service initialized")

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.isRunning.CompareAndSwap(true, false) {
		s.l.Warn().Msg("search service is not running")
		return nil
	}

	s.l.Info().Msg("stopping search service")

	close(s.stopProcessor)
	close(s.stopMonitor)

	// Wait for processor with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	monitorOK := make(chan struct{})

	go func() {
		<-s.monitorDone
		close(monitorOK)
	}()

	select {
	case <-shutdownCtx.Done():
		return eris.New("timeout waiting for search service to stop")
	case <-s.processorDone:
		s.l.Info().Msg("search service stopped successfully")
	case <-monitorOK:
		s.l.Info().Msg("search service stopped successfully")
	}

	if err := s.flushBatch(ctx); err != nil {
		s.l.Error().Err(err).Msg("failed to flush batch")
		return eris.Wrap(err, "final batch flush")
	}

	return nil
}

func (s *Service) Index(ctx context.Context, entity infra.SearchableEntity) error {
	if !s.isRunning.Load() {
		return ErrServiceStopped
	}

	doc := entity.ToDocument()

	select {
	case s.batchQueues <- batchOpt{
		documents: []*infra.SearchDocument{&doc},
		callback: func(err error) {
			if err != nil {
				s.l.Error().
					Err(err).
					Str("id", doc.ID).
					Str("type", doc.Type).
					Msg("failed to index document asynchronously")
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
	default:
		// If queue is full, log warning and continue
		s.l.Warn().
			Str("id", doc.ID).
			Str("type", doc.Type).
			Msg("search index queue full, document indexing skipped")
	}

	return nil
}

func (s *Service) Search(ctx context.Context, params *Request) (*Response, error) {
	log := s.l.With().
		Str("operation", "GlobalSearch").
		Str("query", params.Query).
		Int("limit", params.Limit).
		Interface("types", params.Types).
		Logger()

	if params.Query == "" {
		return nil, errors.NewValidationError("query", errors.ErrRequired, "query is required")
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	results, err := s.client.Search(ctx,
		&infra.SearchOptions{
			Query:  params.Query,
			Types:  params.Types,
			Limit:  params.Limit,
			Offset: params.Offset,
			OrgID:  params.OrgID,
			BuID:   params.BuID,
			SortBy: []string{"createdAt:desc", "updatedAt:desc"},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to search")
		return nil, eris.Wrap(err, "search")
	}

	log.Debug().Int("resultCount", len(results)).Msg("search results")

	return &Response{
		Results: results,
		Total:   len(results),
		Query:   params.Query,
	}, nil
}

func (s *Service) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopMonitor:
			s.l.Debug().Msg("monitor stopped due to service shutdown")
			return
		case <-ticker.C:
			s.checkServiceHealth()
		}
	}
}

func (s *Service) checkServiceHealth() {
	bufferSize := len(s.batchQueues)
	if bufferSize > 0 {
		s.l.Info().
			Int("bufferSize", bufferSize).
			Bool("isRunning", s.isRunning.Load()).
			Msg("service health check - documents pending")
	} else {
		s.l.Debug().
			Int("bufferSize", bufferSize).
			Bool("isRunning", s.isRunning.Load()).
			Msg("service health check - no documents pending")
	}
}

func (s *Service) startBatchProcessor() {
	s.l.Debug().
		Int("maxBatchSize", s.config.MaxBatchSize).
		Int("batchInterval", s.config.BatchInterval).
		Msg("starting batch processor")

	batch := make([]*infra.SearchDocument, 0, s.config.MaxBatchSize)
	callbacks := make([]func(error), 0, s.config.MaxBatchSize)

	// Converting seconds to duration and using a shorter interval for testing
	batchInterval := time.Duration(s.config.BatchInterval) * time.Second

	// Create and start the ticker
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	s.l.Debug().
		Str("interval", batchInterval.String()).
		Msg("batch processor timer started")

	// Log initial state
	s.l.Debug().Msg("batch processor ready to receive documents")

	processBatch := func(reason string) {
		if len(batch) == 0 {
			s.l.Debug().
				Str("reason", reason).
				Msg("timer check - no documents to process")
			return
		}

		s.l.Debug().
			Int("batchSize", len(batch)).
			Str("reason", reason).
			Msg("processing batch")

		s.processBatch(batch, callbacks)
		batch = make([]*infra.SearchDocument, 0, s.config.MaxBatchSize)
		callbacks = make([]func(error), 0, s.config.MaxBatchSize)
	}

	lastTickLog := time.Now()

	for {
		select {
		case <-s.stopProcessor:
			s.l.Info().
				Int("remainingBatch", len(batch)).
				Msg("batch processor received stop signal")
			processBatch("shutdown")
			return

		case op := <-s.batchQueues:
			s.l.Debug().
				Int("currentBatchSize", len(batch)).
				Int("newDocuments", len(op.documents)).
				Int("maxBatchSize", s.config.MaxBatchSize).
				Msg("received new documents")

			batch = append(batch, op.documents...)
			for range op.documents {
				callbacks = append(callbacks, op.callback)
			}

			if len(batch) >= s.config.MaxBatchSize {
				processBatch("max size reached")
			}

		case t := <-ticker.C:
			// Log timer ticks at Info level
			timeSinceLastLog := time.Since(lastTickLog)
			s.l.Debug().
				Time("tickTime", t).
				Int("currentBatchSize", len(batch)).
				Str("timeSinceLastTick", timeSinceLastLog.String()).
				Msg("timer tick received")
			lastTickLog = time.Now()

			if !s.isRunning.Load() {
				s.l.Info().Msg("service stopping, exiting batch processor")
				return
			}

			processBatch("timer")
		}
	}
}

func (s *Service) processBatch(docs []*infra.SearchDocument, callbacks []func(error)) {
	start := time.Now()
	s.l.Debug().
		Int("batchSize", len(docs)).
		Msg("starting batch processing")

	// TODO(Wolfred): We need to add a normal index function to the client that wraps meilisearch
	tInfo, err := s.client.IndexDocuments(s.client.GetIndexName(), docs)
	if err != nil {
		s.l.Error().
			Err(err).
			Int("batchSize", len(docs)).
			Msg("failed to add documents to index")
		for _, cb := range callbacks {
			if cb != nil {
				cb(err)
			}
		}
		return
	}

	// Wait for the indexing task to complete
	task, err := s.client.WaitForTask(tInfo.TaskUID, 10*time.Second)
	if err != nil {
		s.l.Error().
			Err(err).
			Interface("taskInfo", tInfo).
			Msg("failed to wait for indexing task")
		for _, cb := range callbacks {
			if cb != nil {
				cb(err)
			}
		}
		return
	}

	s.l.Debug().
		Int("batchSize", len(docs)).
		Int64("taskId", task.TaskUID).
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

func (s *Service) flushBatch(ctx context.Context) error { //nolint: gocognit
	s.l.Debug().Msg("performing final batch flush")

	// Try to drain the batch queue
	documents := make([]*infra.SearchDocument, 0)
	var callbacks []func(error)

	// Drain the queue with a timeout
	drainCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Collect all pending documents
drainLoop:
	for {
		select {
		case <-drainCtx.Done():
			s.l.Warn().Msg("timeout while draining batch queue")
			break drainLoop
		case op, ok := <-s.batchQueues:
			if !ok {
				break drainLoop
			}
			s.l.Debug().Int("batchSize", len(op.documents)).Msg("drained batch")
			documents = append(documents, op.documents...)
			for range op.documents {
				callbacks = append(callbacks, op.callback)
			}
		default:
			// No more items in queue
			break drainLoop
		}
	}

	if len(documents) == 0 {
		s.l.Debug().Msg("no documents to flush")
		return nil
	}

	s.l.Info().
		Int("batchSize", len(documents)).
		Msg("flushing remaining documents")

	// Process the final batch
	_, err := s.client.IndexDocuments(s.client.GetIndexName(), documents)
	if err != nil {
		s.l.Error().
			Err(err).
			Int("batchSize", len(documents)).
			Msg("failed to flush final batch")

		// Notify callbacks of failure
		for _, cb := range callbacks {
			if cb != nil {
				cb(err)
			}
		}
		return eris.Wrap(err, "flush final batch")
	}

	// Notify callbacks of success
	for _, cb := range callbacks {
		if cb != nil {
			cb(nil)
		}
	}

	s.l.Debug().Msg("final batch flushed successfully")

	return nil
}
