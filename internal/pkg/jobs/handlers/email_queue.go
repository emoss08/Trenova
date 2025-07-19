package handlers

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// EmailQueueHandler handles email queue processing jobs
type EmailQueueHandler struct {
	l            *zerolog.Logger
	emailService services.EmailService
}

// EmailQueueHandlerParams defines dependencies for email queue handler
type EmailQueueHandlerParams struct {
	fx.In

	Logger       *logger.Logger
	EmailService services.EmailService
}

// NewEmailQueueHandler creates a new email queue handler
func NewEmailQueueHandler(p EmailQueueHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "email_queue").
		Logger()

	return &EmailQueueHandler{
		l:            &log,
		emailService: p.EmailService,
	}
}

// JobType returns the job type this handler processes
func (h *EmailQueueHandler) JobType() jobs.JobType {
	return jobs.JobTypeProcessEmailQueue
}

// ProcessTask processes an email queue job
func (h *EmailQueueHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	log := h.l.With().
		Str("job_type", task.Type()).
		Str("task_id", task.ResultWriter().TaskID()).
		Logger()

	log.Info().Msg("processing email queue")

	startTime := time.Now()

	// Process the email queue
	if err := h.emailService.ProcessEmailQueue(ctx); err != nil {
		log.Error().
			Err(err).
			Dur("elapsed", time.Since(startTime)).
			Msg("failed to process email queue")
		return oops.In("email_queue_handler").
			Tags("operation", "process_queue").
			Time(time.Now()).
			Wrapf(err, "failed to process email queue")
	}

	log.Info().
		Dur("processing_time", time.Since(startTime)).
		Msg("email queue processed successfully")

	return nil
}
