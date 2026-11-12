package customertrackingjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/customertrackingjobs/customertrackingcontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	drainDebounce         = 10 * time.Second
	drainIterationsPerRun = 25
	drainPagesPerRound    = 20
	// drainPageSize is the shipments one activity evaluates; each may send
	// an email, so the page stays well inside the activity's timeout.
	drainPageSize        = 25
	drainRetryAfter      = time.Minute
	drainWakeLimit       = 1000
	drainActivityTimeout = 5 * time.Minute
	sweepTenantTimeout   = 15 * time.Minute
)

var drainActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: drainActivityTimeout,
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

// CustomerStatusDrainWorkflow evaluates a tenant's queued shipments shortly
// after they change, a page at a time: a burst of stop updates across many
// shipments costs one workflow and a few activities.
func CustomerStatusDrainWorkflow(
	ctx workflow.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainTotals, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, drainActivityOptions)
	totals, again, err := temporaljobs.RunTenantDrain(ctx, temporaljobs.TenantDrainOptions{
		SignalName:       customertrackingcontract.DrainSignalName,
		Debounce:         drainDebounce,
		PagesPerRound:    drainPagesPerRound,
		IterationsPerRun: drainIterationsPerRun,
		Iterations:       payload.Iterations,
		Page: func(ctx workflow.Context) (*temporaljobs.DrainPage, error) {
			var page temporaljobs.DrainPage
			err := workflow.ExecuteActivity(actx, a.DrainCustomerStatusActivity, payload).
				Get(actx, &page)

			return &page, err
		},
	})
	if err != nil || !again {
		return totals, err
	}

	return totals, workflow.NewContinueAsNewError(ctx, CustomerStatusDrainWorkflow,
		&temporaljobs.DrainPayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
		})
}

func CustomerStatusSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, listOptions)
	tenantCtx := workflow.WithActivityOptions(ctx, sweepTenantOptions)

	// Wake the drains whose wake was lost before sweeping every tenant.
	if err := workflow.ExecuteActivity(listCtx, a.WakeDueCustomerStatusDrainsActivity).
		Get(listCtx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("could not wake the customer status drains", "error", err)
	}

	var tenants []TenantRef
	if err := workflow.ExecuteActivity(listCtx, a.ListCustomerStatusSweepTenantsActivity).
		Get(listCtx, &tenants); err != nil {
		return nil, err
	}

	result := &SweepResult{}
	for i := range tenants {
		var swept EvaluateResult
		err := workflow.ExecuteActivity(tenantCtx, a.SweepCustomerStatusTenantActivity, &tenants[i]).
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
	if err := workflow.ExecuteActivity(actx, a.PurgeDeadCustomerTrackingLinksActivity).
		Get(actx, &purged); err != nil {
		return 0, err
	}
	return purged, nil
}
