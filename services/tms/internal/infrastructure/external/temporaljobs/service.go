/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

// ServiceParams defines dependencies for the workflow service
type ServiceParams struct {
	fx.In

	Logger *logger.Logger
	Client client.Client
	Worker *Worker
}

// Service manages workflow scheduling and execution
type Service struct {
	client           client.Client
	worker           *Worker
	logger           *zerolog.Logger
	workflowsStarted atomic.Int64
	workflowsFailed  atomic.Int64
}

// NewService creates a new workflow service
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "temporal").
		Logger()

	return &Service{
		client: p.Client,
		worker: p.Worker,
		logger: &log,
	}
}

// ExecuteWorkflow schedules a workflow for immediate execution
func (s *Service) ExecuteWorkflow(
	ctx context.Context,
	workflowName string,
	payload any,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = DefaultWorkflowOptions()
	}

	if opts.WorkflowID == "" {
		opts.WorkflowID = pulid.MustNew("wf_").String()
	}

	startTime := time.Now()

	s.logger.Info().
		Str("workflow_name", workflowName).
		Str("workflow_id", opts.WorkflowID).
		Str("task_queue", opts.TaskQueue).
		Int("priority", int(opts.Priority)).
		Interface("payload_summary", s.extractPayloadSummary(payload)).
		Msg("scheduling workflow")

	workflowOptions := client.StartWorkflowOptions{
		ID:                       opts.WorkflowID,
		TaskQueue:                opts.TaskQueue,
		WorkflowExecutionTimeout: opts.ExecutionTimeout,
		WorkflowRunTimeout:       opts.RunTimeout,
		WorkflowTaskTimeout:      opts.TaskTimeout,
		WorkflowIDReusePolicy:    opts.WorkflowIDReusePolicy,
		RetryPolicy:              opts.RetryPolicy,
		SearchAttributes:         opts.SearchAttributes,
		Memo:                     opts.Memo,
	}

	run, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflowName, payload)
	if err != nil {
		s.workflowsFailed.Add(1)
		s.logger.Error().
			Err(err).
			Str("workflow_name", workflowName).
			Str("workflow_id", opts.WorkflowID).
			Dur("elapsed", time.Since(startTime)).
			Msg("failed to execute workflow")
		return nil, fmt.Errorf("execute workflow: %w", err)
	}

	s.workflowsStarted.Add(1)
	s.logger.Info().
		Str("workflow_name", workflowName).
		Str("workflow_id", opts.WorkflowID).
		Str("run_id", run.GetRunID()).
		Dur("scheduling_time", time.Since(startTime)).
		Msg("workflow executed successfully")

	return run, nil
}

// ExecuteWorkflowWithDelay schedules a workflow to execute after a delay
func (s *Service) ExecuteWorkflowWithDelay(
	ctx context.Context,
	workflowName string,
	payload any,
	delay time.Duration,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = DefaultWorkflowOptions()
	}

	s.logger.Info().
		Str("workflow_name", workflowName).
		Dur("delay", delay).
		Msg("scheduling delayed workflow")

	// For delayed execution, we can use Temporal's schedule feature
	// or implement a delay workflow that waits before executing the actual logic
	// For now, we'll execute immediately with a note that the workflow should handle the delay

	// Add delay information to the payload if it's a BasePayload
	if basePayload, ok := payload.(*BasePayload); ok {
		if basePayload.Metadata == nil {
			basePayload.Metadata = make(map[string]any)
		}
		basePayload.Metadata["delay_seconds"] = delay.Seconds()
	}

	return s.ExecuteWorkflow(ctx, workflowName, payload, opts)
}

// SchedulePatternAnalysis schedules a pattern analysis workflow
func (s *Service) SchedulePatternAnalysis(
	ctx context.Context,
	payload *PatternAnalysisPayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = PatternAnalysisOptions()
	}

	if opts.WorkflowID == "" {
		opts.WorkflowID = fmt.Sprintf("pattern_analysis_%s", payload.OrganizationID.String())
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowAnalyzePatterns, payload, opts)
}

// ScheduleDelayShipment schedules a delay shipment workflow
func (s *Service) ScheduleDelayShipment(
	ctx context.Context,
	payload *DelayShipmentPayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = ShipmentWorkflowOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowDelayShipment, payload, opts)
}

// ScheduleExpireSuggestions schedules a workflow to expire old suggestions
func (s *Service) ScheduleExpireSuggestions(
	ctx context.Context,
	payload *ExpireSuggestionsPayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = DefaultWorkflowOptions()
		opts.TaskQueue = TaskQueueDefault
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowExpireOldSuggestions, payload, opts)
}

// ScheduleComplianceCheck schedules a compliance check workflow
func (s *Service) ScheduleComplianceCheck(
	ctx context.Context,
	payload *ComplianceCheckPayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = DefaultWorkflowOptions()
		opts.TaskQueue = TaskQueueCompliance
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowComplianceCheck, payload, opts)
}

// ScheduleShipmentStatusUpdate schedules a shipment status update workflow
func (s *Service) ScheduleShipmentStatusUpdate(
	ctx context.Context,
	payload *ShipmentStatusUpdatePayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = CriticalWorkflowOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowShipmentStatusUpdate, payload, opts)
}

// ScheduleSendEmail schedules an email send workflow
func (s *Service) ScheduleSendEmail(
	ctx context.Context,
	payload *EmailPayload,
	opts *WorkflowOptions,
) (client.WorkflowRun, error) {
	if opts == nil {
		opts = EmailWorkflowOptions()
	}

	payload.JobID = pulid.MustNew("job_").String()
	payload.Timestamp = timeutils.NowUnix()

	return s.ExecuteWorkflow(ctx, WorkflowSendEmail, payload, opts)
}

// CancelWorkflow cancels a running workflow
func (s *Service) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	err := s.client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("workflow_id", workflowID).
			Str("run_id", runID).
			Msg("failed to cancel workflow")
		return fmt.Errorf("cancel workflow: %w", err)
	}

	s.logger.Info().
		Str("workflow_id", workflowID).
		Str("run_id", runID).
		Msg("workflow cancelled successfully")

	return nil
}

// GetWorkflowInfo retrieves information about a workflow
func (s *Service) GetWorkflowInfo(
	ctx context.Context,
	workflowID string,
	runID string,
) (client.WorkflowRun, error) {
	return s.client.GetWorkflow(ctx, workflowID, runID), nil
}

// GetStats returns comprehensive statistics about the workflow service
func (s *Service) GetStats() WorkflowStats {
	workerStats := s.worker.GetStats()
	workerStats.WorkflowsStarted = s.workflowsStarted.Load()
	workerStats.WorkflowsFailed = s.workflowsFailed.Load()
	return workerStats
}

// extractPayloadSummary creates a concise summary of the payload for logging
func (s *Service) extractPayloadSummary(payload any) map[string]any {
	summary := make(map[string]any)

	// Try to marshal and unmarshal to get a generic view
	if data, err := sonic.Marshal(payload); err == nil {
		var generic map[string]any
		if err := sonic.Unmarshal(data, &generic); err == nil {
			// Extract key fields if they exist
			if orgID, ok := generic["organizationId"]; ok {
				summary["organization_id"] = orgID
			}
			if buID, ok := generic["businessUnitId"]; ok {
				summary["business_unit_id"] = buID
			}
			if triggerReason, ok := generic["triggerReason"]; ok {
				summary["trigger_reason"] = triggerReason
			}
			if checkType, ok := generic["checkType"]; ok {
				summary["check_type"] = checkType
			}
		}
	}

	// Add payload type
	summary["payload_type"] = fmt.Sprintf("%T", payload)

	return summary
}
