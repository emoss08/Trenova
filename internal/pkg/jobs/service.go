/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	unknownJobID = "unknown"
)

// JobServiceParams defines dependencies for the job service
type JobServiceParams struct {
	fx.In

	Logger      *logger.Logger
	RedisClient asynq.RedisClientOpt // We'll configure this in the infrastructure module
	Handlers    []JobHandler         `group:"job_handlers"`
}

// JobService manages background job scheduling and processing
type JobService struct {
	client     *asynq.Client
	server     *asynq.Server
	mux        *asynq.ServeMux
	logger     *zerolog.Logger
	handlers   map[JobType]JobHandler
	isRunning  bool
	startTime  time.Time
	lastPanic  *time.Time
	panicCount int
}

// JobServiceInterface defines the contract for the job service
type JobServiceInterface interface {
	// Job Scheduling
	Enqueue(
		jobType JobType,
		payload any,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	EnqueueIn(
		jobType JobType,
		payload any,
		delay time.Duration,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	EnqueueAt(
		jobType JobType,
		payload any,
		processAt time.Time,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Dedicated Lane Pattern Analysis Jobs
	SchedulePatternAnalysis(
		payload *PatternAnalysisPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleDelayShipmentJobs(
		payload *DelayShipmentPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleExpireSuggestions(
		payload *ExpireSuggestionsPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// System Jobs
	ScheduleComplianceCheck(
		payload *ComplianceCheckPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleShipmentStatusUpdate(
		payload *ShipmentStatusUpdatePayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Email Jobs
	ScheduleSendEmail(
		payload *SendEmailPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Job Management
	CancelJob(ctx context.Context, queue string, jobID string) error
	GetJobInfo(ctx context.Context, queue string, jobID string) (*asynq.TaskInfo, error)

	// Worker Management
	Start() error
	Shutdown() error
	RegisterHandler(handler JobHandler)

	// Health Monitoring
	IsHealthy() bool
	GetStats() JobServiceStats
}

// NewJobService creates a new job service instance
//
//nolint:gocritic // this is dependency injection
func NewJobService(p JobServiceParams) JobServiceInterface {
	log := p.Logger.With().
		Str("service", "job").
		Logger()

	// Create Asynq client for job scheduling
	client := asynq.NewClient(p.RedisClient)

	// Create Asynq server for job processing
	server := asynq.NewServer(
		p.RedisClient,
		asynq.Config{
			Concurrency: 10, // Number of concurrent workers
			Queues: map[string]int{
				QueueCritical:   5, // Highest priority - 50% of workers
				QueueEmail:      2, // Email jobs - 20% of workers
				QueueShipment:   1, // Shipment jobs - 10% of workers
				QueuePattern:    1, // Pattern analysis - 10% of workers
				QueueCompliance: 1, // Compliance checks - 10% of workers
				QueueDefault:    1, // Default queue - 10% of workers
			},
			RetryDelayFunc: func(n int, _ error, task *asynq.Task) time.Duration {
				// Exponential backoff with jitter: 1s, 2s, 4s, 8s, 16s
				delay := time.Duration(1<<n) * time.Second
				log.Warn().
					Str("job_type", task.Type()).
					Str("job_id", func() string {
						if rw := task.ResultWriter(); rw != nil {
							return rw.TaskID()
						}
						return unknownJobID
					}()).
					Int("retry_attempt", n+1).
					Dur("retry_delay", delay).
					Msg("job retry scheduled")
				return delay
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(_ context.Context, task *asynq.Task, err error) {
					// Extract payload information for better error context
					var payloadInfo map[string]any
					if unmarshalErr := sonic.Unmarshal(task.Payload(), &payloadInfo); unmarshalErr == nil {
						log.Error().
							Err(err).
							Str("job_type", task.Type()).
							Str("job_id", func() string {
								if rw := task.ResultWriter(); rw != nil {
									return rw.TaskID()
								}
								return unknownJobID
							}()).
							Interface("payload", payloadInfo).
							Msg("job processing failed permanently")
					} else {
						log.Error().
							Err(err).
							Str("job_type", task.Type()).
							Str("job_id", func() string {
								if rw := task.ResultWriter(); rw != nil {
									return rw.TaskID()
								}
								return unknownJobID
							}()).
							Msg("job processing failed permanently")
					}
				},
			),
		},
	)

	// Create multiplexer for routing jobs to handlers
	mux := asynq.NewServeMux()

	js := &JobService{
		client:    client,
		server:    server,
		mux:       mux,
		logger:    &log,
		handlers:  make(map[JobType]JobHandler),
		isRunning: false,
	}

	// Register all provided handlers
	for _, handler := range p.Handlers {
		js.RegisterHandler(handler)
	}

	return js
}

// Enqueue schedules a job for immediate processing
func (js *JobService) Enqueue(
	jobType JobType,
	payload any,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	return js.enqueueJob(jobType, payload, opts, nil, nil)
}

// EnqueueIn schedules a job to be processed after a delay
func (js *JobService) EnqueueIn(
	jobType JobType,
	payload any,
	delay time.Duration,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	processAt := time.Now().Add(delay)
	return js.enqueueJob(jobType, payload, opts, &delay, &processAt)
}

// EnqueueAt schedules a job to be processed at a specific time
func (js *JobService) EnqueueAt(
	jobType JobType,
	payload any,
	processAt time.Time,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	return js.enqueueJob(jobType, payload, opts, nil, &processAt)
}

// SchedulePatternAnalysis schedules a pattern analysis job
func (js *JobService) SchedulePatternAnalysis(
	payload *PatternAnalysisPayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = PatternAnalysisOptions()
	}

	// Set unique key to prevent duplicate analysis for same org/timeframe
	if opts.UniqueKey == "" {
		opts.UniqueKey = fmt.Sprintf("pattern_analysis_%s", payload.OrganizationID.String())
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeAnalyzePatterns, payload, opts)
}

// ScheduleDelayShipmentJobs schedules a delay shipment job
func (js *JobService) ScheduleDelayShipmentJobs(
	payload *DelayShipmentPayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = DelayShipmentOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeDelayShipment, payload, opts)
}

// ScheduleExpireSuggestions schedules a job to expire old suggestions
func (js *JobService) ScheduleExpireSuggestions(
	payload *ExpireSuggestionsPayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = DefaultJobOptions()
		opts.Queue = QueueSystem
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeExpireOldSuggestions, payload, opts)
}

// ScheduleComplianceCheck schedules a compliance check job
func (js *JobService) ScheduleComplianceCheck(
	payload *ComplianceCheckPayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = DefaultJobOptions()
		opts.Queue = QueueCompliance
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeComplianceCheck, payload, opts)
}

// ScheduleShipmentStatusUpdate schedules a shipment status update job
func (js *JobService) ScheduleShipmentStatusUpdate(
	payload *ShipmentStatusUpdatePayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = CriticalJobOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeShipmentStatusUpdate, payload, opts)
}

// ScheduleSendEmail schedules an email send job
func (js *JobService) ScheduleSendEmail(
	payload *SendEmailPayload,
	opts *JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = &JobOptions{
			Queue:    QueueEmail,
			Priority: PriorityNormal,
			MaxRetry: 3,
		}
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(JobTypeSendEmail, payload, opts)
}

// enqueueJob is the internal method that handles job scheduling
func (js *JobService) enqueueJob(
	jobType JobType,
	payload any,
	opts *JobOptions,
	delay *time.Duration,
	processAt *time.Time,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = DefaultJobOptions()
	}

	// Log job scheduling attempt
	startTime := time.Now()

	js.logger.Info().
		Str("job_type", string(jobType)).
		Str("queue", opts.Queue).
		Int("priority", opts.Priority).
		Int("max_retry", opts.MaxRetry).
		Str("unique_key", opts.UniqueKey).
		Interface("payload_summary", js.extractPayloadSummary(payload)).
		Msg("scheduling job")

	// Marshal payload
	data, err := MarshalPayload(payload)
	if err != nil {
		js.logger.Error().
			Err(err).
			Str("job_type", string(jobType)).
			Msg("failed to marshal job payload")
		return nil, fmt.Errorf("marshal job payload: %w", err)
	}

	// Create Asynq task
	task := asynq.NewTask(string(jobType), data, asynq.Retention(24*time.Hour))

	// Build task options
	taskOpts := []asynq.Option{
		asynq.Queue(opts.Queue),
		asynq.MaxRetry(opts.MaxRetry),
	}

	// Priority is set via the priority field in the task options

	if opts.UniqueKey != "" {
		taskOpts = append(taskOpts, asynq.Unique(time.Hour)) // Unique for 1 hour
	}

	if opts.Deadline > 0 {
		deadline := time.Unix(opts.Deadline, 0)
		taskOpts = append(taskOpts, asynq.Deadline(deadline))
	}

	// Add process time options if specified
	if processAt != nil {
		taskOpts = append(taskOpts, asynq.ProcessAt(*processAt))
	} else if delay != nil {
		taskOpts = append(taskOpts, asynq.ProcessIn(*delay))
	}

	// Schedule job
	info, err := js.client.Enqueue(task, taskOpts...)

	if err != nil {
		js.logger.Error().
			Err(err).
			Str("job_type", string(jobType)).
			Str("queue", opts.Queue).
			Dur("elapsed", time.Since(startTime)).
			Msg("failed to enqueue job")
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	// Success logging with timing and context
	js.logger.Info().
		Str("job_type", string(jobType)).
		Str("job_id", info.ID).
		Str("queue", info.Queue).
		Time("scheduled_at", info.NextProcessAt).
		Dur("scheduling_time", time.Since(startTime)).
		Msg("job enqueued successfully")

	return info, nil
}

// CancelJob cancels a scheduled job
func (js *JobService) CancelJob(_ context.Context, _ string, _ string) error {
	// Note: Asynq doesn't support direct job cancellation by ID
	// This would need to be implemented using job inspection and manual cancellation
	return errors.New("job cancellation not implemented - use Asynq web UI or inspector")
}

// GetJobInfo retrieves information about a job
func (js *JobService) GetJobInfo(
	_ context.Context,
	_ string,
	_ string,
) (*asynq.TaskInfo, error) {
	// Create inspector with the same Redis config as the client
	// Note: This requires access to the underlying Redis config
	return nil, errors.New("job info retrieval not implemented - use Asynq web UI or inspector")
}

// RegisterHandler registers a job handler with panic recovery
func (js *JobService) RegisterHandler(handler JobHandler) {
	jobType := handler.JobType()
	js.handlers[jobType] = handler

	// Wrap handler with panic recovery
	wrappedHandler := js.wrapHandlerWithRecovery(handler)

	// Register with Asynq mux
	js.mux.HandleFunc(string(jobType), wrappedHandler)

	js.logger.Info().
		Str("job_type", string(jobType)).
		Msg("registered job handler")
}

// wrapHandlerWithRecovery wraps a job handler with panic recovery
func (js *JobService) wrapHandlerWithRecovery(
	handler JobHandler,
) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) (err error) {
		defer func() {
			if r := recover(); r != nil {
				js.logger.Error().
					Str("job_type", task.Type()).
					Interface("panic", r).
					Msg("job handler panicked")

				// Convert panic to error
				err = fmt.Errorf("job handler panicked: %v", r)
			}
		}()

		// Execute the actual handler
		return handler.ProcessTask(ctx, task)
	}
}

// Start begins processing jobs with panic recovery
func (js *JobService) Start() error {
	js.logger.Info().
		Int("concurrency", 10).
		Interface("queue_distribution", map[string]int{
			QueueCritical:   5,
			QueueEmail:      2,
			QueueShipment:   1,
			QueuePattern:    1,
			QueueCompliance: 1,
			QueueDefault:    1,
		}).
		Int("registered_handlers", len(js.handlers)).
		Msg("starting job service")

	// Mark as running and set start time
	js.isRunning = true
	js.startTime = time.Now()

	// Start the server in a goroutine with panic recovery
	go js.runWithRecovery()

	js.logger.Info().Msg("job service started successfully")
	return nil
}

// runWithRecovery runs the job server with panic recovery
func (js *JobService) runWithRecovery() {
	defer func() {
		if r := recover(); r != nil {
			// Track panic stats
			now := time.Now()
			js.panicCount++
			js.lastPanic = &now

			js.logger.Error().
				Interface("panic", r).
				Int("total_panics", js.panicCount).
				Msg("job service panicked - attempting restart")

			// Wait a bit before restarting to avoid rapid restart loops
			time.Sleep(5 * time.Second)

			// Attempt to restart the service
			go js.runWithRecovery()
		}
	}()

	js.logger.Info().Msg("job worker server started")
	if err := js.server.Run(js.mux); err != nil {
		js.logger.Error().Err(err).Msg("job server stopped unexpectedly")
	}
}

// Shutdown gracefully stops the job service
func (js *JobService) Shutdown() error {
	js.logger.Info().Msg("shutting down job service")

	js.isRunning = false
	js.server.Shutdown()

	if err := js.client.Close(); err != nil {
		js.logger.Warn().Err(err).Msg("error closing job client")
	}

	js.logger.Info().Msg("job service shutdown completed")
	return nil
}

// IsHealthy returns true if the job service is running and healthy
func (js *JobService) IsHealthy() bool {
	return js.isRunning && js.panicCount < 10 // Consider unhealthy after 10 panics
}

// GetStats returns comprehensive statistics about the job service
func (js *JobService) GetStats() JobServiceStats {
	uptime := ""
	if js.isRunning && !js.startTime.IsZero() {
		uptime = time.Since(js.startTime).String()
	}

	return JobServiceStats{
		IsRunning:    js.isRunning,
		StartTime:    js.startTime,
		Uptime:       uptime,
		PanicCount:   js.panicCount,
		LastPanic:    js.lastPanic,
		HandlerCount: len(js.handlers),
	}
}

// extractPayloadSummary creates a concise summary of the payload for logging
func (js *JobService) extractPayloadSummary(payload any) map[string]any {
	summary := make(map[string]any)

	switch p := payload.(type) {
	case *PatternAnalysisPayload:
		summary["type"] = "pattern_analysis"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["trigger_reason"] = p.TriggerReason
		summary["min_frequency"] = p.MinFrequency

	case *ExpireSuggestionsPayload:
		summary["type"] = "expire_suggestions"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["batch_size"] = p.BatchSize

	case *ShipmentStatusUpdatePayload:
		summary["type"] = "shipment_status_update"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["shipment_id"] = p.ShipmentID.String()
		summary["status_change"] = p.OldStatus + " -> " + p.NewStatus
	case *DelayShipmentPayload:
		summary["type"] = "delay_shipment"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
	case *ComplianceCheckPayload:
		summary["type"] = "compliance_check"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["check_type"] = p.CheckType
		if p.WorkerID != nil {
			summary["worker_id"] = p.WorkerID.String()
		}
		if p.ShipmentID != nil {
			summary["shipment_id"] = p.ShipmentID.String()
		}
	case *DuplicateShipmentPayload:
		summary["type"] = "duplicate_shipment"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["count"] = p.Count

	case *SendEmailPayload:
		summary["type"] = "send_email"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["email_type"] = p.EmailType
		if p.Request != nil {
			summary["subject"] = p.Request.Subject
			summary["to_count"] = len(p.Request.To)
		}
		if p.TemplatedRequest != nil {
			summary["template_id"] = p.TemplatedRequest.TemplateID.String()
			summary["to_count"] = len(p.TemplatedRequest.To)
		}

	default:
		summary["type"] = "unknown"
		summary["payload_type"] = fmt.Sprintf("%T", payload)
	}

	return summary
}
