package compliancejobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var complianceRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var complianceActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         complianceRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        CredentialExpirySweepWorkflowName,
			Fn:          CredentialExpirySweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Remind drivers and compliance about expiring FMCSA credentials",
		},
	}
}

func CredentialExpirySweepWorkflow(
	ctx workflow.Context,
) (*CredentialExpirySweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, complianceActivityOptions)

	var a *Activities
	result := new(CredentialExpirySweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.CredentialExpirySweepActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Credential expiry sweep workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Credential expiry sweep workflow completed",
		"workersChecked", result.WorkersChecked,
		"driverNotifications", result.DriverNotifications,
		"complianceAlerts", result.ComplianceAlerts,
		"failed", result.Failed,
	)
	return result, nil
}
