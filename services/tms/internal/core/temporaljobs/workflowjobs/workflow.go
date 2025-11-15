package workflowjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	wf "go.temporal.io/sdk/workflow"
)

const (
	WorkflowTaskQueue = "workflow-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "ExecuteWorkflow",
			Fn:          ExecuteWorkflow,
			TaskQueue:   WorkflowTaskQueue,
			Description: "Execute a workflow automation",
		},
	}
}

// ExecuteWorkflow is the main Temporal workflow for executing workflow automations
func ExecuteWorkflow(
	ctx wf.Context,
	payload *ExecuteWorkflowPayload,
) (*ExecuteWorkflowResult, error) {
	logger := wf.GetLogger(ctx)
	logger.Info("Starting workflow execution",
		"executionId", payload.ExecutionID.String(),
		"workflowId", payload.WorkflowID.String(),
		"triggerType", payload.TriggerType,
	)

	// Set activity options
	ao := wf.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}
	ctx = wf.WithActivityOptions(ctx, ao)

	startTime := wf.Now(ctx)
	workflowState := make(map[string]any)
	stepsExecuted := 0

	// Activity reference
	var a *Activities

	// Step 1: Load workflow definition
	logger.Info("Loading workflow definition")
	var loadDefResult LoadWorkflowDefinitionResult
	err := wf.ExecuteActivity(ctx, a.LoadWorkflowDefinition, &LoadWorkflowDefinitionPayload{
		WorkflowVersionID: payload.WorkflowVersionID,
		OrgID:             payload.OrganizationID,
		BuID:              payload.BusinessUnitID,
	}).Get(ctx, &loadDefResult)
	if err != nil {
		return &ExecuteWorkflowResult{
			ExecutionID: payload.ExecutionID,
			Status:      string(workflow.ExecutionStatusFailed),
			Error:       fmt.Sprintf("Failed to load workflow definition: %v", err),
		}, err
	}

	// Step 2: Update execution status to running
	err = wf.ExecuteActivity(ctx, a.UpdateExecutionStatus, &UpdateExecutionStatusPayload{
		ExecutionID: payload.ExecutionID,
		Status:      workflow.ExecutionStatusRunning,
		OrgID:       payload.OrganizationID,
		BuID:        payload.BusinessUnitID,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update execution status to running", "error", err)
	}

	// Step 3: Execute workflow nodes
	logger.Info("Executing workflow nodes", "nodeCount", len(loadDefResult.Nodes))

	// Find the trigger node (starting point)
	var triggerNode *workflow.WorkflowNode
	for _, node := range loadDefResult.Nodes {
		if node.NodeType == workflow.NodeTypeTrigger {
			triggerNode = node
			break
		}
	}

	if triggerNode == nil {
		err := fmt.Errorf("no trigger node found in workflow")
		return &ExecuteWorkflowResult{
			ExecutionID: payload.ExecutionID,
			Status:      string(workflow.ExecutionStatusFailed),
			Error:       err.Error(),
		}, err
	}

	// Initialize workflow state with trigger data
	workflowState["trigger"] = payload.TriggerData

	// Execute the workflow graph starting from trigger node
	err = executeNodeGraph(
		ctx,
		a,
		triggerNode,
		loadDefResult.Nodes,
		loadDefResult.Edges,
		&NodeExecutionContext{
			ExecutionID:   payload.ExecutionID,
			WorkflowID:    payload.WorkflowID,
			OrgID:         payload.OrganizationID,
			BuID:          payload.BusinessUnitID,
			UserID:        payload.UserID,
			TriggerData:   payload.TriggerData,
			WorkflowState: workflowState,
		},
		&stepsExecuted,
	)

	duration := wf.Now(ctx).Sub(startTime).Milliseconds()

	if err != nil {
		logger.Error("Workflow execution failed", "error", err, "stepsExecuted", stepsExecuted)

		// Update execution status to failed
		_ = wf.ExecuteActivity(ctx, a.UpdateExecutionStatus, &UpdateExecutionStatusPayload{
			ExecutionID: payload.ExecutionID,
			Status:      workflow.ExecutionStatusFailed,
			OrgID:       payload.OrganizationID,
			BuID:        payload.BusinessUnitID,
			ErrorMsg:    err.Error(),
		}).Get(ctx, nil)

		return &ExecuteWorkflowResult{
			ExecutionID:   payload.ExecutionID,
			Status:        string(workflow.ExecutionStatusFailed),
			StepsExecuted: stepsExecuted,
			Duration:      duration,
			Error:         err.Error(),
		}, err
	}

	// Step 4: Update execution status to completed
	logger.Info("Workflow execution completed successfully", "stepsExecuted", stepsExecuted)
	err = wf.ExecuteActivity(ctx, a.UpdateExecutionStatus, &UpdateExecutionStatusPayload{
		ExecutionID: payload.ExecutionID,
		Status:      workflow.ExecutionStatusCompleted,
		OrgID:       payload.OrganizationID,
		BuID:        payload.BusinessUnitID,
		OutputData:  workflowState,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update execution status to completed", "error", err)
	}

	return &ExecuteWorkflowResult{
		ExecutionID:   payload.ExecutionID,
		Status:        string(workflow.ExecutionStatusCompleted),
		OutputData:    workflowState,
		StepsExecuted: stepsExecuted,
		Duration:      duration,
	}, nil
}

// executeNodeGraph recursively executes nodes following the workflow edges
func executeNodeGraph(
	ctx wf.Context,
	a *Activities,
	currentNode *workflow.WorkflowNode,
	allNodes []*workflow.WorkflowNode,
	allEdges []*workflow.WorkflowEdge,
	execCtx *NodeExecutionContext,
	stepsExecuted *int,
) error {
	logger := wf.GetLogger(ctx)

	// Skip trigger nodes (they don't execute, just mark the start)
	if currentNode.NodeType == workflow.NodeTypeTrigger {
		logger.Info("Skipping trigger node", "nodeKey", currentNode.NodeKey)
	} else {
		// Execute the current node
		*stepsExecuted++
		logger.Info("Executing node", "nodeKey", currentNode.NodeKey, "nodeType", currentNode.NodeType, "stepNumber", *stepsExecuted)

		var nodeResult ExecuteNodeResult
		err := wf.ExecuteActivity(ctx, a.ExecuteNode, &ExecuteNodePayload{
			ExecutionID: execCtx.ExecutionID,
			StepNumber:  *stepsExecuted,
			NodeID:      currentNode.ID,
			NodeKey:     currentNode.NodeKey,
			NodeType:    currentNode.NodeType,
			ActionType:  currentNode.ActionType,
			Config:      currentNode.Config,
			InputData:   execCtx.WorkflowState,
			OrgID:       execCtx.OrgID,
			BuID:        execCtx.BuID,
			UserID:      execCtx.UserID,
		}).Get(ctx, &nodeResult)
		if err != nil {
			logger.Error("Node execution failed", "nodeKey", currentNode.NodeKey, "error", err)
			return fmt.Errorf("node %s execution failed: %w", currentNode.NodeKey, err)
		}

		// Update workflow state with node output
		execCtx.WorkflowState[currentNode.NodeKey] = nodeResult.OutputData
		logger.Info("Node executed successfully", "nodeKey", currentNode.NodeKey, "duration", nodeResult.Duration)
	}

	// Check if this is an end node
	if currentNode.NodeType == workflow.NodeTypeEnd {
		logger.Info("Reached end node", "nodeKey", currentNode.NodeKey)
		return nil
	}

	// Find outgoing edges from current node
	var outgoingEdges []*workflow.WorkflowEdge
	for _, edge := range allEdges {
		if edge.SourceNodeID == currentNode.ID {
			outgoingEdges = append(outgoingEdges, edge)
		}
	}

	if len(outgoingEdges) == 0 {
		logger.Warn("No outgoing edges found for non-end node", "nodeKey", currentNode.NodeKey)
		return nil // Workflow ends here
	}

	// Handle different node types
	switch currentNode.NodeType {
	case workflow.NodeTypeCondition:
		// For condition nodes, evaluate which path to take
		// The condition result should be in the node output
		conditionResult := execCtx.WorkflowState[currentNode.NodeKey]
		var takeTruePath bool
		if resultMap, ok := conditionResult.(map[string]any); ok {
			if result, ok := resultMap["result"].(bool); ok {
				takeTruePath = result
			}
		}

		// Find the appropriate edge based on condition result
		for _, edge := range outgoingEdges {
			sourceHandle := ""
			if edge.SourceHandle != nil {
				sourceHandle = *edge.SourceHandle
			}

			if (takeTruePath && sourceHandle == "true") ||
				(!takeTruePath && sourceHandle == "false") {
				nextNode := findNodeByID(allNodes, edge.TargetNodeID)
				if nextNode != nil {
					return executeNodeGraph(
						ctx,
						a,
						nextNode,
						allNodes,
						allEdges,
						execCtx,
						stepsExecuted,
					)
				}
			}
		}

	default:
		// For other nodes, execute all outgoing paths (usually just one)
		for _, edge := range outgoingEdges {
			nextNode := findNodeByID(allNodes, edge.TargetNodeID)
			if nextNode != nil {
				err := executeNodeGraph(
					ctx,
					a,
					nextNode,
					allNodes,
					allEdges,
					execCtx,
					stepsExecuted,
				)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// findNodeByID finds a node by its ID
func findNodeByID(nodes []*workflow.WorkflowNode, id pulid.ID) *workflow.WorkflowNode {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}
