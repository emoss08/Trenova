// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package audit

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc"
)

// EntryQueue represents a thread-safe queue for audit entries
type EntryQueue struct {
	entries      chan *audit.Entry
	batchSize    int
	flushTimeout time.Duration
	processor    EntryProcessor
	wg           *conc.WaitGroup
	stopCh       chan struct{}
	logger       *zerolog.Logger
}

// EntryProcessor defines the interface for processing audit entries
type EntryProcessor interface {
	ProcessBatch(ctx context.Context, entries []*audit.Entry) error
}

// QueueConfig holds configuration for the entry queue
type QueueConfig struct {
	BufferSize   int
	BatchSize    int
	FlushTimeout time.Duration
	Workers      int
}

// NewEntryQueue creates a new audit entry queue
func NewEntryQueue(cfg QueueConfig, processor EntryProcessor, lg *logger.Logger) *EntryQueue {
	log := lg.With().Str("component", "audit_queue").Logger()

	return &EntryQueue{
		entries:      make(chan *audit.Entry, cfg.BufferSize),
		batchSize:    cfg.BatchSize,
		flushTimeout: cfg.FlushTimeout,
		processor:    processor,
		wg:           conc.NewWaitGroup(),
		stopCh:       make(chan struct{}),
		logger:       &log,
	}
}

// Enqueue adds an entry to the queue
func (q *EntryQueue) Enqueue(entry *audit.Entry) error {
	select {
	case q.entries <- entry:
		return nil
	default:
		// * Queue is full, return error
		return ErrQueueFull
	}
}

// EnqueueWithTimeout adds an entry with a timeout
func (q *EntryQueue) EnqueueWithTimeout(entry *audit.Entry, timeout time.Duration) error {
	select {
	case q.entries <- entry:
		return nil
	case <-time.After(timeout):
		return ErrQueueTimeout
	}
}

// Start begins processing entries
func (q *EntryQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Go(q.worker)
	}
}

// Stop gracefully shuts down the queue
func (q *EntryQueue) Stop(ctx context.Context) error {
	q.logger.Info().Msg("stopping audit queue")

	// * Signal workers to stop
	close(q.stopCh)

	// * Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// * Process any remaining entries
		remaining := q.drainQueue()
		if len(remaining) > 0 {
			if err := q.processor.ProcessBatch(ctx, remaining); err != nil {
				q.logger.Error().
					Err(err).
					Int("count", len(remaining)).
					Msg("failed to process remaining entries")
				return err
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// worker processes entries from the queue
func (q *EntryQueue) worker() {
	batch := make([]*audit.Entry, 0, q.batchSize)
	ticker := time.NewTicker(q.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			// * Process any remaining batch before exiting
			if len(batch) > 0 {
				q.processBatch(batch)
			}
			return

		case entry := <-q.entries:
			batch = append(batch, entry)
			if len(batch) >= q.batchSize {
				q.processBatch(batch)
				batch = make([]*audit.Entry, 0, q.batchSize)
				ticker.Reset(q.flushTimeout)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				q.processBatch(batch)
				batch = make([]*audit.Entry, 0, q.batchSize)
			}
		}
	}
}

// processBatch processes a batch of entries
func (q *EntryQueue) processBatch(entries []*audit.Entry) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := q.processor.ProcessBatch(ctx, entries); err != nil {
		q.logger.Error().
			Err(err).
			Int("batch_size", len(entries)).
			Msg("failed to process batch")

		// * TODO: Implement dead letter queue or fallback mechanism
	}
}

// drainQueue returns all remaining entries in the queue
func (q *EntryQueue) drainQueue() []*audit.Entry {
	var remaining []*audit.Entry

	for {
		select {
		case entry := <-q.entries:
			remaining = append(remaining, entry)
		default:
			return remaining
		}
	}
}

// QueueStats returns statistics about the queue
func (q *EntryQueue) QueueStats() QueueStatistics {
	return QueueStatistics{
		QueuedEntries: len(q.entries),
		QueueCapacity: cap(q.entries),
	}
}

// QueueStatistics holds queue statistics
type QueueStatistics struct {
	QueuedEntries int
	QueueCapacity int
}
