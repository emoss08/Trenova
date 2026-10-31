package billingqueuejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	recheckDebounce         = 15 * time.Second
	recheckIterationsPerRun = 25
	recheckSweepBatch       = 500
	recheckStaleAfter       = 6 * time.Hour
)

var recheckActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var agentReviewOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

var briefActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

const (
	briefDebounce         = time.Minute
	briefIterationsPerRun = 25
)

var recheckSweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	},
}

type RecheckWorkflowResult struct {
	Rechecks int  `json:"rechecks"`
	Ready    bool `json:"ready"`
}

type RecheckSweepResult struct {
	Requested int `json:"requested"`
}

func BillingQueueRecheckWorkflow(
	ctx workflow.Context,
	payload *billingqueuecontract.RecheckPayload,
) (*RecheckWorkflowResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, billingqueuecontract.RecheckSignalName)
	actx := workflow.WithActivityOptions(ctx, recheckActivityOptions)
	result := &RecheckWorkflowResult{}

	for {
		if err := workflow.Sleep(ctx, recheckDebounce); err != nil {
			return result, err
		}
		drainRecheckSignals(signals)

		var checked billingqueueports.RecheckResult
		if err := workflow.ExecuteActivity(actx, a.RecheckItemActivity, payload).
			Get(actx, &checked); err != nil {
			return result, err
		}
		result.Rechecks++
		result.Ready = checked.Ready
		afterRecheck(ctx, payload, &checked)

		if !drainRecheckSignals(signals) {
			return result, nil
		}
		payload.Iterations++
		if payload.Iterations >= recheckIterationsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				BillingQueueRecheckWorkflow,
				&billingqueuecontract.RecheckPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					ItemID:         payload.ItemID,
				},
			)
		}
	}
}

func drainRecheckSignals(signals workflow.ReceiveChannel) bool {
	received := false
	for {
		var signal billingqueuecontract.RecheckSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}

func BillingQueueRecheckSweepWorkflow(ctx workflow.Context) (*RecheckSweepResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, recheckSweepOptions)
	staleBefore := workflow.Now(ctx).Add(-recheckStaleAfter).Unix()

	var requested int
	if err := workflow.ExecuteActivity(actx, a.RequestStaleRechecksActivity, staleBefore).
		Get(actx, &requested); err != nil {
		return nil, err
	}

	return &RecheckSweepResult{Requested: requested}, nil
}

func afterRecheck(
	ctx workflow.Context,
	payload *billingqueuecontract.RecheckPayload,
	checked *billingqueueports.RecheckResult,
) {
	var a *Activities
	logger := workflow.GetLogger(ctx)

	reviewCtx := workflow.WithActivityOptions(ctx, agentReviewOptions)
	if err := workflow.ExecuteActivity(reviewCtx, a.AgentReviewActivity, payload).
		Get(reviewCtx, nil); err != nil {
		logger.Warn(
			"billing queue agent review failed",
			"item",
			payload.ItemID.String(),
			"error",
			err,
		)
	}

	actx := workflow.WithActivityOptions(ctx, recheckActivityOptions)
	if checked.Ready {
		if err := workflow.ExecuteActivity(actx, a.AutoApproveActivity, payload).
			Get(actx, nil); err != nil {
			logger.Warn(
				"billing queue auto-approve failed",
				"item",
				payload.ItemID.String(),
				"error",
				err,
			)
		}
	}
	if err := workflow.ExecuteActivity(actx, a.RequestBriefActivity, payload).Get(actx, nil); err != nil {
		logger.Warn(
			"billing queue brief request failed",
			"item",
			payload.ItemID.String(),
			"error",
			err,
		)
	}
}

func BillingQueueBriefWorkflow(
	ctx workflow.Context,
	payload *billingqueuecontract.BriefPayload,
) error {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, billingqueuecontract.BriefSignalName)
	actx := workflow.WithActivityOptions(ctx, briefActivityOptions)

	for {
		if err := workflow.Sleep(ctx, briefDebounce); err != nil {
			return err
		}
		drainBriefSignals(signals)

		if err := workflow.ExecuteActivity(actx, a.RefreshBriefActivity, payload).Get(actx, nil); err != nil {
			workflow.GetLogger(ctx).Warn("billing queue brief failed", "error", err)
		}

		if !drainBriefSignals(signals) {
			return nil
		}
		payload.Iterations++
		if payload.Iterations >= briefIterationsPerRun {
			return workflow.NewContinueAsNewError(
				ctx,
				BillingQueueBriefWorkflow,
				&billingqueuecontract.BriefPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
				},
			)
		}
	}
}

func drainBriefSignals(signals workflow.ReceiveChannel) bool {
	received := false
	for {
		var signal billingqueuecontract.BriefSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}
