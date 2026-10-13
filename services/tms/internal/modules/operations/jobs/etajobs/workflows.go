package etajobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs/etacontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	refreshDebounce         = 10 * time.Second
	refreshIterationsPerRun = 25
	sweepTenantTimeout      = 10 * time.Minute
	refreshActivityTimeout  = 2 * time.Minute
)

var refreshActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: refreshActivityTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var sweepListOptions = workflow.ActivityOptions{
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

func RefreshShipmentEtaWorkflow(
	ctx workflow.Context,
	payload *etacontract.RefreshPayload,
) (*RefreshResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, etacontract.RefreshSignalName)
	actx := workflow.WithActivityOptions(ctx, refreshActivityOptions)
	result := &RefreshResult{}

	for {
		if err := workflow.Sleep(ctx, refreshDebounce); err != nil {
			return result, err
		}
		drainSignals(signals)

		var refreshed RefreshResult
		if err := workflow.ExecuteActivity(actx, a.RefreshShipmentActivity, payload).
			Get(actx, &refreshed); err != nil {
			return result, err
		}
		result.Refreshes++
		result.Rows = refreshed.Rows
		result.Attention = refreshed.Attention

		if !drainSignals(signals) {
			return result, nil
		}
		payload.Iterations++
		if payload.Iterations >= refreshIterationsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				RefreshShipmentEtaWorkflow,
				&etacontract.RefreshPayload{
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
		var signal etacontract.RefreshSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}

func ShipmentEtaSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, sweepListOptions)
	tenantCtx := workflow.WithActivityOptions(ctx, sweepTenantOptions)

	var tenants []TenantRef
	if err := workflow.ExecuteActivity(listCtx, a.ListSweepTenantsActivity).
		Get(listCtx, &tenants); err != nil {
		return nil, err
	}

	result := &SweepResult{}
	for i := range tenants {
		var swept RefreshResult
		err := workflow.ExecuteActivity(tenantCtx, a.SweepTenantActivity, &tenants[i]).
			Get(tenantCtx, &swept)
		if err != nil {
			workflow.GetLogger(ctx).Error("shipment eta sweep failed for organization",
				"organization", tenants[i].OrganizationID.String(),
				"error", err)
			result.FailedOrganizations = append(
				result.FailedOrganizations,
				tenants[i].OrganizationID.String(),
			)
			continue
		}
		result.OrganizationsProcessed++
		result.Shipments += swept.Shipments
		result.Rows += swept.Rows
		result.Attention += swept.Attention
	}

	return result, nil
}
