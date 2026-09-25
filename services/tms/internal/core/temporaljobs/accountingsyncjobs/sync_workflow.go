package accountingsyncjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var drainActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

var sweepLongActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    2 * time.Minute,
	},
}

func syncWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        DrainAccountingOutboxWorkflowName,
			Fn:          DrainAccountingOutboxWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Send one connection's queued documents to its accounting system",
		},
		{
			Name:        KickDueAccountingSyncWorkflowName,
			Fn:          KickDueAccountingSyncWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Wake the sender for every accounting connection with documents due",
		},
		{
			Name:        AccountingSafetyNetWorkflowName,
			Fn:          AccountingSafetyNetWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Queue any posted document that never reached the accounting outbox",
		},
		{
			Name:        PurgeAccountingSyncWorkflowName,
			Fn:          PurgeAccountingSyncWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Clear old accounting sync payloads and attempts",
		},
		{
			Name:        BackfillAccountingWorkflowName,
			Fn:          BackfillAccountingWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Queue documents dated from the start date until sync began",
		},
	}
}

func DrainAccountingOutboxWorkflow(
	ctx workflow.Context,
	payload *DrainPayload,
) (*DrainRunResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, DrainSignalName)
	actx := workflow.WithActivityOptions(
		ctx,
		withFairness(&drainActivityOptions, payload.OrganizationID),
	)
	result := &DrainRunResult{Batches: payload.Batches}

	for {
		drainSignals(signals)
		if result.Batches >= drainBatchesPerRun {
			return result, workflow.NewContinueAsNewError(ctx, DrainAccountingOutboxWorkflow, &DrainPayload{
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
				ConnectionID:   payload.ConnectionID,
			})
		}

		var batch DrainBatchResult
		if err := workflow.ExecuteActivity(actx, a.DrainAccountingOutboxActivity, payload).
			Get(actx, &batch); err != nil {
			return result, err
		}
		result.Batches++
		result.Claimed += batch.Claimed
		result.Synced += batch.Synced
		if batch.Held {
			result.Held = true
			return result, nil
		}
		if batch.Claimed >= drainBatchLimit {
			continue
		}
		if !waitForKick(ctx, signals) {
			return result, nil
		}
	}
}

func drainSignals(signals workflow.ReceiveChannel) {
	for {
		var signal DrainSignal
		if !signals.ReceiveAsync(&signal) {
			return
		}
	}
}

func waitForKick(ctx workflow.Context, signals workflow.ReceiveChannel) bool {
	timerCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	received := false
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
		var signal DrainSignal
		channel.Receive(ctx, &signal)
		received = true
	})
	selector.AddFuture(workflow.NewTimer(timerCtx, drainIdleWait), func(workflow.Future) {})
	selector.Select(ctx)
	return received
}

func KickDueAccountingSyncWorkflow(ctx workflow.Context) (*KickDueResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepActivityOptions)
	result := new(KickDueResult)
	err := workflow.ExecuteActivity(actx, a.KickDueAccountingSyncActivity).Get(actx, result)
	return result, err
}

func AccountingSafetyNetWorkflow(ctx workflow.Context) (*SafetyNetSweepResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepLongActivityOptions)
	result := new(SafetyNetSweepResult)
	err := workflow.ExecuteActivity(actx, a.AccountingSafetyNetActivity).Get(actx, result)
	return result, err
}

func PurgeAccountingSyncWorkflow(ctx workflow.Context) (*PurgeResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepLongActivityOptions)
	result := new(PurgeResult)
	err := workflow.ExecuteActivity(actx, a.PurgeAccountingSyncHistoryActivity).Get(actx, result)
	return result, err
}

func BackfillAccountingWorkflow(
	ctx workflow.Context,
	payload *BackfillPayload,
) (*BackfillRunResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(
		ctx,
		withFairness(&markActivityOptions, payload.OrganizationID),
	)
	result := &BackfillRunResult{Steps: payload.Steps}

	for {
		if result.Steps-payload.Steps >= backfillStepsPerRun {
			return result, workflow.NewContinueAsNewError(ctx, BackfillAccountingWorkflow, &BackfillPayload{
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
				BackfillID:     payload.BackfillID,
				Steps:          result.Steps,
			})
		}

		var step BackfillStepResult
		if err := workflow.ExecuteActivity(actx, a.AccountingBackfillStepActivity, payload).
			Get(actx, &step); err != nil {
			return result, err
		}
		result.Steps++
		result.Enqueued += step.Enqueued
		if step.Done || step.Stopped {
			result.Done = step.Done
			return result, nil
		}
	}
}
