package edijobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/jobs/edijobs/edijobscontract"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
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
			Name:        temporaltype.EDIFlowTimerWorkflowName,
			Fn:          EDIFlowTimerWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Wake an EDI flow instance when its timer is due",
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
) (*edi.LadingReplay, error) {
	ctx = workflow.WithActivityOptions(ctx, replayActivityOptions)
	var a *Activities
	result := new(edi.LadingReplay)
	if err := workflow.ExecuteActivity(ctx, a.RunLadingReplayActivity, payload).
		Get(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Activities) RunLadingReplayActivity(
	ctx context.Context,
	payload *edilading.ReplayWorkflowPayload,
) (*edi.LadingReplay, error) {
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

// EDIFlowTimerWorkflow sleeps until a flow instance is due, wakes it, and sleeps again until it
// is next due; a wake signal moves the time. It ends when the instance has nothing to wait for.
func EDIFlowTimerWorkflow(ctx workflow.Context, input *edijobscontract.FlowTimerInput) error {
	activityCtx := workflow.WithActivityOptions(ctx, flowActivityOptions)
	signals := workflow.GetSignalChannel(ctx, temporaltype.EDIFlowTimerWakeSignal)
	wakeAt := input.WakeAt
	var a *Activities
	for round := 0; round < edijobscontract.RoundsPerRun; round++ {
		for drained := true; drained; {
			drained = signals.ReceiveAsync(&wakeAt)
		}
		if wakeAt <= 0 {
			return nil
		}
		timerCtx, cancel := workflow.WithCancel(ctx)
		wait := time.Unix(wakeAt, 0).Sub(workflow.Now(ctx))
		if wait < 0 {
			wait = 0
		}
		timer := workflow.NewTimer(timerCtx, wait)
		moved := false
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(timer, func(workflow.Future) {})
		selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
			channel.Receive(ctx, &wakeAt)
			moved = true
		})
		selector.Select(ctx)
		cancel()
		if moved {
			continue
		}
		var next int64
		if err := workflow.ExecuteActivity(activityCtx, a.DispatchFlowTimerActivity, input).
			Get(activityCtx, &next); err != nil {
			return err
		}
		wakeAt = next
	}
	return workflow.NewContinueAsNewError(ctx, EDIFlowTimerWorkflow, &edijobscontract.FlowTimerInput{
		TenantInfo: input.TenantInfo,
		InstanceID: input.InstanceID,
		WakeAt:     wakeAt,
	})
}

// DispatchFlowTimerActivity wakes the instance if it is due and returns when it is next due, or
// zero when it waits for nothing.
func (a *Activities) DispatchFlowTimerActivity(
	ctx context.Context,
	input *edijobscontract.FlowTimerInput,
) (int64, error) {
	instance, err := a.ediService.DispatchFlowTimer(ctx, input.TenantInfo, input.InstanceID)
	if err != nil {
		return 0, err
	}
	if instance == nil || instance.WakeAt == nil || instance.Status == edi.FlowInstanceStatusClosed {
		return 0, nil
	}
	return *instance.WakeAt, nil
}

func EDIFlowShipmentEventWorkflow(ctx workflow.Context, input *edijobscontract.FlowEventInput) error {
	ctx = workflow.WithActivityOptions(ctx, flowActivityOptions)
	var a *Activities
	return workflow.ExecuteActivity(ctx, a.NotifyFlowShipmentEventActivity, input).Get(ctx, nil)
}

func (a *Activities) NotifyFlowShipmentEventActivity(
	ctx context.Context,
	input *edijobscontract.FlowEventInput,
) error {
	return a.ediService.NotifyShipmentEvent(ctx, &ediservice.NotifyShipmentEventRequest{
		TenantInfo: input.TenantInfo,
		EventID:    input.EventID,
	})
}
