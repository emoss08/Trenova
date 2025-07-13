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

// EmailHandler handles email jobs
type EmailHandler struct {
	l            *zerolog.Logger
	emailService services.EmailService
}

// EmailHandlerParams defines dependencies for email handler
type EmailHandlerParams struct {
	fx.In

	Logger       *logger.Logger
	EmailService services.EmailService
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(p EmailHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "email").
		Logger()

	return &EmailHandler{
		l:            &log,
		emailService: p.EmailService,
	}
}

// JobType returns the job type this handler processes
func (h *EmailHandler) JobType() jobs.JobType {
	return jobs.JobTypeSendEmail
}

// ProcessTask processes an email job
func (h *EmailHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	log := h.l.With().
		Str("job_type", task.Type()).
		Str("task_id", task.ResultWriter().TaskID()).
		Logger()

	log.Info().Msg("processing email job")

	// Unmarshal payload
	var payload jobs.SendEmailPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal email payload")
		return oops.In("email_handler").
			Tags("operation", "unmarshal_payload").
			Time(time.Now()).
			Wrapf(err, "failed to unmarshal email payload")
	}

	// Log job details
	log = log.With().
		Str("job_id", payload.JobID).
		Str("org_id", payload.OrganizationID.String()).
		Str("email_type", payload.EmailType).
		Int("recipient_count", len(payload.Request.To)).
		Logger()

	// Process based on email type
	switch payload.EmailType {
	case "regular":
		return h.processSendEmail(ctx, &payload, &log)
	case "templated":
		return h.processSendTemplatedEmail(ctx, &payload, &log)
	default:
		err := oops.In("email_handler").
			Tags("operation", "process_email").
			Tags("email_type", payload.EmailType).
			Time(time.Now()).
			Errorf("unknown email type: %s", payload.EmailType)
		log.Error().Err(err).Msg("unknown email type")
		return err
	}
}

// processSendEmail processes a regular email send request
func (h *EmailHandler) processSendEmail(ctx context.Context, payload *jobs.SendEmailPayload, log *zerolog.Logger) error {
	log.Info().
		Str("subject", payload.Request.Subject).
		Strs("to", payload.Request.To).
		Msg("sending regular email")

	startTime := time.Now()

	// Send email
	resp, err := h.emailService.SendEmail(ctx, payload.Request)
	if err != nil {
		log.Error().
			Err(err).
			Dur("elapsed", time.Since(startTime)).
			Msg("failed to send email")
		return oops.In("email_handler").
			Tags("operation", "send_email").
			Tags("org_id", payload.OrganizationID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to send email")
	}

	log.Info().
		Str("queue_id", resp.QueueID.String()).
		Str("message_id", resp.MessageID).
		Str("status", resp.Status).
		Dur("send_time", time.Since(startTime)).
		Msg("email sent successfully")

	return nil
}

// processSendTemplatedEmail processes a templated email send request
func (h *EmailHandler) processSendTemplatedEmail(ctx context.Context, payload *jobs.SendEmailPayload, log *zerolog.Logger) error {
	if payload.TemplatedRequest == nil {
		err := oops.In("email_handler").
			Tags("operation", "validate_templated_request").
			Time(time.Now()).
			Errorf("templated request is nil")
		log.Error().Err(err).Msg("templated request missing")
		return err
	}

	log.Info().
		Str("template_id", payload.TemplatedRequest.TemplateID.String()).
		Strs("to", payload.TemplatedRequest.To).
		Msg("sending templated email")

	startTime := time.Now()

	// Send templated email
	resp, err := h.emailService.SendTemplatedEmail(ctx, payload.TemplatedRequest)
	if err != nil {
		log.Error().
			Err(err).
			Dur("elapsed", time.Since(startTime)).
			Msg("failed to send templated email")
		return oops.In("email_handler").
			Tags("operation", "send_templated_email").
			Tags("org_id", payload.OrganizationID.String()).
			Tags("template_id", payload.TemplatedRequest.TemplateID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to send templated email")
	}

	log.Info().
		Str("queue_id", resp.QueueID.String()).
		Str("message_id", resp.MessageID).
		Str("status", resp.Status).
		Dur("send_time", time.Since(startTime)).
		Msg("templated email sent successfully")

	return nil
}