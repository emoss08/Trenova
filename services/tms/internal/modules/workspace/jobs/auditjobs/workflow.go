package auditjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/auditexportservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditretention"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	RetentionWorkflowName     = "AuditRetentionWorkflow"
	PartitionWorkflowName     = "AuditPartitionWorkflow"
	SealWorkflowName          = "AuditSealWorkflow"
	VerifyWorkflowName        = "AuditVerifyWorkflow"
	ExportCleanupWorkflowName = "AuditExportCleanupWorkflow"
)

var retryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    5 * time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeDataIntegrity.String(),
	},
}

func longActivity(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Hour,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
}

func shortActivity(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    time.Minute,
		RetryPolicy:         retryPolicy,
	})
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RetentionWorkflowName,
			Fn:          AuditRetentionWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Delete audit entries past each organization's retention, class by class, outside legal holds",
		},
		{
			Name:        PartitionWorkflowName,
			Fn:          AuditPartitionWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Create the audit trail's monthly partitions ahead and drop those past ten years",
		},
		{
			Name:        SealWorkflowName,
			Fn:          AuditSealWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Seal the last hour of each organization's audit entries",
		},
		{
			Name:        VerifyWorkflowName,
			Fn:          AuditVerifyWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Verify each organization's audit seals against its entries",
		},
		{
			Name:        auditexportservice.ExportWorkflowName,
			Fn:          AuditExportWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Write an audit log export to object storage",
		},
		{
			Name:        ExportCleanupWorkflowName,
			Fn:          AuditExportCleanupWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Delete audit export files past their download window",
		},
		{
			Name:        auditports.AlertWorkflowName,
			Fn:          AuditAlertWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Notify an organization's security reviewers of a security alert",
		},
	}
}

func AuditRetentionWorkflow(ctx workflow.Context) (*auditretention.SweepResult, error) {
	var a *Activities
	result := new(auditretention.SweepResult)
	err := workflow.ExecuteActivity(longActivity(ctx), a.SweepRetentionActivity).Get(ctx, result)
	return result, err
}

func AuditPartitionWorkflow(ctx workflow.Context) (*auditretention.PartitionResult, error) {
	var a *Activities
	result := new(auditretention.PartitionResult)
	err := workflow.ExecuteActivity(shortActivity(ctx), a.MaintainPartitionsActivity).
		Get(ctx, result)
	return result, err
}

func AuditSealWorkflow(ctx workflow.Context) (*SealResult, error) {
	var a *Activities
	result := new(SealResult)
	err := workflow.ExecuteActivity(longActivity(ctx), a.SealActivity).Get(ctx, result)
	return result, err
}

func AuditVerifyWorkflow(ctx workflow.Context) (*VerifyResult, error) {
	var a *Activities
	result := new(VerifyResult)
	err := workflow.ExecuteActivity(longActivity(ctx), a.VerifyActivity).Get(ctx, result)
	return result, err
}

func AuditExportWorkflow(
	ctx workflow.Context,
	payload *auditexportservice.ExportPayload,
) (*ExportResult, error) {
	var a *Activities
	result := new(ExportResult)
	err := workflow.ExecuteActivity(longActivity(ctx), a.RunExportActivity, payload).
		Get(ctx, result)
	return result, err
}

func AuditExportCleanupWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	var expired int
	err := workflow.ExecuteActivity(shortActivity(ctx), a.ExpireExportsActivity).Get(ctx, &expired)
	return expired, err
}

func AuditAlertWorkflow(ctx workflow.Context, payload *auditports.AlertPayload) (int, error) {
	var a *Activities
	var sent int
	err := workflow.ExecuteActivity(shortActivity(ctx), a.NotifyAlertActivity, payload).
		Get(ctx, &sent)
	return sent, err
}
