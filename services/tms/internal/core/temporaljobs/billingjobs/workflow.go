package billingjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var autoPostInvoiceRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
}

var autoPostInvoiceActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         autoPostInvoiceRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AutoPostInvoiceWorkflowName,
			Fn:          AutoPostInvoiceWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Automatically post a draft invoice after billing approval",
		},
	}
}

func AutoPostInvoiceWorkflow(
	ctx workflow.Context,
	payload *AutoPostInvoicePayload,
) (*AutoPostInvoiceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoPostInvoiceActivityOptions)

	var a *Activities
	result := new(AutoPostInvoiceResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.AutoPostInvoiceActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Invoice auto-post workflow failed", "error", err)
		return nil, err
	}

	return result, nil
}
