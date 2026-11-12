package etajobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs/etacontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	drainDebounce         = 10 * time.Second
	drainIterationsPerRun = 25
	drainPagesPerRound    = 20
	// drainPageSize is the shipments one activity projects, in one batched
	// refresh.
	drainPageSize        = 50
	drainRetryAfter      = time.Minute
	drainWakeLimit       = 1000
	sweepTenantTimeout   = 10 * time.Minute
	drainActivityTimeout = 5 * time.Minute
)

var drainActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: drainActivityTimeout,
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

// ShipmentEtaDrainWorkflow projects the arrivals of a tenant's queued
// shipments shortly after they change, a page at a time: one workflow per
// tenant, however many shipments move.
func ShipmentEtaDrainWorkflow(
	ctx workflow.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainTotals, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, drainActivityOptions)
	totals, again, err := temporaljobs.RunTenantDrain(ctx, temporaljobs.TenantDrainOptions{
		SignalName:       etacontract.DrainSignalName,
		Debounce:         drainDebounce,
		PagesPerRound:    drainPagesPerRound,
		IterationsPerRun: drainIterationsPerRun,
		Iterations:       payload.Iterations,
		Page: func(ctx workflow.Context) (*temporaljobs.DrainPage, error) {
			var page temporaljobs.DrainPage
			err := workflow.ExecuteActivity(actx, a.DrainShipmentEtasActivity, payload).Get(actx, &page)

			return &page, err
		},
	})
	if err != nil || !again {
		return totals, err
	}

	return totals, workflow.NewContinueAsNewError(ctx, ShipmentEtaDrainWorkflow,
		&temporaljobs.DrainPayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
		})
}

func ShipmentEtaSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, sweepListOptions)
	tenantCtx := workflow.WithActivityOptions(ctx, sweepTenantOptions)

	// Wake the drains whose wake was lost before refreshing every tenant.
	if err := workflow.ExecuteActivity(listCtx, a.WakeDueEtaDrainsActivity).Get(listCtx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("could not wake the shipment eta drains", "error", err)
	}

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
