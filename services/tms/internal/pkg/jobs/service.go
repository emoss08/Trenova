/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
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
	RedisClient asynq.RedisClientOpt
	Handlers    []services.JobHandler `group:"job_handlers"`
}

// JobService manages background job scheduling and processing
type JobService struct {
	client     *asynq.Client
	server     *asynq.Server
	mux        *asynq.ServeMux
	logger     *zerolog.Logger
	inspector  *asynq.Inspector
	handlers   map[services.JobType]services.JobHandler
	isRunning  bool
	startTime  time.Time
	lastPanic  *time.Time
	panicCount int
}

// NewJobService creates a new job service instance
//
//nolint:gocritic // this is dependency injection
func NewJobService(p JobServiceParams) services.JobService {
	log := p.Logger.With().
		Str("service", "job").
		Logger()

	client := asynq.NewClient(p.RedisClient)
	inspector := asynq.NewInspector(p.RedisClient)

	server := asynq.NewServer(
		p.RedisClient,
		asynq.Config{
			Concurrency: 10, // Number of concurrent workers
			Queues: map[string]int{
				services.QueueCritical:   5, // Highest priority - 50% of workers
				services.QueueEmail:      2, // Email jobs - 20% of workers
				services.QueueShipment:   1, // Shipment jobs - 10% of workers
				services.QueuePattern:    1, // Pattern analysis - 10% of workers
				services.QueueCompliance: 1, // Compliance checks - 10% of workers
				services.QueueDefault:    1, // Default queue - 10% of workers
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

	mux := asynq.NewServeMux()

	js := &JobService{
		client:    client,
		server:    server,
		mux:       mux,
		logger:    &log,
		handlers:  make(map[services.JobType]services.JobHandler),
		inspector: inspector,
		isRunning: false,
	}

	for _, handler := range p.Handlers {
		js.RegisterHandler(handler)
	}

	return js
}

// Enqueue schedules a job for immediate processing
func (js *JobService) Enqueue(
	jobType services.JobType,
	payload any,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return js.enqueueJob(jobType, payload, opts, nil, nil)
}

// EnqueueIn schedules a job to be processed after a delay
func (js *JobService) EnqueueIn(
	jobType services.JobType,
	payload any,
	delay time.Duration,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	processAt := time.Now().Add(delay)
	return js.enqueueJob(jobType, payload, opts, &delay, &processAt)
}

// EnqueueAt schedules a job to be processed at a specific time
func (js *JobService) EnqueueAt(
	jobType services.JobType,
	payload any,
	processAt time.Time,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return js.enqueueJob(jobType, payload, opts, nil, &processAt)
}

// SchedulePatternAnalysis schedules a pattern analysis job
func (js *JobService) SchedulePatternAnalysis(
	payload *services.PatternAnalysisPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.PatternAnalysisOptions()
	}

	if opts.UniqueKey == "" {
		opts.UniqueKey = fmt.Sprintf("pattern_analysis_%s", payload.OrganizationID.String())
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeAnalyzePatterns, payload, opts)
}

// ScheduleDelayShipmentJobs schedules a delay shipment job
func (js *JobService) ScheduleDelayShipmentJobs(
	payload *services.DelayShipmentPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.DelayShipmentOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeDelayShipment, payload, opts)
}

// ScheduleExpireSuggestions schedules a job to expire old suggestions
func (js *JobService) ScheduleExpireSuggestions(
	payload *services.ExpireSuggestionsPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.DefaultJobOptions()
		opts.Queue = services.QueueSystem
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeExpireOldSuggestions, payload, opts)
}

// ScheduleComplianceCheck schedules a compliance check job
func (js *JobService) ScheduleComplianceCheck(
	payload *services.ComplianceCheckPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.DefaultJobOptions()
		opts.Queue = services.QueueCompliance
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeComplianceCheck, payload, opts)
}

// ScheduleShipmentStatusUpdate schedules a shipment status update job
func (js *JobService) ScheduleShipmentStatusUpdate(
	payload *services.ShipmentStatusUpdatePayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.CriticalJobOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeShipmentStatusUpdate, payload, opts)
}

// ScheduleSendEmail schedules an email send job
func (js *JobService) ScheduleSendEmail(
	payload *services.SendEmailPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = &services.JobOptions{
			Queue:    services.QueueEmail,
			Priority: services.PriorityNormal,
			MaxRetry: 3,
		}
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return js.Enqueue(services.JobTypeSendEmail, payload, opts)
}

// enqueueJob is the internal method that handles job scheduling
func (js *JobService) enqueueJob(
	jobType services.JobType,
	payload any,
	opts *services.JobOptions,
	delay *time.Duration,
	processAt *time.Time,
) (*asynq.TaskInfo, error) {
	if opts == nil {
		opts = services.DefaultJobOptions()
	}

	startTime := time.Now()

	js.logger.Info().
		Str("job_type", string(jobType)).
		Str("queue", opts.Queue).
		Int("priority", opts.Priority).
		Int("max_retry", opts.MaxRetry).
		Str("unique_key", opts.UniqueKey).
		Interface("payload_summary", js.extractPayloadSummary(payload)).
		Msg("scheduling job")

	data, err := services.MarshalPayload(payload)
	if err != nil {
		js.logger.Error().
			Err(err).
			Str("job_type", string(jobType)).
			Msg("failed to marshal job payload")
		return nil, fmt.Errorf("marshal job payload: %w", err)
	}

	task := asynq.NewTask(string(jobType), data, asynq.Retention(24*time.Hour))

	taskOpts := []asynq.Option{
		asynq.Queue(opts.Queue),
		asynq.MaxRetry(opts.MaxRetry),
	}

	if opts.UniqueKey != "" {
		taskOpts = append(taskOpts, asynq.Unique(time.Hour))
	}

	if opts.Deadline > 0 {
		deadline := time.Unix(opts.Deadline, 0)
		taskOpts = append(taskOpts, asynq.Deadline(deadline))
	}

	if processAt != nil {
		taskOpts = append(taskOpts, asynq.ProcessAt(*processAt))
	} else if delay != nil {
		taskOpts = append(taskOpts, asynq.ProcessIn(*delay))
	}

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
func (js *JobService) CancelJob(jobID string) error {
	err := js.inspector.CancelProcessing(jobID)
	if err != nil {
		return err
	}

	return nil
}

// GetJobInfo retrieves information about a job
func (js *JobService) GetJobInfo(
	queue string,
	jobID string,
) (*asynq.TaskInfo, error) {
	info, err := js.inspector.GetTaskInfo(queue, jobID)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// RegisterHandler registers a job handler with panic recovery
func (js *JobService) RegisterHandler(handler services.JobHandler) {
	jobType := handler.JobType()
	js.handlers[jobType] = handler

	wrappedHandler := js.wrapHandlerWithRecovery(handler)

	js.mux.HandleFunc(string(jobType), wrappedHandler)

	js.logger.Info().
		Str("job_type", string(jobType)).
		Msg("registered job handler")
}

// wrapHandlerWithRecovery wraps a job handler with panic recovery
func (js *JobService) wrapHandlerWithRecovery(
	handler services.JobHandler,
) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) (err error) {
		defer func() {
			if r := recover(); r != nil {
				js.logger.Error().
					Str("job_type", task.Type()).
					Interface("panic", r).
					Msg("job handler panicked")

				err = fmt.Errorf("job handler panicked: %v", r)
			}
		}()

		return handler.ProcessTask(ctx, task)
	}
}

// Start begins processing jobs with panic recovery
func (js *JobService) Start() error {
	js.logger.Info().
		Int("concurrency", 10).
		Interface("queue_distribution", map[string]int{
			services.QueueCritical:   5,
			services.QueueEmail:      2,
			services.QueueShipment:   1,
			services.QueuePattern:    1,
			services.QueueCompliance: 1,
			services.QueueDefault:    1,
		}).
		Int("registered_handlers", len(js.handlers)).
		Msg("starting job service")

	js.isRunning = true
	js.startTime = time.Now()

	go js.runWithRecovery()

	js.logger.Info().Msg("job service started successfully")
	return nil
}

// runWithRecovery runs the job server with panic recovery
func (js *JobService) runWithRecovery() {
	defer func() {
		if r := recover(); r != nil {
			now := time.Now()
			js.panicCount++
			js.lastPanic = &now

			js.logger.Error().
				Interface("panic", r).
				Int("total_panics", js.panicCount).
				Msg("job service panicked - attempting restart")

			time.Sleep(5 * time.Second)

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
	return js.isRunning && js.panicCount < 10
}

// GetStats returns comprehensive statistics about the job service
func (js *JobService) GetStats() services.JobServiceStats {
	uptime := ""
	if js.isRunning && !js.startTime.IsZero() {
		uptime = time.Since(js.startTime).String()
	}

	return services.JobServiceStats{
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
	case *services.PatternAnalysisPayload:
		summary["type"] = "pattern_analysis"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["trigger_reason"] = p.TriggerReason
		summary["min_frequency"] = p.MinFrequency

	case *services.ExpireSuggestionsPayload:
		summary["type"] = "expire_suggestions"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["batch_size"] = p.BatchSize

	case *services.ShipmentStatusUpdatePayload:
		summary["type"] = "shipment_status_update"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["shipment_id"] = p.ShipmentID.String()
		summary["status_change"] = p.OldStatus + " -> " + p.NewStatus
	case *services.DelayShipmentPayload:
		summary["type"] = "delay_shipment"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
	case *services.ComplianceCheckPayload:
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
	case *services.DuplicateShipmentPayload:
		summary["type"] = "duplicate_shipment"
		summary["organization_id"] = p.OrganizationID.String()
		summary["business_unit_id"] = p.BusinessUnitID.String()
		summary["count"] = p.Count

	case *services.SendEmailPayload:
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
