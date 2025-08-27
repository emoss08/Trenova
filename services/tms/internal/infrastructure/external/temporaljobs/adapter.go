/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/stringutils"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

// AdapterParams defines dependencies for the adapter
type AdapterParams struct {
	fx.In

	Logger         *logger.Logger
	TemporalClient client.Client
	AsynqService   services.JobService `name:"asynq_service"` // The original Asynq service
}

// TemporalJobServiceAdapter adapts between the JobService interface and Temporal
// It routes specific jobs to Temporal while maintaining backward compatibility
type TemporalJobServiceAdapter struct {
	temporalClient client.Client
	asynqService   services.JobService
	logger         *zerolog.Logger
}

// NewTemporalJobServiceAdapter creates a new adapter
func NewTemporalJobServiceAdapter(p AdapterParams) services.JobService {
	log := p.Logger.With().
		Str("component", "temporal-adapter").
		Logger()

	return &TemporalJobServiceAdapter{
		temporalClient: p.TemporalClient,
		asynqService:   p.AsynqService,
		logger:         &log,
	}
}

// Enqueue schedules a job for immediate processing
func (a *TemporalJobServiceAdapter) Enqueue(
	jobType services.JobType,
	payload any,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	// Route duplicate shipment jobs to Temporal
	if jobType == services.JobTypeDuplicateShipment {
		return a.enqueueTemporal(jobType, payload, opts)
	}

	// All other jobs go to Asynq
	return a.asynqService.Enqueue(jobType, payload, opts)
}

// EnqueueIn schedules a job to be processed after a delay
func (a *TemporalJobServiceAdapter) EnqueueIn(
	jobType services.JobType,
	payload any,
	delay time.Duration,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	// Route duplicate shipment jobs to Temporal
	if jobType == services.JobTypeDuplicateShipment {
		return a.enqueueTemporalIn(jobType, payload, delay, opts)
	}

	// All other jobs go to Asynq
	return a.asynqService.EnqueueIn(jobType, payload, delay, opts)
}

// EnqueueAt schedules a job to be processed at a specific time
func (a *TemporalJobServiceAdapter) EnqueueAt(
	jobType services.JobType,
	payload any,
	processAt time.Time,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	// Route duplicate shipment jobs to Temporal
	if jobType == services.JobTypeDuplicateShipment {
		delay := time.Until(processAt)
		return a.enqueueTemporalIn(jobType, payload, delay, opts)
	}

	// All other jobs go to Asynq
	return a.asynqService.EnqueueAt(jobType, payload, processAt, opts)
}

// enqueueTemporal handles Temporal workflow execution
func (a *TemporalJobServiceAdapter) enqueueTemporal(
	jobType services.JobType,
	payload any,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	ctx := context.Background()

	// Convert payload to Temporal format
	duplicatePayload, err := a.convertToTemporalPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload: %w", err)
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("duplicate-shipment-%s-%s", 
		duplicatePayload.ShipmentID.String(),
		stringutils.GenerateRandomString(8),
	)

	// Configure workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueueShipment,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowExecutionTimeout: 30 * time.Minute,
		WorkflowRunTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}

	// TODO: Add search attributes once they are configured in Temporal server
	// Search attributes need to be registered in Temporal before use:
	// temporal operator search-attribute create --name OrganizationId --type Keyword
	// temporal operator search-attribute create --name UserId --type Keyword
	// 
	// if duplicatePayload.OrganizationID.String() != "" {
	// 	workflowOptions.SearchAttributes = map[string]any{
	// 		"OrganizationId": duplicatePayload.OrganizationID.String(),
	// 		"UserId":         duplicatePayload.UserID.String(),
	// 	}
	// }

	// Start the workflow
	run, err := a.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "DuplicateShipmentWorkflow", duplicatePayload)
	if err != nil {
		a.logger.Error().Err(err).
			Str("workflow_id", workflowID).
			Msg("failed to start Temporal workflow")
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	a.logger.Info().
		Str("workflow_id", run.GetID()).
		Str("run_id", run.GetRunID()).
		Str("job_type", string(jobType)).
		Msg("Temporal workflow started")

	// Create a mock TaskInfo for compatibility
	// This allows the existing code to continue working
	taskInfo := &asynq.TaskInfo{
		ID:        run.GetID(),
		Queue:     TaskQueueShipment,
		Type:      string(jobType),
		State:     asynq.TaskStatePending,
		NextProcessAt: time.Now(),
	}

	return taskInfo, nil
}

// enqueueTemporalIn handles delayed Temporal workflow execution
func (a *TemporalJobServiceAdapter) enqueueTemporalIn(
	jobType services.JobType,
	payload any,
	delay time.Duration,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	ctx := context.Background()

	// Convert payload to Temporal format
	duplicatePayload, err := a.convertToTemporalPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload: %w", err)
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("duplicate-shipment-%s-%s-delayed", 
		duplicatePayload.ShipmentID.String(),
		stringutils.GenerateRandomString(8),
	)

	// Configure workflow options with delay
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueueShipment,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowExecutionTimeout: 30 * time.Minute,
		WorkflowRunTimeout: 10 * time.Minute,
		StartDelay: delay, // This is the key difference for delayed execution
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}

	// Start the workflow
	run, err := a.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "DuplicateShipmentWorkflow", duplicatePayload)
	if err != nil {
		a.logger.Error().Err(err).
			Str("workflow_id", workflowID).
			Dur("delay", delay).
			Msg("failed to start delayed Temporal workflow")
		return nil, fmt.Errorf("failed to start delayed workflow: %w", err)
	}

	a.logger.Info().
		Str("workflow_id", run.GetID()).
		Str("run_id", run.GetRunID()).
		Dur("delay", delay).
		Msg("Delayed Temporal workflow scheduled")

	// Create a mock TaskInfo for compatibility
	taskInfo := &asynq.TaskInfo{
		ID:        run.GetID(),
		Queue:     TaskQueueShipment,
		Type:      string(jobType),
		State:     asynq.TaskStateScheduled,
		NextProcessAt: time.Now().Add(delay),
	}

	return taskInfo, nil
}

// convertToTemporalPayload converts the generic payload to Temporal-specific format
func (a *TemporalJobServiceAdapter) convertToTemporalPayload(payload any) (*payloads.DuplicateShipmentPayload, error) {
	// First serialize to JSON to handle the conversion
	data, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Try to deserialize as the old format first
	var oldPayload services.DuplicateShipmentPayload
	if err := sonic.Unmarshal(data, &oldPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Convert to new Temporal format
	temporalPayload := &payloads.DuplicateShipmentPayload{
		BasePayload: payloads.BasePayload{
			JobID:          oldPayload.JobID,
			OrganizationID: oldPayload.OrganizationID,
			BusinessUnitID: oldPayload.BusinessUnitID,
			UserID:         oldPayload.UserID,
			Timestamp:      oldPayload.Timestamp,
			Metadata:       oldPayload.Metadata,
		},
		ShipmentID:               oldPayload.ShipmentID,
		Count:                    oldPayload.Count,
		OverrideDates:            oldPayload.OverrideDates,
		IncludeCommodities:       oldPayload.IncludeCommodities,
		IncludeAdditionalCharges: oldPayload.IncludeAdditionalCharges,
	}

	return temporalPayload, nil
}

// RegisterHandler registers a job handler (no-op for Temporal jobs)
func (a *TemporalJobServiceAdapter) RegisterHandler(handler services.JobHandler) {
	// Only register non-Temporal handlers with Asynq
	if handler.JobType() != services.JobTypeDuplicateShipment {
		a.asynqService.RegisterHandler(handler)
	}
}

// Start starts the job processing server
func (a *TemporalJobServiceAdapter) Start() error {
	a.logger.Info().Msg("starting job service adapter")
	
	// Only start the Asynq service, Temporal worker is started separately
	return a.asynqService.Start()
}

// Shutdown stops the job processing server
func (a *TemporalJobServiceAdapter) Shutdown() error {
	a.logger.Info().Msg("stopping job service adapter")
	
	// Only stop the Asynq service, Temporal worker is stopped separately
	return a.asynqService.Shutdown()
}

// GetStats returns job statistics
func (a *TemporalJobServiceAdapter) GetStats() services.JobServiceStats {
	// For now, return Asynq statistics
	// In the future, we can combine Asynq and Temporal statistics
	return a.asynqService.GetStats()
}

// IsHealthy checks if the service is healthy
func (a *TemporalJobServiceAdapter) IsHealthy() bool {
	return a.asynqService.IsHealthy()
}

// CancelJob cancels a scheduled job
func (a *TemporalJobServiceAdapter) CancelJob(jobID string) error {
	// Try to cancel in Temporal first if it looks like a Temporal job ID
	if a.isTemporalJobID(jobID) {
		ctx := context.Background()
		err := a.temporalClient.CancelWorkflow(ctx, jobID, "")
		if err != nil {
			a.logger.Warn().Err(err).
				Str("job_id", jobID).
				Msg("failed to cancel Temporal workflow, trying Asynq")
		} else {
			return nil
		}
	}

	// Fall back to Asynq
	return a.asynqService.CancelJob(jobID)
}

// GetJobInfo retrieves information about a specific job
func (a *TemporalJobServiceAdapter) GetJobInfo(queue string, jobID string) (*asynq.TaskInfo, error) {
	// For now, only check Asynq
	// In the future, we can add Temporal job info retrieval
	return a.asynqService.GetJobInfo(queue, jobID)
}

// SchedulePatternAnalysis schedules a pattern analysis job
func (a *TemporalJobServiceAdapter) SchedulePatternAnalysis(
	payload *services.PatternAnalysisPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.SchedulePatternAnalysis(payload, opts)
}

// ScheduleDelayShipmentJobs schedules delay shipment jobs
func (a *TemporalJobServiceAdapter) ScheduleDelayShipmentJobs(
	payload *services.DelayShipmentPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.ScheduleDelayShipmentJobs(payload, opts)
}

// ScheduleExpireSuggestions schedules expire suggestions job
func (a *TemporalJobServiceAdapter) ScheduleExpireSuggestions(
	payload *services.ExpireSuggestionsPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.ScheduleExpireSuggestions(payload, opts)
}

// ScheduleComplianceCheck schedules compliance check job
func (a *TemporalJobServiceAdapter) ScheduleComplianceCheck(
	payload *services.ComplianceCheckPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.ScheduleComplianceCheck(payload, opts)
}

// ScheduleShipmentStatusUpdate schedules shipment status update job
func (a *TemporalJobServiceAdapter) ScheduleShipmentStatusUpdate(
	payload *services.ShipmentStatusUpdatePayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.ScheduleShipmentStatusUpdate(payload, opts)
}

// ScheduleSendEmail schedules send email job
func (a *TemporalJobServiceAdapter) ScheduleSendEmail(
	payload *services.SendEmailPayload,
	opts *services.JobOptions,
) (*asynq.TaskInfo, error) {
	return a.asynqService.ScheduleSendEmail(payload, opts)
}

// isTemporalJobID checks if a job ID looks like a Temporal workflow ID
func (a *TemporalJobServiceAdapter) isTemporalJobID(jobID string) bool {
	// Temporal workflow IDs contain "duplicate-shipment-" prefix
	return len(jobID) > 0 && 
		(contains(jobID, "duplicate-shipment-") || 
		 contains(jobID, "workflow-"))
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}