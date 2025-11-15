package workflowjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

// Registry provides workflow and activity registration for workflow automation
type Registry struct {
	logger              *zap.Logger
	workflowRepo        repositories.WorkflowRepository
	executionRepo       repositories.WorkflowExecutionRepository
	shipmentRepo        repositories.ShipmentRepository
	notificationService services.NotificationService
	auditService        services.AuditService
}

// RegistryParams holds the dependencies for creating a workflow jobs registry
type RegistryParams struct {
	Logger              *zap.Logger
	WorkflowRepo        repositories.WorkflowRepository
	ExecutionRepo       repositories.WorkflowExecutionRepository
	ShipmentRepo        repositories.ShipmentRepository
	NotificationService services.NotificationService
	AuditService        services.AuditService
}

// NewRegistry creates a new workflow jobs registry
func NewRegistry(p RegistryParams) *Registry {
	return &Registry{
		logger:              p.Logger,
		workflowRepo:        p.WorkflowRepo,
		executionRepo:       p.ExecutionRepo,
		shipmentRepo:        p.ShipmentRepo,
		notificationService: p.NotificationService,
		auditService:        p.AuditService,
	}
}

// Register implements the WorkflowRegistry interface
func (r *Registry) Register(w worker.Worker) error {
	// Register workflows
	workflows := RegisterWorkflows()
	for _, workflow := range workflows {
		w.RegisterWorkflow(workflow.Fn)
		r.logger.Info("Registered workflow",
			zap.String("name", workflow.Name),
			zap.String("taskQueue", string(workflow.TaskQueue)),
		)
	}

	// Create activities instance
	activities := NewActivities(ActivitiesParams{
		Logger:              r.logger,
		WorkflowRepo:        r.workflowRepo,
		ExecutionRepo:       r.executionRepo,
		ShipmentRepo:        r.shipmentRepo,
		NotificationService: r.notificationService,
		AuditService:        r.auditService,
	})

	// Register activities
	activityList := RegisterActivities()
	for _, activity := range activityList {
		w.RegisterActivity(activity.Fn)
		r.logger.Info("Registered activity",
			zap.String("name", activity.Name),
		)
	}

	// Register the activities struct (for method-based activity registration)
	w.RegisterActivity(activities)

	r.logger.Info("Workflow jobs registry completed",
		zap.Int("workflows", len(workflows)),
		zap.Int("activities", len(activityList)),
	)

	return nil
}

// GetTaskQueues returns the task queues this registry handles
func (r *Registry) GetTaskQueues() []temporaltype.TaskQueue {
	return []temporaltype.TaskQueue{
		WorkflowTaskQueue,
	}
}

// RegisterActivities returns the list of activities to register
func RegisterActivities() []temporaltype.ActivityDefinition {
	return []temporaltype.ActivityDefinition{
		{
			Name:        "LoadWorkflowDefinition",
			Description: "Load workflow definition and nodes from database",
		},
		{
			Name:        "UpdateExecutionStatus",
			Description: "Update workflow execution status",
		},
		{
			Name:        "ExecuteNode",
			Description: "Execute a single workflow node",
		},
	}
}
