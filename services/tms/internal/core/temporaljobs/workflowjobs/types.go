package workflowjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils"
)

// ExecuteWorkflowPayload is the input for workflow execution
type ExecuteWorkflowPayload struct {
	temporaltype.BasePayload
	ExecutionID       pulid.ID        `json:"executionId"`
	WorkflowID        pulid.ID        `json:"workflowId"`
	WorkflowVersionID pulid.ID        `json:"workflowVersionId"`
	TriggerData       map[string]any  `json:"triggerData"`
	TriggerType       workflow.TriggerType `json:"triggerType"`
}

// ExecuteWorkflowResult is the output of workflow execution
type ExecuteWorkflowResult struct {
	ExecutionID   pulid.ID       `json:"executionId"`
	Status        string         `json:"status"`
	OutputData    map[string]any `json:"outputData"`
	StepsExecuted int            `json:"stepsExecuted"`
	Duration      int64          `json:"duration"` // milliseconds
	Error         string         `json:"error,omitempty"`
}

// ExecuteNodePayload is the input for executing a single node
type ExecuteNodePayload struct {
	ExecutionID pulid.ID             `json:"executionId"`
	StepNumber  int                  `json:"stepNumber"`
	NodeID      pulid.ID             `json:"nodeId"`
	NodeKey     string               `json:"nodeKey"`
	NodeType    workflow.NodeType    `json:"nodeType"`
	ActionType  *workflow.ActionType `json:"actionType,omitempty"`
	Config      utils.JSONB          `json:"config"`
	InputData   map[string]any       `json:"inputData"`
	OrgID       pulid.ID             `json:"orgId"`
	BuID        pulid.ID             `json:"buId"`
	UserID      pulid.ID             `json:"userId"`
}

// ExecuteNodeResult is the output of node execution
type ExecuteNodeResult struct {
	StepID     pulid.ID       `json:"stepId"`
	Status     string         `json:"status"`
	OutputData map[string]any `json:"outputData"`
	Duration   int64          `json:"duration"` // milliseconds
	Error      string         `json:"error,omitempty"`
}

// WorkflowDefinitionData represents the parsed workflow definition
type WorkflowDefinitionData struct {
	Nodes []*workflow.WorkflowNode `json:"nodes"`
	Edges []*workflow.WorkflowEdge `json:"edges"`
}

// NodeExecutionContext contains context for node execution
type NodeExecutionContext struct {
	ExecutionID   pulid.ID
	WorkflowID    pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	TriggerData   map[string]any
	WorkflowState map[string]any // State accumulated during workflow execution
}

// LoadWorkflowDefinitionPayload is the input for loading workflow definition
type LoadWorkflowDefinitionPayload struct {
	WorkflowVersionID pulid.ID `json:"workflowVersionId"`
	OrgID             pulid.ID `json:"orgId"`
	BuID              pulid.ID `json:"buId"`
}

// LoadWorkflowDefinitionResult is the output of loading workflow definition
type LoadWorkflowDefinitionResult struct {
	Nodes []*workflow.WorkflowNode `json:"nodes"`
	Edges []*workflow.WorkflowEdge `json:"edges"`
}

// UpdateExecutionStatusPayload is the input for updating execution status
type UpdateExecutionStatusPayload struct {
	ExecutionID pulid.ID                  `json:"executionId"`
	Status      workflow.ExecutionStatus  `json:"status"`
	OrgID       pulid.ID                  `json:"orgId"`
	BuID        pulid.ID                  `json:"buId"`
	OutputData  map[string]any            `json:"outputData,omitempty"`
	ErrorMsg    string                    `json:"errorMsg,omitempty"`
}
