/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package patterns

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"go.temporal.io/sdk/workflow"
)

// PatternAnalysisWorkflow analyzes patterns for dedicated lanes
func PatternAnalysisWorkflow(ctx workflow.Context, payload *payloads.PatternAnalysisPayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting pattern analysis",
		"organizationId", payload.OrganizationID.String(),
		"triggerReason", payload.TriggerReason,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    1 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, AnalyzePatternActivity, payload).Get(ctx, &result)
	if err != nil {
		logger.Error("pattern analysis failed", "error", err)
		return err
	}

	logger.Info("pattern analysis completed", "result", result)
	return nil
}

// ExpireSuggestionsWorkflow expires old pattern suggestions
func ExpireSuggestionsWorkflow(ctx workflow.Context, payload *payloads.ExpireSuggestionsPayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("expiring old suggestions",
		"organizationId", payload.OrganizationID.String(),
		"batchSize", payload.BatchSize,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var expiredCount int
	err := workflow.ExecuteActivity(ctx, ExpireSuggestionsActivity, payload).Get(ctx, &expiredCount)
	if err != nil {
		logger.Error("failed to expire suggestions", "error", err)
		return err
	}

	logger.Info("expired suggestions successfully", "count", expiredCount)
	return nil
}

// WorkflowDefinition defines a workflow with its configuration
type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

// RegisterWorkflows registers all pattern-related workflows
func RegisterWorkflows() []WorkflowDefinition {
	return []WorkflowDefinition{
		{
			Name:        "PatternAnalysisWorkflow",
			Fn:          PatternAnalysisWorkflow,
			TaskQueue:   "pattern-analysis-tasks",
			Description: "Analyzes shipment patterns for dedicated lanes",
		},
		{
			Name:        "ExpireSuggestionsWorkflow",
			Fn:          ExpireSuggestionsWorkflow,
			TaskQueue:   "default-tasks",
			Description: "Expires old pattern suggestions",
		},
	}
}