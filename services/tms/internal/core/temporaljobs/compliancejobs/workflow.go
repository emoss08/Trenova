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

	training := new(TrainingReminderSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.TrainingReminderSweepActivity,
	).Get(ctx, training); err != nil {
		workflow.GetLogger(ctx).Error("Training reminder sweep failed", "error", err)
		return nil, err
	}
	result.Training = training

	safety := new(SafetyRollupSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.SafetyRollupSweepActivity,
	).Get(ctx, safety); err != nil {
		workflow.GetLogger(ctx).Error("Safety rollup sweep failed", "error", err)
		return nil, err
	}
	result.Safety = safety

	drugAlcohol := new(DrugAlcoholSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.DrugAlcoholSweepActivity,
	).Get(ctx, drugAlcohol); err != nil {
		workflow.GetLogger(ctx).Error("Drug and alcohol sweep failed", "error", err)
		return nil, err
	}
	result.DrugAlcohol = drugAlcohol

	// The digest runs last, after every reminder pass has had its say, so a
	// bundled notice carries the whole night rather than half of it.
	digest := new(DriverDigestSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.DriverDigestActivity,
	).Get(ctx, digest); err != nil {
		workflow.GetLogger(ctx).Error("Driver digest sweep failed", "error", err)
		return nil, err
	}
	result.Digest = digest

	workflow.GetLogger(ctx).Info("Credential expiry sweep workflow completed",
		"workersChecked", result.WorkersChecked,
		"driverNotifications", result.DriverNotifications,
		"complianceAlerts", result.ComplianceAlerts,
		"failed", result.Failed,
		"trainingChecked", training.RecordsChecked,
		"trainingNotifications", training.DriverNotifications,
		"trainingAlerts", training.ComplianceAlerts,
		"drugAlcoholChecked", drugAlcohol.WorkersChecked,
		"drugAlcoholProhibited", drugAlcohol.Prohibited,
		"digestDrivers", digest.DriversNotified,
		"digestObligations", digest.Obligations,
	)
	return result, nil
}
