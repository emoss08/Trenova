package customertrackingjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/jobs/customertrackingjobs/customertrackingcontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	evaluateDebounce         = 10 * time.Second
	evaluateIterationsPerRun = 25
	evaluateActivityTimeout  = 2 * time.Minute
	sweepTenantTimeout       = 15 * time.Minute
)

var evaluateActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: evaluateActivityTimeout,
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

// EvaluateCustomerStatusWorkflow evaluates one shipment shortly after it
// changes. Signals that arrive while it waits or works are folded into the
// next pass, so a burst of stop updates costs one evaluation.
func EvaluateCustomerStatusWorkflow(
	ctx workflow.Context,
	payload *customertrackingcontract.EvaluatePayload,
) (*EvaluateResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, customertrackingcontract.EvaluateSignalName)
	actx := workflow.WithActivityOptions(ctx, evaluateActivityOptions)
	result := &EvaluateResult{}

	for {
		if err := workflow.Sleep(ctx, evaluateDebounce); err != nil {
			return result, err
		}
		drainSignals(signals)

		var evaluated EvaluateResult
		if err := workflow.ExecuteActivity(actx, a.EvaluateShipmentActivity, payload).
			Get(actx, &evaluated); err != nil {
			return result, err
		}
		result.add(&evaluated)

		if !drainSignals(signals) {
			return result, nil
		}
		payload.Iterations++
		if payload.Iterations >= evaluateIterationsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				EvaluateCustomerStatusWorkflow,
				&customertrackingcontract.EvaluatePayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					ShipmentID:     payload.ShipmentID,
				},
			)
		}
	}
}

func drainSignals(signals workflow.ReceiveChannel) bool {
	received := false
	for {
		var signal customertrackingcontract.EvaluateSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}

func CustomerStatusSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, listOptions)
	tenantCtx := workflow.WithActivityOptions(ctx, sweepTenantOptions)

	var tenants []TenantRef
	if err := workflow.ExecuteActivity(listCtx, a.ListSweepTenantsActivity).
		Get(listCtx, &tenants); err != nil {
		return nil, err
	}

	result := &SweepResult{}
	for i := range tenants {
		var swept EvaluateResult
		err := workflow.ExecuteActivity(tenantCtx, a.SweepTenantActivity, &tenants[i]).
			Get(tenantCtx, &swept)
		if err != nil {
			workflow.GetLogger(ctx).Error("customer status sweep failed for organization",
				"organization", tenants[i].OrganizationID.String(),
				"error", err)
			result.FailedOrganizations = append(
				result.FailedOrganizations,
				tenants[i].OrganizationID.String(),
			)
			continue
		}
		result.OrganizationsProcessed++
		result.Totals.add(&swept)
	}
	return result, nil
}

func CustomerTrackingLinkPurgeWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, listOptions)
	var purged int
	if err := workflow.ExecuteActivity(actx, a.PurgeDeadLinksActivity).
		Get(actx, &purged); err != nil {
		return 0, err
	}
	return purged, nil
}
