package edijobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/jobs/edijobs/edijobscontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var replayActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Hour,
	HeartbeatTimeout:    5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    5 * time.Minute,
	},
}

var flowActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    10,
		MaximumInterval:    5 * time.Minute,
	},
}

func ladingWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.RunLadingReplayWorkflowName,
			Fn:          RunLadingReplayWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Replay a Lading version against another on a partner's real messages",
		},
		{
			Name:        DispatchDueFlowTimersWorkflowName,
			Fn:          DispatchDueFlowTimersWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Wake every open EDI flow instance whose timer is due",
		},
		{
			Name:        edijobscontract.FlowEventWorkflowName,
			Fn:          EDIFlowShipmentEventWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Give a shipment event to the EDI flows that follow the shipment",
		},
	}
}

// RunLadingReplayWorkflow runs a replay a person started; the activity heartbeats after every
// batch, so a long replay survives a worker restart from where it was.
func RunLadingReplayWorkflow(
	ctx workflow.Context,
	payload *edilading.ReplayWorkflowPayload,
) (*ladingproject.LadingReplay, error) {
	ctx = workflow.WithActivityOptions(ctx, replayActivityOptions)
	var a *Activities
	result := new(ladingproject.LadingReplay)
	if err := workflow.ExecuteActivity(ctx, a.RunLadingReplayActivity, payload).
		Get(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Activities) RunLadingReplayActivity(
	ctx context.Context,
	payload *edilading.ReplayWorkflowPayload,
) (*ladingproject.LadingReplay, error) {
	replay, err := a.replayer.Run(ctx, payload, func(done int) {
		activity.RecordHeartbeat(ctx, done)
	})
	if err != nil {
		a.logger.Error("Lading replay failed",
			zap.String("replayId", payload.ReplayID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return replay, nil
}

// DispatchDueFlowTimersWorkflowName is the minute sweep behind every flow
// timer: an instance holds no workflow while it waits.
const DispatchDueFlowTimersWorkflowName = "DispatchDueFlowTimersWorkflow"

var dueFlowTimerOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    30 * time.Second,
	},
}

func DispatchDueFlowTimersWorkflow(ctx workflow.Context) (int, error) {
	ctx = workflow.WithActivityOptions(ctx, dueFlowTimerOptions)
	var a *Activities

	var dispatched int
	err := workflow.ExecuteActivity(ctx, a.DispatchDueFlowTimersActivity).Get(ctx, &dispatched)

	return dispatched, err
}

// DispatchDueFlowTimersActivity wakes every open flow instance whose timer is
// due, across tenants.
func (a *Activities) DispatchDueFlowTimersActivity(ctx context.Context) (int, error) {
	return a.outboundService.DispatchDueFlowTimers(ctx, time.Now().Unix())
}

func EDIFlowShipmentEventWorkflow(
	ctx workflow.Context,
	input *edijobscontract.FlowEventInput,
) error {
	ctx = workflow.WithActivityOptions(ctx, flowActivityOptions)
	var a *Activities
	return workflow.ExecuteActivity(ctx, a.NotifyFlowShipmentEventActivity, input).Get(ctx, nil)
}

func (a *Activities) NotifyFlowShipmentEventActivity(
	ctx context.Context,
	input *edijobscontract.FlowEventInput,
) error {
	return a.outboundService.NotifyShipmentEvent(ctx, &edioutbound.NotifyShipmentEventRequest{
		TenantInfo: input.TenantInfo,
		EventID:    input.EventID,
	})
}
