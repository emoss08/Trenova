/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package compliance

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"go.temporal.io/sdk/workflow"
)

// ComplianceCheckWorkflow performs compliance checks
func ComplianceCheckWorkflow(ctx workflow.Context, payload *payloads.ComplianceCheckPayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting compliance check",
		"organizationId", payload.OrganizationID.String(),
		"checkType", payload.CheckType,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    1 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run different compliance checks based on type
	var violations []string
	switch payload.CheckType {
	case "all":
		// Run all compliance checks
		var dotViolations []string
		err := workflow.ExecuteActivity(ctx, CheckDOTComplianceActivity, payload).Get(ctx, &dotViolations)
		if err != nil {
			logger.Error("DOT compliance check failed", "error", err)
			return err
		}
		violations = append(violations, dotViolations...)

		var hazmatViolations []string
		err = workflow.ExecuteActivity(ctx, CheckHazmatComplianceActivity, payload).Get(ctx, &hazmatViolations)
		if err != nil {
			logger.Error("Hazmat compliance check failed", "error", err)
			return err
		}
		violations = append(violations, hazmatViolations...)

	case "dot":
		err := workflow.ExecuteActivity(ctx, CheckDOTComplianceActivity, payload).Get(ctx, &violations)
		if err != nil {
			return err
		}

	case "hazmat":
		err := workflow.ExecuteActivity(ctx, CheckHazmatComplianceActivity, payload).Get(ctx, &violations)
		if err != nil {
			return err
		}
	}

	logger.Info("compliance check completed", "violationCount", len(violations))
	return nil
}

// WorkflowDefinition defines a workflow with its configuration
type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

// RegisterWorkflows registers all compliance-related workflows
func RegisterWorkflows() []WorkflowDefinition {
	return []WorkflowDefinition{
		{
			Name:        "ComplianceCheckWorkflow",
			Fn:          ComplianceCheckWorkflow,
			TaskQueue:   "compliance-tasks",
			Description: "Performs compliance checks for DOT and Hazmat regulations",
		},
	}
}