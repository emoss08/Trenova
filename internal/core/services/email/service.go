package email

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger            *logger.Logger
	Sender            Sender
	QueueProcessor    QueueProcessor
	AttachmentHandler AttachmentHandler
	ProfileService    services.EmailProfileService
	TemplateService   services.EmailTemplateService
	QueueService      services.EmailQueueService
	LogService        services.EmailLogService
}

type Service struct {
	l                 *zerolog.Logger
	sender            Sender
	queueProcessor    QueueProcessor
	attachmentHandler AttachmentHandler
	profileService    services.EmailProfileService
	templateService   services.EmailTemplateService
	queueService      services.EmailQueueService
	logService        services.EmailLogService
}

// NewService creates a new email service
//
//nolint:gocritic // this is for dependency injection
func NewService(p ServiceParams) services.EmailService {
	log := p.Logger.With().
		Str("service", "email").
		Logger()

	return &Service{
		l:                 &log,
		sender:            p.Sender,
		queueProcessor:    p.QueueProcessor,
		attachmentHandler: p.AttachmentHandler,
		profileService:    p.ProfileService,
		templateService:   p.TemplateService,
		queueService:      p.QueueService,
		logService:        p.LogService,
	}
}

// SendEmail sends an email immediately
func (s *Service) SendEmail(
	ctx context.Context,
	req *services.SendEmailRequest,
) (*services.SendEmailResponse, error) {
	log := s.l.With().
		Str("operation", "send_email").
		Str("org_id", req.OrganizationID.String()).
		Logger()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, oops.In("email_service").
			Tags("operation", "validate_request").
			Time(time.Now()).
			Wrapf(err, "invalid request")
	}

	// Get email profile
	profile, err := s.getProfileOrDefault(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:      req.OrganizationID,
		BuID:       req.BusinessUnitID,
		UserID:     req.UserID,
		ProfileID:  pulid.ConvertFromPtr(req.ProfileID),
		ExpandData: false,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get email profile")
		return nil, err
	}

	// Create and save queue entry
	queue, err := s.createQueueEntry(ctx, req, profile.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create queue entry")
		return nil, err
	}

	// Process immediately
	if err = s.queueProcessor.ProcessSingleItem(ctx, queue); err != nil {
		log.Error().Err(err).Msg("failed to send email")
		return nil, oops.In("email_service").
			Tags("operation", "send_immediate").
			Tags("queue_id", queue.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to send email")
	}

	// Get updated queue entry for response
	queue, err = s.queueService.Get(ctx, queue.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get queue status")
		return nil, oops.In("email_service").
			Tags("operation", "get_queue_status").
			Tags("queue_id", queue.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get queue status")
	}

	log.Info().
		Str("queue_id", queue.ID.String()).
		Str("status", string(queue.Status)).
		Msg("email sent successfully")

	return &services.SendEmailResponse{
		QueueID:   queue.ID,
		MessageID: s.getMessageIDFromLogs(ctx, queue.ID),
		Status:    string(queue.Status),
	}, nil
}

// SendTemplatedEmail sends an email using a template
func (s *Service) SendTemplatedEmail(
	ctx context.Context,
	req *services.SendTemplatedEmailRequest,
) (*services.SendEmailResponse, error) {
	log := s.l.With().
		Str("operation", "send_templated_email").
		Str("org_id", req.OrganizationID.String()).
		Str("template_id", req.TemplateID.String()).
		Logger()

	// Validate request
	if err := req.Validate(); err != nil {
		log.Error().Err(err).Msg("invalid templated email request")
		return nil, oops.In("email_service").
			Tags("operation", "validate_templated_request").
			Time(time.Now()).
			Wrapf(err, "invalid request")
	}

	// Get and validate template
	template, err := s.templateService.Get(ctx, req.TemplateID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get email template")
		return nil, oops.In("email_service").
			Tags("operation", "get_template").
			Tags("template_id", req.TemplateID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email template")
	}

	// Validate variables
	if err = s.templateService.ValidateVariables(ctx, template.ID, req.Variables); err != nil {
		log.Error().Err(err).Msg("template variable validation failed")
		return nil, oops.In("email_service").
			Tags("operation", "validate_template_variables").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "template variable validation failed")
	}

	// Render template
	rendered, err := s.templateService.RenderTemplate(ctx, template, req.Variables)
	if err != nil {
		log.Error().Err(err).Msg("failed to render template")
		return nil, oops.In("email_service").
			Tags("operation", "render_template").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to render template")
	}

	// Convert to regular email request
	emailReq := &services.SendEmailRequest{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		ProfileID:      req.ProfileID,
		To:             req.To,
		CC:             req.CC,
		BCC:            req.BCC,
		Subject:        rendered.Subject,
		HTMLBody:       rendered.HTMLBody,
		TextBody:       rendered.TextBody,
		Attachments:    req.Attachments,
		Priority:       req.Priority,
		Metadata:       req.Metadata,
	}

	log.Info().
		Str("subject", rendered.Subject).
		Int("to_count", len(req.To)).
		Msg("sending templated email")

	return s.SendEmail(ctx, emailReq)
}

// QueueEmail adds an email to the queue for later processing
func (s *Service) QueueEmail(ctx context.Context, queue *email.Queue) (*email.Queue, error) {
	// Validate queue entry
	multiErr := errors.NewMultiError()
	queue.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Set default status based on scheduling
	if queue.Status == "" {
		queue.Status = s.determineInitialStatus(queue)
	}

	// Set default priority
	if queue.Priority == "" {
		queue.Priority = email.PriorityMedium
	}

	// Create the queue entry
	return s.queueService.Create(ctx, queue)
}

// ProcessEmailQueue processes pending emails in the queue
func (s *Service) ProcessEmailQueue(ctx context.Context) error {
	return s.queueProcessor.ProcessQueue(ctx)
}

// TestEmailProfile tests an email profile configuration
func (s *Service) TestEmailProfile(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*services.TestEmailProfileResponse, error) {
	log := s.l.With().
		Str("operation", "test_email_profile").
		Str("profile_id", req.ProfileID.String()).
		Logger()

	profile, err := s.profileService.Get(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get email profile")
		return nil, oops.In("email_service").
			Tags("operation", "get_profile_for_test").
			Tags("profile_id", req.ProfileID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email profile")
	}

	// Test the connection
	if err = s.sender.TestConnection(ctx, profile); err != nil {
		log.Warn().
			Err(err).
			Str("provider", string(profile.ProviderType)).
			Str("host", profile.Host).
			Int("port", profile.Port).
			Msg("email profile connection test failed")

		return &services.TestEmailProfileResponse{
			Success: false,
			Message: "Connection test failed: " + err.Error(),
			Details: map[string]any{
				"provider": profile.ProviderType,
				"host":     profile.Host,
				"port":     profile.Port,
				"error":    err.Error(),
			},
		}, nil
	}

	log.Info().
		Str("provider", string(profile.ProviderType)).
		Str("host", profile.Host).
		Int("port", profile.Port).
		Msg("email profile connection test succeeded")

	return &services.TestEmailProfileResponse{
		Success: true,
		Message: "Email profile configuration is valid",
		Details: map[string]any{
			"provider": profile.ProviderType,
			"host":     profile.Host,
			"port":     profile.Port,
		},
	}, nil
}

// LogEmailEvent logs an email event (open, click, bounce, etc.)
func (s *Service) LogEmailEvent(ctx context.Context, log *email.Log) error {
	return s.logEmailEvent(ctx, log)
}

// GetEmailStatus retrieves the status of a queued email
func (s *Service) GetEmailStatus(
	ctx context.Context,
	queueID pulid.ID,
) (*services.EmailStatusResponse, error) {
	log := s.l.With().
		Str("operation", "get_email_status").
		Str("queue_id", queueID.String()).
		Logger()

	// Get queue entry
	queue, err := s.queueService.Get(ctx, queueID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get queue entry")
		return nil, oops.In("email_service").
			Tags("operation", "get_queue_entry").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get queue entry")
	}

	// Get logs
	logs, err := s.logService.GetByQueueID(ctx, queueID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get email logs")
		return nil, oops.In("email_service").
			Tags("operation", "get_email_logs").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email logs")
	}

	return &services.EmailStatusResponse{
		QueueID:      queue.ID,
		Status:       queue.Status,
		SentAt:       queue.SentAt,
		ScheduledAt:  queue.ScheduledAt,
		ErrorMessage: queue.ErrorMessage,
		RetryCount:   queue.RetryCount,
		Logs:         logs,
	}, nil
}

// RetryFailedEmail retries a failed email
func (s *Service) RetryFailedEmail(ctx context.Context, queueID pulid.ID) error {
	log := s.l.With().
		Str("operation", "retry_failed_email").
		Str("queue_id", queueID.String()).
		Logger()

	queue, err := s.queueService.Get(ctx, queueID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get queue entry for retry")
		return oops.In("email_service").
			Tags("operation", "get_queue_for_retry").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get queue entry")
	}

	if !queue.CanRetry() {
		log.Warn().
			Str("status", string(queue.Status)).
			Int("retry_count", queue.RetryCount).
			Msg("email cannot be retried")
		return oops.In("email_service").
			Tags("operation", "check_retry_eligibility").
			Tags("queue_id", queueID.String()).
			Tags("status", string(queue.Status)).
			Time(time.Now()).
			Errorf("email cannot be retried")
	}

	// Reset status to pending
	queue.Status = email.QueueStatusPending
	queue.ErrorMessage = ""

	_, err = s.queueService.Update(ctx, queue)
	if err != nil {
		log.Error().Err(err).Msg("failed to update queue for retry")
		return oops.In("email_service").
			Tags("operation", "update_queue_for_retry").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to update queue entry")
	}

	log.Info().Msg("email queued for retry")
	return nil
}

// CancelScheduledEmail cancels a scheduled email
func (s *Service) CancelScheduledEmail(ctx context.Context, queueID pulid.ID) error {
	log := s.l.With().
		Str("operation", "cancel_scheduled_email").
		Str("queue_id", queueID.String()).
		Logger()

	queue, err := s.queueService.Get(ctx, queueID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get queue entry for cancellation")
		return oops.In("email_service").
			Tags("operation", "get_queue_for_cancel").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get queue entry")
	}

	if queue.Status != email.QueueStatusScheduled {
		log.Warn().
			Str("current_status", string(queue.Status)).
			Msg("cannot cancel non-scheduled email")
		return oops.In("email_service").
			Tags("operation", "check_cancel_eligibility").
			Tags("queue_id", queueID.String()).
			Tags("status", string(queue.Status)).
			Time(time.Now()).
			Errorf("email is not scheduled")
	}

	queue.Status = email.QueueStatusCancelled
	_, err = s.queueService.Update(ctx, queue)
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel scheduled email")
		return oops.In("email_service").
			Tags("operation", "update_queue_for_cancel").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to update queue entry")
	}

	log.Info().Msg("scheduled email cancelled")
	return nil
}

// Helper methods

func (s *Service) getProfileOrDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	if req.ProfileID != pulid.Nil {
		return s.profileService.Get(ctx, req)
	}

	return s.profileService.GetDefault(ctx, req)
}

func (s *Service) createQueueEntry(
	ctx context.Context,
	req *services.SendEmailRequest,
	profileID pulid.ID,
) (*email.Queue, error) {
	log := s.l.With().
		Str("operation", "create_queue_entry").
		Str("org_id", req.OrganizationID.String()).
		Logger()

	// Process attachments if present
	var processedAttachments []email.AttachmentMeta
	if len(req.Attachments) > 0 {
		log.Debug().
			Int("attachment_count", len(req.Attachments)).
			Msg("processing email attachments")

		// Save attachments to storage
		savedAttachments, err := s.attachmentHandler.SaveAttachments(
			ctx,
			req.Attachments,
			req.OrganizationID,
			req.BusinessUnitID, // Using BusinessUnitID as the user context
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to save attachments")
			return nil, oops.In("email_service").
				Tags("operation", "save_attachments").
				Tags("org_id", req.OrganizationID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to save attachments")
		}

		processedAttachments = savedAttachments
		log.Info().
			Int("saved_count", len(savedAttachments)).
			Msg("attachments saved successfully")
	}

	queue := &email.Queue{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		ProfileID:      profileID,
		ToAddresses:    req.To,
		CCAddresses:    req.CC,
		BCCAddresses:   req.BCC,
		Subject:        req.Subject,
		HTMLBody:       req.HTMLBody,
		TextBody:       req.TextBody,
		Attachments:    processedAttachments,
		Priority:       req.Priority,
		Status:         email.QueueStatusPending,
		Metadata:       req.Metadata,
	}

	created, err := s.queueService.Create(ctx, queue)
	if err != nil {
		// If queue creation fails and we saved attachments, try to clean them up
		if len(processedAttachments) > 0 {
			if deleteErr := s.attachmentHandler.DeleteAttachments(ctx, processedAttachments, req.OrganizationID); deleteErr != nil {
				log.Error().
					Err(deleteErr).
					Msg("failed to cleanup attachments after queue creation failure")
			}
		}
		return nil, err
	}

	return created, nil
}

func (s *Service) determineInitialStatus(queue *email.Queue) email.QueueStatus {
	if queue.ScheduledAt != nil && *queue.ScheduledAt > time.Now().Unix() {
		return email.QueueStatusScheduled
	}
	return email.QueueStatusPending
}

func (s *Service) getMessageIDFromLogs(ctx context.Context, queueID pulid.ID) string {
	logs, err := s.logService.GetByQueueID(ctx, queueID)
	if err != nil || len(logs) == 0 {
		return ""
	}
	return logs[0].MessageID
}

func (s *Service) logEmailEvent(ctx context.Context, emailLog *email.Log) error {
	log := s.l.With().
		Str("operation", "log_email_event").
		Str("queue_id", emailLog.QueueID.String()).
		Str("status", string(emailLog.Status)).
		Logger()

	// Get existing logs
	existingLogs, err := s.logService.GetByQueueID(ctx, emailLog.QueueID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing logs")
		return oops.In("email_service").
			Tags("operation", "get_existing_logs").
			Tags("queue_id", emailLog.QueueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get existing logs")
	}

	// Update existing log if found
	if len(existingLogs) > 0 {
		log.Debug().Msg("updating existing log entry")
		return s.updateExistingLog(ctx, existingLogs[0], emailLog)
	}

	// Create new log
	_, err = s.logService.Create(ctx, emailLog)
	if err != nil {
		log.Error().Err(err).Msg("failed to create email log")
		return oops.In("email_service").
			Tags("operation", "create_email_log").
			Tags("queue_id", emailLog.QueueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to create email log")
	}

	log.Info().Msg("email event logged")
	return nil
}

func (s *Service) updateExistingLog(
	ctx context.Context,
	existing *email.Log,
	newEmailLog *email.Log,
) error {
	// Update status
	existing.Status = newEmailLog.Status

	// Update timestamps based on event type
	switch newEmailLog.Status { //nolint:exhaustive // We only support the 4 statuses we need
	case email.LogStatusOpened:
		existing.OpenedAt = newEmailLog.OpenedAt
		existing.UserAgent = newEmailLog.UserAgent
		existing.IPAddress = newEmailLog.IPAddress
	case email.LogStatusClicked:
		existing.ClickedAt = newEmailLog.ClickedAt
		existing.UserAgent = newEmailLog.UserAgent
		existing.IPAddress = newEmailLog.IPAddress
		existing.ClickedURLs = append(existing.ClickedURLs, newEmailLog.ClickedURLs...)
	case email.LogStatusBounced:
		existing.BouncedAt = newEmailLog.BouncedAt
		existing.BounceType = newEmailLog.BounceType
		existing.BounceReason = newEmailLog.BounceReason
	case email.LogStatusComplained:
		existing.ComplainedAt = newEmailLog.ComplainedAt
	case email.LogStatusUnsubscribed:
		existing.UnsubscribedAt = newEmailLog.UnsubscribedAt
	}

	// Add webhook events
	if newEmailLog.WebhookEvents != nil {
		existing.WebhookEvents = append(existing.WebhookEvents, newEmailLog.WebhookEvents...)
	}

	_, err := s.logService.Create(ctx, existing)
	return err
}
