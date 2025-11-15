package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListWorkflowRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetWorkflowByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type WorkflowRepository interface {
	List(ctx context.Context, req *ListWorkflowRequest) (*pagination.ListResult[*workflow.Workflow], error)
	GetByID(ctx context.Context, opts GetWorkflowByIDRequest) (*workflow.Workflow, error)
	Create(ctx context.Context, w *workflow.Workflow) (*workflow.Workflow, error)
	Update(ctx context.Context, w *workflow.Workflow) (*workflow.Workflow, error)
	Delete(ctx context.Context, id, orgID, buID pulid.ID) error

	// Version management
	CreateVersion(ctx context.Context, v *workflow.WorkflowVersion) (*workflow.WorkflowVersion, error)
	GetVersionByID(ctx context.Context, id, orgID, buID pulid.ID) (*workflow.WorkflowVersion, error)
	GetVersionsByWorkflowID(ctx context.Context, workflowID, orgID, buID pulid.ID) ([]*workflow.WorkflowVersion, error)
	GetLatestVersion(ctx context.Context, workflowID, orgID, buID pulid.ID) (*workflow.WorkflowVersion, error)
	PublishVersion(ctx context.Context, workflowID, versionID, orgID, buID, userID pulid.ID) error

	// Node management
	CreateNodes(ctx context.Context, nodes []*workflow.WorkflowNode) error
	GetNodesByVersionID(ctx context.Context, versionID, orgID, buID pulid.ID) ([]*workflow.WorkflowNode, error)
	DeleteNodesByVersionID(ctx context.Context, versionID, orgID, buID pulid.ID) error

	// Edge management
	CreateEdges(ctx context.Context, edges []*workflow.WorkflowEdge) error
	GetEdgesByVersionID(ctx context.Context, versionID, orgID, buID pulid.ID) ([]*workflow.WorkflowEdge, error)
	DeleteEdgesByVersionID(ctx context.Context, versionID, orgID, buID pulid.ID) error

	// Status management
	UpdateStatus(ctx context.Context, id, orgID, buID pulid.ID, status workflow.WorkflowStatus) error
	GetActiveWorkflowsByTrigger(ctx context.Context, triggerType workflow.TriggerType, orgID, buID pulid.ID) ([]*workflow.Workflow, error)
}

type ListWorkflowExecutionRequest struct {
	Filter     *pagination.QueryOptions `json:"filter" form:"filter"`
	WorkflowID *pulid.ID                `json:"workflowId,omitempty" form:"workflowId"`
	Status     *workflow.ExecutionStatus `json:"status,omitempty" form:"status"`
}

type GetWorkflowExecutionByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type WorkflowExecutionRepository interface {
	List(ctx context.Context, req *ListWorkflowExecutionRequest) (*pagination.ListResult[*workflow.WorkflowExecution], error)
	GetByID(ctx context.Context, opts GetWorkflowExecutionByIDRequest) (*workflow.WorkflowExecution, error)
	Create(ctx context.Context, exec *workflow.WorkflowExecution) (*workflow.WorkflowExecution, error)
	Update(ctx context.Context, exec *workflow.WorkflowExecution) (*workflow.WorkflowExecution, error)

	// Execution step management
	CreateStep(ctx context.Context, step *workflow.WorkflowExecutionStep) (*workflow.WorkflowExecutionStep, error)
	UpdateStep(ctx context.Context, step *workflow.WorkflowExecutionStep) (*workflow.WorkflowExecutionStep, error)
	GetStepsByExecutionID(ctx context.Context, executionID, orgID, buID pulid.ID) ([]*workflow.WorkflowExecutionStep, error)

	// Status management
	UpdateStatus(ctx context.Context, id, orgID, buID pulid.ID, status workflow.ExecutionStatus) error
	CancelExecution(ctx context.Context, id, orgID, buID pulid.ID) error

	// Temporal integration
	GetByTemporalWorkflowID(ctx context.Context, temporalWorkflowID string, orgID, buID pulid.ID) (*workflow.WorkflowExecution, error)
	UpdateTemporalInfo(ctx context.Context, id, orgID, buID pulid.ID, temporalWorkflowID, temporalRunID string) error
}

type ListWorkflowTemplateRequest struct {
	Filter   *pagination.QueryOptions `json:"filter" form:"filter"`
	Category *string                  `json:"category,omitempty" form:"category"`
}

type GetWorkflowTemplateByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type WorkflowTemplateRepository interface {
	List(ctx context.Context, req *ListWorkflowTemplateRequest) (*pagination.ListResult[*workflow.WorkflowTemplate], error)
	GetByID(ctx context.Context, opts GetWorkflowTemplateByIDRequest) (*workflow.WorkflowTemplate, error)
	Create(ctx context.Context, t *workflow.WorkflowTemplate) (*workflow.WorkflowTemplate, error)
	Update(ctx context.Context, t *workflow.WorkflowTemplate) (*workflow.WorkflowTemplate, error)
	Delete(ctx context.Context, id, orgID, buID pulid.ID) error

	// System templates
	GetSystemTemplates(ctx context.Context) ([]*workflow.WorkflowTemplate, error)
	GetPublicTemplates(ctx context.Context, orgID, buID pulid.ID) ([]*workflow.WorkflowTemplate, error)

	// Usage tracking
	IncrementUsage(ctx context.Context, id, orgID, buID pulid.ID) error
}
