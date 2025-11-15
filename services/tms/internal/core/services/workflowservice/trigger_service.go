package workflowservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/jobscheduler"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TriggerService handles workflow trigger management
type TriggerService struct {
	logger           *zap.Logger
	workflowRepo     repositories.WorkflowRepository
	executionService *ExecutionService
	schedulerManager *jobscheduler.Manager
	temporalClient   client.Client
	auditService     services.AuditService
}

// TriggerServiceParams holds dependencies for trigger service
type TriggerServiceParams struct {
	fx.In

	Logger           *zap.Logger
	WorkflowRepo     repositories.WorkflowRepository
	ExecutionService *ExecutionService
	SchedulerManager *jobscheduler.Manager
	TemporalClient   client.Client
	AuditService     services.AuditService
}

// NewTriggerService creates a new trigger service
func NewTriggerService(p TriggerServiceParams) *TriggerService {
	return &TriggerService{
		logger:           p.Logger.Named("workflow-trigger-service"),
		workflowRepo:     p.WorkflowRepo,
		executionService: p.ExecutionService,
		schedulerManager: p.SchedulerManager,
		temporalClient:   p.TemporalClient,
		auditService:     p.AuditService,
	}
}

// SetupScheduledTrigger creates or updates a Temporal schedule for a workflow
func (s *TriggerService) SetupScheduledTrigger(
	ctx context.Context,
	wf *workflow.Workflow,
	triggerConfig *workflow.ScheduledTriggerConfig,
) error {
	if wf.TriggerType != workflow.TriggerTypeScheduled {
		return fmt.Errorf("workflow trigger type is not scheduled")
	}

	if wf.Status != workflow.WorkflowStatusActive {
		return fmt.Errorf("workflow must be active to setup scheduled trigger")
	}

	if wf.PublishedVersionID == nil {
		return fmt.Errorf("workflow must have a published version")
	}

	scheduleID := fmt.Sprintf("workflow-%s", wf.ID.String())

	// Parse cron expression
	if triggerConfig.CronExpression == "" {
		return fmt.Errorf("cron expression is required for scheduled triggers")
	}

	scheduleConfig := &jobscheduler.ScheduleConfig{
		ID:           scheduleID,
		WorkflowType: "ExecuteWorkflow",
		TaskQueue:    temporaltype.WorkflowTaskQueue,
		Schedule: jobscheduler.ScheduleSpec{
			Cron: triggerConfig.CronExpression,
		},
		Description: "Scheduled trigger for workflow " + wf.Name,
		Metadata: map[string]string{
			"workflowId": wf.ID.String(),
		},
	}

	if err := s.schedulerManager.CreateOrUpdateSchedule(ctx, scheduleConfig); err != nil {
		s.logger.Error("Failed to setup scheduled trigger",
			zap.String("workflowId", wf.ID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to setup scheduled trigger: %w", err)
	}

	s.logger.Info("Scheduled trigger setup successfully",
		zap.String("workflowId", wf.ID.String()),
		zap.String("scheduleId", scheduleID),
		zap.String("cronExpression", triggerConfig.CronExpression),
	)

	return nil
}

// RemoveScheduledTrigger removes a scheduled trigger for a workflow
func (s *TriggerService) RemoveScheduledTrigger(ctx context.Context, workflowID pulid.ID) error {
	scheduleID := fmt.Sprintf("workflow-%s", workflowID.String())

	if err := s.schedulerManager.DeleteSchedule(ctx, scheduleID); err != nil {
		s.logger.Error("Failed to remove scheduled trigger",
			zap.String("workflowId", workflowID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to remove scheduled trigger: %w", err)
	}

	s.logger.Info("Scheduled trigger removed successfully",
		zap.String("workflowId", workflowID.String()),
		zap.String("scheduleId", scheduleID),
	)

	return nil
}

// PauseScheduledTrigger pauses a scheduled trigger
func (s *TriggerService) PauseScheduledTrigger(ctx context.Context, workflowID pulid.ID) error {
	scheduleID := fmt.Sprintf("workflow-%s", workflowID.String())
	handle := s.schedulerManager.GetHandle(ctx, scheduleID)

	if err := handle.Pause(ctx, client.SchedulePauseOptions{
		Note: "Workflow deactivated",
	}); err != nil {
		s.logger.Error("Failed to pause scheduled trigger",
			zap.String("workflowId", workflowID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to pause scheduled trigger: %w", err)
	}

	s.logger.Info("Scheduled trigger paused",
		zap.String("workflowId", workflowID.String()),
	)

	return nil
}

// ResumeScheduledTrigger resumes a paused scheduled trigger
func (s *TriggerService) ResumeScheduledTrigger(ctx context.Context, workflowID pulid.ID) error {
	scheduleID := fmt.Sprintf("workflow-%s", workflowID.String())
	handle := s.schedulerManager.GetHandle(ctx, scheduleID)

	if err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{
		Note: "Workflow activated",
	}); err != nil {
		s.logger.Error("Failed to resume scheduled trigger",
			zap.String("workflowId", workflowID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to resume scheduled trigger: %w", err)
	}

	s.logger.Info("Scheduled trigger resumed",
		zap.String("workflowId", workflowID.String()),
	)

	return nil
}

// TriggerManualExecution triggers a manual workflow execution
func (s *TriggerService) TriggerManualExecution(
	ctx context.Context,
	req *TriggerWorkflowRequest,
) (*workflow.WorkflowExecution, error) {
	// This is already implemented in ExecutionService.TriggerWorkflow
	// Just delegate to it
	return s.executionService.TriggerWorkflow(ctx, req)
}

// TriggerWebhookExecution triggers a workflow from a webhook
func (s *TriggerService) TriggerWebhookExecution(
	ctx context.Context,
	workflowID pulid.ID,
	webhookData map[string]any,
	orgID, buID, userID pulid.ID,
) (*workflow.WorkflowExecution, error) {
	// Get workflow and validate
	wf, err := s.workflowRepo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     workflowID,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if wf.TriggerType != workflow.TriggerTypeWebhook {
		return nil, fmt.Errorf("workflow trigger type is not webhook")
	}

	// Trigger execution
	return s.executionService.TriggerWorkflow(ctx, &TriggerWorkflowRequest{
		WorkflowID:  workflowID,
		OrgID:       orgID,
		BuID:        buID,
		UserID:      userID,
		TriggerData: webhookData,
	})
}

// TriggerShipmentStatusChange triggers workflows based on shipment status change
func (s *TriggerService) TriggerShipmentStatusChange(
	ctx context.Context,
	shipmentID pulid.ID,
	oldStatus, newStatus string,
	orgID, buID, userID pulid.ID,
) error {
	// Find all active workflows with shipment_status trigger
	workflows, err := s.workflowRepo.List(ctx, &repositories.ListWorkflowRequest{
		Filter: nil, // Get all workflows
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	triggeredCount := 0
	for _, wf := range workflows.Items {
		// Check if workflow should trigger for this status change
		if wf.TriggerType != workflow.TriggerTypeShipmentStatus {
			continue
		}

		if wf.Status != workflow.WorkflowStatusActive {
			continue
		}

		if wf.OrganizationID != orgID || wf.BusinessUnitID != buID {
			continue
		}

		// Parse trigger config to check if this status change matches
		var triggerConfig workflow.ShipmentStatusTriggerConfig
		if err := jsonutils.MustFromJSON(wf.TriggerConfig, &triggerConfig); err != nil {
			s.logger.Warn("Failed to parse trigger config",
				zap.String("workflowId", wf.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Check if the new status matches the trigger
		shouldTrigger := false
		for _, status := range triggerConfig.Statuses {
			if status == newStatus {
				shouldTrigger = true
				break
			}
		}

		if !shouldTrigger {
			continue
		}

		// Trigger workflow execution
		_, err := s.executionService.TriggerWorkflow(ctx, &TriggerWorkflowRequest{
			WorkflowID: wf.ID,
			OrgID:      orgID,
			BuID:       buID,
			UserID:     userID,
			TriggerData: map[string]any{
				"shipmentId":  shipmentID.String(),
				"oldStatus":   oldStatus,
				"newStatus":   newStatus,
				"triggeredAt": time.Now().Format(time.RFC3339),
			},
		})

		if err != nil {
			s.logger.Error("Failed to trigger workflow for shipment status change",
				zap.String("workflowId", wf.ID.String()),
				zap.String("shipmentId", shipmentID.String()),
				zap.String("newStatus", newStatus),
				zap.Error(err),
			)
		} else {
			triggeredCount++
		}
	}

	s.logger.Info("Triggered workflows for shipment status change",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("newStatus", newStatus),
		zap.Int("triggeredCount", triggeredCount),
	)

	return nil
}

// TriggerDocumentUpload triggers workflows based on document upload
func (s *TriggerService) TriggerDocumentUpload(
	ctx context.Context,
	documentID pulid.ID,
	documentType string,
	entityType string,
	entityID pulid.ID,
	orgID, buID, userID pulid.ID,
) error {
	// Find all active workflows with document_uploaded trigger
	workflows, err := s.workflowRepo.List(ctx, &repositories.ListWorkflowRequest{
		Filter: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	triggeredCount := 0
	for _, wf := range workflows.Items {
		if wf.TriggerType != workflow.TriggerTypeDocumentUploaded {
			continue
		}

		if wf.Status != workflow.WorkflowStatusActive {
			continue
		}

		if wf.OrganizationID != orgID || wf.BusinessUnitID != buID {
			continue
		}

		// Parse trigger config
		var triggerConfig workflow.DocumentUploadTriggerConfig
		if err := jsonutils.MustFromJSON(wf.TriggerConfig, &triggerConfig); err != nil {
			s.logger.Warn("Failed to parse trigger config",
				zap.String("workflowId", wf.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Check if document type matches
		shouldTrigger := false
		for _, dt := range triggerConfig.DocumentTypes {
			if dt == documentType {
				shouldTrigger = true
				break
			}
		}

		if !shouldTrigger {
			continue
		}

		// Trigger workflow execution
		_, err := s.executionService.TriggerWorkflow(ctx, &TriggerWorkflowRequest{
			WorkflowID: wf.ID,
			OrgID:      orgID,
			BuID:       buID,
			UserID:     userID,
			TriggerData: map[string]any{
				"documentId":   documentID.String(),
				"documentType": documentType,
				"entityType":   entityType,
				"entityId":     entityID.String(),
				"triggeredAt":  time.Now().Format(time.RFC3339),
			},
		})

		if err != nil {
			s.logger.Error("Failed to trigger workflow for document upload",
				zap.String("workflowId", wf.ID.String()),
				zap.String("documentId", documentID.String()),
				zap.Error(err),
			)
		} else {
			triggeredCount++
		}
	}

	s.logger.Info("Triggered workflows for document upload",
		zap.String("documentId", documentID.String()),
		zap.String("documentType", documentType),
		zap.Int("triggeredCount", triggeredCount),
	)

	return nil
}

// TriggerEntityEvent triggers workflows based on entity create/update events
func (s *TriggerService) TriggerEntityEvent(
	ctx context.Context,
	entityType string,
	entityID pulid.ID,
	eventType workflow.TriggerType, // TriggerTypeEntityCreated or TriggerTypeEntityUpdated
	eventData map[string]any,
	orgID, buID, userID pulid.ID,
) error {
	// Find all active workflows with matching trigger type
	workflows, err := s.workflowRepo.List(ctx, &repositories.ListWorkflowRequest{
		Filter: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	triggeredCount := 0
	for _, wf := range workflows.Items {
		if wf.TriggerType != eventType {
			continue
		}

		if wf.Status != workflow.WorkflowStatusActive {
			continue
		}

		if wf.OrganizationID != orgID || wf.BusinessUnitID != buID {
			continue
		}

		// Parse trigger config
		var triggerConfig workflow.EntityEventTriggerConfig
		if err := jsonutils.MustFromJSON(wf.TriggerConfig, &triggerConfig); err != nil {
			s.logger.Warn("Failed to parse trigger config",
				zap.String("workflowId", wf.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Check if entity type matches
		if triggerConfig.EntityType != entityType {
			continue
		}

		// Trigger workflow execution
		triggerData := map[string]any{
			"entityType":  entityType,
			"entityId":    entityID.String(),
			"eventType":   string(eventType),
			"triggeredAt": time.Now().Format(time.RFC3339),
		}

		// Merge event data
		for k, v := range eventData {
			triggerData[k] = v
		}

		_, err := s.executionService.TriggerWorkflow(ctx, &TriggerWorkflowRequest{
			WorkflowID:  wf.ID,
			OrgID:       orgID,
			BuID:        buID,
			UserID:      userID,
			TriggerData: triggerData,
		})

		if err != nil {
			s.logger.Error("Failed to trigger workflow for entity event",
				zap.String("workflowId", wf.ID.String()),
				zap.String("entityType", entityType),
				zap.String("eventType", string(eventType)),
				zap.Error(err),
			)
		} else {
			triggeredCount++
		}
	}

	s.logger.Info("Triggered workflows for entity event",
		zap.String("entityType", entityType),
		zap.String("eventType", string(eventType)),
		zap.Int("triggeredCount", triggeredCount),
	)

	return nil
}
