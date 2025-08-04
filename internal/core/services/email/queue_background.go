/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// BackgroundEmailService provides methods for queueing emails as background jobs
type BackgroundEmailService interface {
	QueueEmail(
		ctx context.Context,
		req *services.SendEmailRequest,
		opts *services.JobOptions,
	) (*asynq.TaskInfo, error)
	QueueTemplatedEmail(
		ctx context.Context,
		req *services.SendTemplatedEmailRequest,
		opts *services.JobOptions,
	) (*asynq.TaskInfo, error)
}

type backgroundEmailService struct {
	l          *zerolog.Logger
	jobService services.JobService
}

type BackgroundEmailServiceParams struct {
	fx.In

	Logger     *logger.Logger
	JobService services.JobService
}

// NewBackgroundEmailService creates a new background email service
func NewBackgroundEmailService(p BackgroundEmailServiceParams) BackgroundEmailService {
	log := p.Logger.With().
		Str("service", "background_email").
		Logger()

	return &backgroundEmailService{
		l:          &log,
		jobService: p.JobService,
	}
}

// QueueEmail queues a regular email to be sent in the background
func (s *backgroundEmailService) QueueEmail(
	ctx context.Context,
	req *services.SendEmailRequest,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	log := s.l.With().
		Str("operation", "queue_email").
		Str("org_id", req.OrganizationID.String()).
		Str("subject", req.Subject).
		Logger()

	// Validate request
	if err := req.Validate(); err != nil {
		log.Error().Err(err).Msg("invalid email request")
		return nil, oops.In("background_email_service").
			Tags("operation", "validate_request").
			Time(time.Now()).
			Wrapf(err, "invalid email request")
	}

	// Create job payload
	payload := &services.SendEmailPayload{
		JobBasePayload: services.JobBasePayload{
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
		},
		EmailType: "regular",
		Request:   req,
	}

	// Set default options if not provided
	if opts == nil {
		opts = &services.JobOptions{
			Queue:    services.QueueEmail,
			Priority: s.determinePriority(req.Priority),
			MaxRetry: 3,
		}
	}

	log.Info().
		Int("to_count", len(req.To)).
		Int("priority", opts.Priority).
		Msg("queueing email for background processing")

	// Schedule the job
	info, err := s.jobService.ScheduleSendEmail(payload, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to queue email")
		return nil, oops.In("background_email_service").
			Tags("operation", "queue_email").
			Tags("org_id", req.OrganizationID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to queue email")
	}

	log.Info().
		Str("job_id", info.ID).
		Str("queue", info.Queue).
		Time("process_at", info.NextProcessAt).
		Msg("email queued successfully")

	return info, nil
}

// QueueTemplatedEmail queues a templated email to be sent in the background
func (s *backgroundEmailService) QueueTemplatedEmail(
	ctx context.Context,
	req *services.SendTemplatedEmailRequest,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	log := s.l.With().
		Str("operation", "queue_templated_email").
		Str("org_id", req.OrganizationID.String()).
		Str("template_id", req.TemplateID.String()).
		Logger()

	// Validate request
	if err := req.Validate(); err != nil {
		log.Error().Err(err).Msg("invalid templated email request")
		return nil, oops.In("background_email_service").
			Tags("operation", "validate_templated_request").
			Time(time.Now()).
			Wrapf(err, "invalid templated email request")
	}

	// Create job payload
	payload := &services.SendEmailPayload{
		JobBasePayload: services.JobBasePayload{
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
		},
		EmailType:        "templated",
		TemplatedRequest: req,
	}

	// Set default options if not provided
	if opts == nil {
		opts = &services.JobOptions{
			Queue:    services.QueueEmail,
			Priority: s.determinePriority(req.Priority),
			MaxRetry: 3,
		}
	}

	log.Info().
		Int("to_count", len(req.To)).
		Int("priority", opts.Priority).
		Msg("queueing templated email for background processing")

	// Schedule the job
	info, err := s.jobService.ScheduleSendEmail(payload, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to queue templated email")
		return nil, oops.In("background_email_service").
			Tags("operation", "queue_templated_email").
			Tags("org_id", req.OrganizationID.String()).
			Tags("template_id", req.TemplateID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to queue templated email")
	}

	log.Info().
		Str("job_id", info.ID).
		Str("queue", info.Queue).
		Time("process_at", info.NextProcessAt).
		Msg("templated email queued successfully")

	return info, nil
}

// determinePriority maps email priority to job priority
func (s *backgroundEmailService) determinePriority(emailPriority email.Priority) int {
	switch emailPriority {
	case email.PriorityHigh:
		return services.PriorityHigh
	case email.PriorityLow:
		return services.PriorityLow
	default:
		return services.PriorityNormal
	}
}
