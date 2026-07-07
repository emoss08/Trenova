package edijobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var approveLoadTenderRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    time.Minute,
}

var approveLoadTenderActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         approveLoadTenderRetryPolicy,
}

var deliverMessageActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    6,
		MaximumInterval:    15 * time.Minute,
	},
}

var deadLetterActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    5,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.ApproveLoadTenderTransferWorkflowName,
			Fn:          ApproveLoadTenderTransferWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Approve an inbound EDI load tender transfer",
		},
		{
			Name:        temporaltype.DeliverEDIMessageWorkflowName,
			Fn:          DeliverEDIMessageWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Deliver an outbound EDI message to its trading partner",
		},
		{
			Name:        PollInboundMailboxesWorkflowName,
			Fn:          PollInboundMailboxesWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Poll partner SFTP and VAN mailboxes for inbound EDI files",
		},
		{
			Name:        temporaltype.ProcessInboundEDIFileWorkflowName,
			Fn:          ProcessInboundEDIFileWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Parse and route a staged inbound EDI file",
		},
	}
}

func ApproveLoadTenderTransferWorkflow(
	ctx workflow.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, approveLoadTenderActivityOptions)

	var a *Activities
	result := new(ApproveLoadTenderTransferWorkflowResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.ApproveLoadTenderTransferActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("EDI load tender approval workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("EDI load tender approval workflow completed")
	return result, nil
}

func DeliverEDIMessageWorkflow(
	ctx workflow.Context,
	payload *DeliverEDIMessageWorkflowPayload,
) (*DeliverEDIMessageWorkflowResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, deliverMessageActivityOptions)

	var a *Activities
	result := new(DeliverEDIMessageWorkflowResult)
	err := workflow.ExecuteActivity(
		activityCtx,
		a.DeliverEDIMessageActivity,
		payload,
	).Get(activityCtx, result)
	if err == nil {
		workflow.GetLogger(ctx).Info("EDI message delivery workflow completed")
		return result, nil
	}

	workflow.GetLogger(ctx).Error("EDI message delivery exhausted retries", "error", err)
	deadLetterCtx := workflow.WithActivityOptions(ctx, deadLetterActivityOptions)
	deadLetterPayload := &MarkEDIMessageDeadLetteredPayload{
		MessageID:  payload.MessageID,
		TenantInfo: payload.TenantInfo,
		Reason:     err.Error(),
	}
	if deadLetterErr := workflow.ExecuteActivity(
		deadLetterCtx,
		a.MarkEDIMessageDeadLetteredActivity,
		deadLetterPayload,
	).Get(deadLetterCtx, nil); deadLetterErr != nil {
		workflow.GetLogger(ctx).Error(
			"failed to dead-letter EDI message after delivery failure",
			"error", deadLetterErr,
		)
	}
	return nil, err
}
