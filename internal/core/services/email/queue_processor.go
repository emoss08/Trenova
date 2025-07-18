package email

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// QueueProcessor processes email queue items
type QueueProcessor interface {
	ProcessQueue(ctx context.Context) error
	ProcessSingleItem(ctx context.Context, queueItem *email.Queue) error
}

type queueProcessor struct {
	l            *zerolog.Logger
	sender       Sender
	profileRepo  repositories.EmailProfileRepository
	queueService services.EmailQueueService
	logService   services.EmailLogService
}

type QueueProcessorParams struct {
	fx.In

	Logger       *logger.Logger
	Sender       Sender
	ProfileRepo  repositories.EmailProfileRepository
	QueueService services.EmailQueueService
	LogService   services.EmailLogService
}

// NewQueueProcessor creates a new queue processor
func NewQueueProcessor(p QueueProcessorParams) QueueProcessor {
	log := p.Logger.With().
		Str("component", "queue_processor").
		Logger()

	return &queueProcessor{
		l:            &log,
		sender:       p.Sender,
		profileRepo:  p.ProfileRepo,
		queueService: p.QueueService,
		logService:   p.LogService,
	}
}

// ProcessQueue processes all pending and scheduled emails
func (p *queueProcessor) ProcessQueue(ctx context.Context) error {
	// Get pending emails
	pending, err := p.queueService.GetPending(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to get pending emails: %w", err)
	}

	// Get scheduled emails that are due
	scheduled, err := p.queueService.GetScheduled(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to get scheduled emails: %w", err)
	}

	// Process all emails
	toProcess := append(pending, scheduled...)

	p.l.Info().
		Int("pending", len(pending)).
		Int("scheduled", len(scheduled)).
		Msg("processing email queue")

	for _, item := range toProcess {
		if err := p.ProcessSingleItem(ctx, item); err != nil {
			p.l.Error().
				Err(err).
				Str("queue_id", item.ID.String()).
				Msg("failed to process queue item")
		}
	}

	return nil
}

// ProcessSingleItem processes a single queue item
func (p *queueProcessor) ProcessSingleItem(ctx context.Context, queueItem *email.Queue) error {
	if err := p.updateQueueStatus(ctx, queueItem, email.QueueStatusProcessing); err != nil {
		return err
	}

	profile, err := p.profileRepo.Get(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:      queueItem.OrganizationID,
		BuID:       queueItem.BusinessUnitID,
		ProfileID:  queueItem.ProfileID,
		ExpandData: false,
	})
	if err != nil {
		return p.handleFailure(ctx, queueItem, fmt.Errorf("failed to get email profile: %w", err))
	}

	messageID, err := p.sender.Send(ctx, profile, queueItem)
	if err != nil {
		return p.handleSendError(ctx, queueItem, err)
	}

	if err := p.queueService.MarkAsSent(ctx, queueItem.ID, messageID); err != nil {
		p.l.Error().Err(err).Msg("failed to mark email as sent")
	}

	if err := p.logDelivery(ctx, queueItem, messageID); err != nil {
		p.l.Error().Err(err).Msg("failed to log email delivery")
	}

	return nil
}

// updateQueueStatus updates the queue item status
func (p *queueProcessor) updateQueueStatus(
	ctx context.Context,
	item *email.Queue,
	status email.QueueStatus,
) error {
	item.Status = status
	if _, err := p.queueService.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update queue status: %w", err)
	}
	return nil
}

// handleFailure handles a complete failure (e.g., missing profile)
func (p *queueProcessor) handleFailure(ctx context.Context, item *email.Queue, err error) error {
	if markErr := p.queueService.MarkAsFailed(ctx, item.ID, err.Error()); markErr != nil {
		p.l.Error().Err(markErr).Msg("failed to mark email as failed")
	}
	return err
}

// handleSendError handles errors during email sending
func (p *queueProcessor) handleSendError(ctx context.Context, item *email.Queue, err error) error {
	// Check if we should retry
	if item.CanRetry() {
		if retryErr := p.queueService.IncrementRetryCount(ctx, item.ID); retryErr != nil {
			p.l.Error().Err(retryErr).Msg("failed to increment retry count")
		}
		return oops.In("queue_processor").
			Tags("operation", "send_retry").
			Tags("queue_id", item.ID.String()).
			Time(time.Now()).
			Wrapf(err, "email send failed (will retry)")
	}

	// Max retries reached, mark as failed
	if markErr := p.queueService.MarkAsFailed(ctx, item.ID, err.Error()); markErr != nil {
		p.l.Error().Err(markErr).Msg("failed to mark email as failed")
	}

	return oops.In("queue_processor").
		Tags("operation", "send_failed").
		Tags("queue_id", item.ID.String()).
		Tags("max_retries", "reached").
		Time(time.Now()).
		Wrapf(err, "email send failed (max retries reached)")
}

// logDelivery creates a delivery log entry
func (p *queueProcessor) logDelivery(
	ctx context.Context,
	queueItem *email.Queue,
	messageID string,
) error {
	log := &email.Log{
		OrganizationID: queueItem.OrganizationID,
		BusinessUnitID: queueItem.BusinessUnitID,
		QueueID:        queueItem.ID,
		MessageID:      messageID,
		Status:         email.LogStatusDelivered,
	}

	_, err := p.logService.Create(ctx, log)
	return err
}
