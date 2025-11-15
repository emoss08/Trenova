package workflowjobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	WorkflowRepo          repositories.WorkflowRepository
	WorkflowExecutionRepo repositories.WorkflowExecutionRepository
	Logger                *zap.Logger
	ActionHandlers        *ActionHandlers
}

type Activities struct {
	workflowRepo     repositories.WorkflowRepository
	executionRepo    repositories.WorkflowExecutionRepository
	logger           *zap.Logger
	actionHandlers   *ActionHandlers
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		workflowRepo:   p.WorkflowRepo,
		executionRepo:  p.WorkflowExecutionRepo,
		logger:         p.Logger.Named("workflow-activities"),
		actionHandlers: p.ActionHandlers,
	}
}

// LoadWorkflowDefinition loads the workflow definition from the database
func (a *Activities) LoadWorkflowDefinition(
	ctx context.Context,
	payload *LoadWorkflowDefinitionPayload,
) (*LoadWorkflowDefinitionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Loading workflow definition", "versionId", payload.WorkflowVersionID.String())

	activity.RecordHeartbeat(ctx, "loading workflow version")

	// Get the workflow version with nodes and edges
	version, err := a.workflowRepo.GetVersionByID(ctx, payload.WorkflowVersionID, payload.OrgID, payload.BuID)
	if err != nil {
		a.logger.Error("failed to load workflow version", zap.Error(err))
		return nil, fmt.Errorf("failed to load workflow version: %w", err)
	}

	activity.RecordHeartbeat(ctx, "loading nodes and edges")

	// Get nodes
	nodes, err := a.workflowRepo.GetNodesByVersionID(ctx, payload.WorkflowVersionID, payload.OrgID, payload.BuID)
	if err != nil {
		a.logger.Error("failed to load workflow nodes", zap.Error(err))
		return nil, fmt.Errorf("failed to load workflow nodes: %w", err)
	}

	// Get edges
	edges, err := a.workflowRepo.GetEdgesByVersionID(ctx, payload.WorkflowVersionID, payload.OrgID, payload.BuID)
	if err != nil {
		a.logger.Error("failed to load workflow edges", zap.Error(err))
		return nil, fmt.Errorf("failed to load workflow edges: %w", err)
	}

	logger.Info("Workflow definition loaded successfully",
		"nodeCount", len(nodes),
		"edgeCount", len(edges))

	return &LoadWorkflowDefinitionResult{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// UpdateExecutionStatus updates the execution status in the database
func (a *Activities) UpdateExecutionStatus(
	ctx context.Context,
	payload *UpdateExecutionStatusPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating execution status",
		"executionId", payload.ExecutionID.String(),
		"status", payload.Status)

	activity.RecordHeartbeat(ctx, "updating execution status")

	// Get the execution
	execution, err := a.executionRepo.GetByID(ctx, repositories.GetWorkflowExecutionByIDRequest{
		ID:    payload.ExecutionID,
		OrgID: payload.OrgID,
		BuID:  payload.BuID,
	})
	if err != nil {
		a.logger.Error("failed to get execution", zap.Error(err))
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Update status
	execution.Status = payload.Status
	now := time.Now().Unix()

	switch payload.Status {
	case workflow.ExecutionStatusRunning:
		execution.StartedAt = &now
	case workflow.ExecutionStatusCompleted, workflow.ExecutionStatusFailed, workflow.ExecutionStatusCanceled:
		execution.CompletedAt = &now
		if execution.StartedAt != nil {
			duration := (now - *execution.StartedAt) * 1000 // Convert to milliseconds
			execution.DurationMs = &duration
		}
		if payload.OutputData != nil {
			outputData := jsonutils.MustToJSONB(payload.OutputData)
			execution.OutputData = &outputData
		}
		if payload.ErrorMsg != "" {
			execution.ErrorMessage = &payload.ErrorMsg
		}
	}

	// Save updated execution
	_, err = a.executionRepo.Update(ctx, execution)
	if err != nil {
		a.logger.Error("failed to update execution", zap.Error(err))
		return fmt.Errorf("failed to update execution: %w", err)
	}

	logger.Info("Execution status updated successfully")
	return nil
}

// ExecuteNode executes a single workflow node
func (a *Activities) ExecuteNode(
	ctx context.Context,
	payload *ExecuteNodePayload,
) (*ExecuteNodeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing node",
		"nodeKey", payload.NodeKey,
		"nodeType", payload.NodeType,
		"stepNumber", payload.StepNumber)

	startTime := time.Now()

	activity.RecordHeartbeat(ctx, "creating execution step")

	// Create execution step record
	step := &workflow.WorkflowExecutionStep{
		ExecutionID:    payload.ExecutionID,
		OrganizationID: payload.OrgID,
		BusinessUnitID: payload.BuID,
		NodeID:         payload.NodeID,
		NodeKey:        payload.NodeKey,
		NodeType:       payload.NodeType,
		ActionType:     payload.ActionType,
		StepNumber:     payload.StepNumber,
		Status:         workflow.StepStatusRunning,
		InputData:      jsonToJSONBPtr(payload.InputData),
		StartedAt:      timePtr(time.Now().Unix()),
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	createdStep, err := a.executionRepo.CreateStep(ctx, step)
	if err != nil {
		a.logger.Error("failed to create execution step", zap.Error(err))
		return nil, fmt.Errorf("failed to create execution step: %w", err)
	}

	activity.RecordHeartbeat(ctx, "executing node logic")

	// Execute node based on type
	var outputData map[string]any
	var execErr error

	switch payload.NodeType {
	case workflow.NodeTypeAction:
		outputData, execErr = a.executeAction(ctx, payload)
	case workflow.NodeTypeCondition:
		outputData, execErr = a.evaluateCondition(ctx, payload)
	case workflow.NodeTypeDelay:
		outputData, execErr = a.executeDelay(ctx, payload)
	case workflow.NodeTypeLoop:
		outputData, execErr = a.executeLoop(ctx, payload)
	default:
		outputData = map[string]any{"result": "skipped"}
	}

	duration := time.Since(startTime).Milliseconds()

	// Update step status
	createdStep.CompletedAt = timePtr(time.Now().Unix())
	createdStep.DurationMs = &duration

	if execErr != nil {
		logger.Error("Node execution failed", "error", execErr)
		createdStep.Status = workflow.StepStatusFailed
		errMsg := execErr.Error()
		createdStep.ErrorMessage = &errMsg
		createdStep.UpdatedAt = time.Now().Unix()

		_, _ = a.executionRepo.UpdateStep(ctx, createdStep)

		return &ExecuteNodeResult{
			StepID:   createdStep.ID,
			Status:   string(workflow.StepStatusFailed),
			Duration: duration,
			Error:    execErr.Error(),
		}, execErr
	}

	createdStep.Status = workflow.StepStatusCompleted
	createdStep.OutputData = jsonToJSONBPtr(outputData)
	createdStep.UpdatedAt = time.Now().Unix()

	_, err = a.executionRepo.UpdateStep(ctx, createdStep)
	if err != nil {
		logger.Warn("Failed to update step status", "error", err)
	}

	logger.Info("Node executed successfully", "duration", duration)

	return &ExecuteNodeResult{
		StepID:     createdStep.ID,
		Status:     string(workflow.StepStatusCompleted),
		OutputData: outputData,
		Duration:   duration,
	}, nil
}

// executeAction executes an action node
func (a *Activities) executeAction(ctx context.Context, payload *ExecuteNodePayload) (map[string]any, error) {
	if payload.ActionType == nil {
		return nil, fmt.Errorf("action type is required for action nodes")
	}

	// Parse config from JSONB
	var config map[string]any
	if err := json.Unmarshal(payload.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse node config: %w", err)
	}

	// Execute the appropriate action handler
	return a.actionHandlers.Execute(ctx, &ActionExecutionContext{
		ActionType: *payload.ActionType,
		Config:     config,
		InputData:  payload.InputData,
		OrgID:      payload.OrgID,
		BuID:       payload.BuID,
		UserID:     payload.UserID,
	})
}

// evaluateCondition evaluates a condition node
func (a *Activities) evaluateCondition(ctx context.Context, payload *ExecuteNodePayload) (map[string]any, error) {
	// Parse config from JSONB
	var config map[string]any
	if err := json.Unmarshal(payload.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse condition config: %w", err)
	}

	// Evaluate condition (simplified - you can make this more sophisticated)
	result := evaluateConditionLogic(config, payload.InputData)

	return map[string]any{
		"result": result,
		"config": config,
	}, nil
}

// executeDelay executes a delay node
func (a *Activities) executeDelay(ctx context.Context, payload *ExecuteNodePayload) (map[string]any, error) {
	// Parse config from JSONB
	var config map[string]any
	if err := json.Unmarshal(payload.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse delay config: %w", err)
	}

	// Get delay duration from config
	delaySeconds, ok := config["delaySeconds"].(float64)
	if !ok {
		delaySeconds = 1 // Default 1 second
	}

	// Sleep for the specified duration
	activity.GetLogger(ctx).Info("Delaying execution", "seconds", delaySeconds)
	time.Sleep(time.Duration(delaySeconds) * time.Second)

	return map[string]any{
		"delayed": delaySeconds,
	}, nil
}

// executeLoop executes a loop node
func (a *Activities) executeLoop(ctx context.Context, payload *ExecuteNodePayload) (map[string]any, error) {
	// This is a placeholder - loop logic would be implemented in the workflow itself
	// For now, we'll just pass through
	return map[string]any{
		"loopExecuted": true,
	}, nil
}

// Helper functions

func evaluateConditionLogic(config map[string]any, inputData map[string]any) bool {
	// Simplified condition evaluation
	// You can extend this to support complex expressions

	field, ok := config["field"].(string)
	if !ok {
		return false
	}

	operator, ok := config["operator"].(string)
	if !ok {
		return false
	}

	expectedValue := config["value"]
	actualValue := getNestedValue(inputData, field)

	switch operator {
	case "equals":
		return actualValue == expectedValue
	case "notEquals":
		return actualValue != expectedValue
	case "contains":
		if str, ok := actualValue.(string); ok {
			if expected, ok := expectedValue.(string); ok {
				return contains(str, expected)
			}
		}
	case "greaterThan":
		return compareNumbers(actualValue, expectedValue, func(a, b float64) bool { return a > b })
	case "lessThan":
		return compareNumbers(actualValue, expectedValue, func(a, b float64) bool { return a < b })
	}

	return false
}

func getNestedValue(data map[string]any, path string) any {
	// Simple implementation - can be extended to support nested paths
	return data[path]
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && haystack[:len(needle)] == needle
}

func compareNumbers(a, b any, compare func(float64, float64) bool) bool {
	var aNum, bNum float64
	var ok bool

	if aNum, ok = a.(float64); !ok {
		if aInt, ok := a.(int); ok {
			aNum = float64(aInt)
		} else {
			return false
		}
	}

	if bNum, ok = b.(float64); !ok {
		if bInt, ok := b.(int); ok {
			bNum = float64(bInt)
		} else {
			return false
		}
	}

	return compare(aNum, bNum)
}

func jsonToJSONBPtr(data map[string]any) *utils.JSONB {
	if data == nil {
		return nil
	}
	jsonb := jsonutils.MustToJSONB(data)
	return &jsonb
}

func timePtr(t int64) *int64 {
	return &t
}
