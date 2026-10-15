package factoringjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	evaluateDebounce         = 10 * time.Second
	evaluateIterationsPerRun = 25
	evaluateActivityTimeout  = 10 * time.Minute
	sweepTenantTimeout       = 30 * time.Minute
)

var evaluateActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: evaluateActivityTimeout,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	},
}

var sweepTenantOptions = workflow.ActivityOptions{
	StartToCloseTimeout: sweepTenantTimeout,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

func EvaluateFactoringReadinessWorkflow(
	ctx workflow.Context,
	payload *factoringcontract.EvaluatePayload,
) (*factoringports.EvaluateResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, factoringcontract.EvaluateSignalName)
	actx := workflow.WithActivityOptions(ctx, evaluateActivityOptions)
	result := &factoringports.EvaluateResult{}

	for {
		if err := workflow.Sleep(ctx, evaluateDebounce); err != nil {
			return result, err
		}
		drainSignals(signals, payload)
		if payload.HasWork() {
			batch := payload.Take()
			var evaluated factoringports.EvaluateResult
			if err := workflow.ExecuteActivity(actx, a.EvaluateFactoringTargetsActivity, batch).
				Get(actx, &evaluated); err != nil {
				return result, err
			}
			result.Add(&evaluated)
		}

		if !drainSignals(signals, payload) && !payload.HasWork() {
			return result, nil
		}
		payload.Iterations++
		if payload.Iterations >= evaluateIterationsPerRun {
			next := payload.Take()
			return result, workflow.NewContinueAsNewError(
				ctx,
				EvaluateFactoringReadinessWorkflow,
				next,
			)
		}
	}
}

func drainSignals(
	signals workflow.ReceiveChannel,
	payload *factoringcontract.EvaluatePayload,
) bool {
	received := false
	for {
		var signal factoringcontract.EvaluateSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		payload.Merge(&signal.Targets)
		received = true
	}
}

func FactoringReadinessSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, listOptions)
	tenantCtx := workflow.WithActivityOptions(ctx, sweepTenantOptions)

	var tenants []TenantRef
	if err := workflow.ExecuteActivity(listCtx, a.ListFactoringSweepTenantsActivity).
		Get(listCtx, &tenants); err != nil {
		return nil, err
	}

	result := &SweepResult{}
	for i := range tenants {
		var swept factoringports.SweepResult
		err := workflow.ExecuteActivity(tenantCtx, a.SweepFactoringTenantActivity, &tenants[i]).
			Get(tenantCtx, &swept)
		if err != nil {
			workflow.GetLogger(ctx).Error("factoring readiness sweep failed for organization",
				"organization", tenants[i].OrganizationID.String(),
				"error", err)
			result.FailedOrganizations = append(
				result.FailedOrganizations,
				tenants[i].OrganizationID.String(),
			)
			continue
		}
		result.OrganizationsProcessed++
		result.Totals.Add(&swept.EvaluateResult)
		result.NoasWarned += swept.NoasWarned
		result.ChargebacksWarned += swept.ChargebacksWarned
	}
	return result, nil
}
