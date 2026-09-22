package inboundjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// settleOptions allow for a model call and a handful of lookups. The timeout is
// generous because a provider under load is slow rather than broken, and the
// retry is bounded because a message that will not classify after four tries
// wants a person, not a fifth.
var settleOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    4,
	},
}

// failOptions are separate and tighter. Recording a failure is a single write,
// and retrying it for minutes would leave the message looking unprocessed for
// exactly as long as the thing that is trying to say it failed.
var failOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: 2 * time.Second,
		MaximumAttempts: 3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.ProcessInboundMessageWorkflowName,
			Fn:          ProcessInboundMessageWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Reads a staged inbound message and decides what it is.",
		},
	}
}

// ProcessInboundMessageWorkflow settles one message.
//
// When settling fails for good, the workflow does not simply end: it records
// the failure on the message so the inbox shows it waiting on a person. A
// message left at Received looks like one still being worked on, and that is
// the state nobody goes and checks.
func ProcessInboundMessageWorkflow(
	ctx workflow.Context,
	payload *ProcessInboundMessagePayload,
) (*ProcessInboundMessageResult, error) {
	var a *Activities

	settleCtx := workflow.WithActivityOptions(ctx, settleOptions)

	result := new(ProcessInboundMessageResult)
	err := workflow.ExecuteActivity(settleCtx, a.SettleInboundMessageActivity, payload).
		Get(settleCtx, result)
	if err == nil {
		return result, nil
	}

	workflow.GetLogger(ctx).Error("inbound message could not be settled",
		"messageId", payload.MessageID.String(), "error", err)

	failCtx := workflow.WithActivityOptions(ctx, failOptions)
	failErr := workflow.ExecuteActivity(failCtx, a.FailInboundMessageActivity,
		&FailInboundMessagePayload{
			BasePayload: payload.BasePayload,
			MessageID:   payload.MessageID,
			Code:        "SETTLE_FAILED",
			Reason:      err.Error(),
		}).Get(failCtx, nil)
	if failErr != nil {
		// Both failed. The original is the one worth reporting: the second is
		// only the attempt to write the first one down.
		workflow.GetLogger(ctx).Error("could not record the failure either",
			"messageId", payload.MessageID.String(), "error", failErr)
	}

	return nil, err
}
