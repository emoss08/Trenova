package audit

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

// BatchProcessor processes batches of audit entries
type BatchProcessor struct {
	repo         repositories.AuditRepository
	logger       *zerolog.Logger
	maxRetries   int
	retryBackoff time.Duration
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(repo repositories.AuditRepository, lg *logger.Logger) *BatchProcessor {
	log := lg.With().Str("component", "audit_processor").Logger()

	return &BatchProcessor{
		repo:         repo,
		logger:       &log,
		maxRetries:   3,
		retryBackoff: 100 * time.Millisecond,
	}
}

// ProcessBatch processes a batch of audit entries with retry logic
func (p *BatchProcessor) ProcessBatch(ctx context.Context, entries []*audit.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	p.logger.Debug().
		Int("batch_size", len(entries)).
		Msg("processing audit batch")

	// * TODO: Implement retry logic with exponential backoff
	var lastErr error
	backoff := p.retryBackoff

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			}
		}

		err := p.repo.InsertAuditEntries(ctx, entries)
		if err == nil {
			p.logger.Debug().
				Int("batch_size", len(entries)).
				Int("attempt", attempt+1).
				Msg("successfully processed audit batch")
			return nil
		}

		lastErr = err
		p.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_retries", p.maxRetries).
			Msg("failed to process batch, retrying")
	}

	return eris.Wrap(lastErr, "exhausted all retry attempts")
}

// ProcessorMetrics holds metrics for the processor
type ProcessorMetrics struct {
	ProcessedBatches  int64
	ProcessedEntries  int64
	FailedBatches     int64
	RetryAttempts     int64
	AverageLatency    time.Duration
	LastProcessedTime time.Time
}
