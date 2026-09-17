package carrierintelligencejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	SweepWorkflowName       = "CarrierIntelSweepWorkflow"
	ReconcileWorkflowName   = "CarrierIntelReconcileWorkflow"
	DigestWorkflowName      = "CarrierIntelDigestWorkflow"
	MaintenanceWorkflowName = "CarrierIntelMaintenanceWorkflow"
)

var nonRetryableTypes = []string{"BusinessError", "ValidationError", "AuthorizationError"}

var sweepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        10 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        2 * time.Minute,
		MaximumAttempts:        3,
		NonRetryableErrorTypes: nonRetryableTypes,
	},
}

var shortActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        5 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        time.Minute,
		MaximumAttempts:        3,
		NonRetryableErrorTypes: nonRetryableTypes,
	},
}

type FanOutInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SweepWorkflowName,
			Fn:          CarrierIntelSweepWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Recompute findings, sync monitoring watchlists, poll change feeds and refresh due carrier intelligence",
		},
		{
			Name:        ReconcileWorkflowName,
			Fn:          CarrierIntelReconcileWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Reconcile carrier monitoring enrollment with each organization's policy",
		},
		{
			Name:        DigestWorkflowName,
			Fn:          CarrierIntelDigestWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Send the daily carrier intelligence change digest",
		},
		{
			Name:        MaintenanceWorkflowName,
			Fn:          CarrierIntelMaintenanceWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Roll up provider usage, purge expired payloads and prune snapshot history",
		},
	}
}

func listPage(
	ctx workflow.Context,
	after *temporaljobs.TenantWorkItem,
) (*temporaljobs.TenantPage, error) {
	var a *Activities
	var page temporaljobs.TenantPage
	listCtx := workflow.WithActivityOptions(ctx, shortActivityOptions)
	err := workflow.ExecuteActivity(listCtx, a.ListTenantsActivity, &ListTenantsPayload{
		After: after,
		Limit: tenantPageSize,
	}).Get(ctx, &page)
	return &page, err
}

func continueIfNeeded(
	ctx workflow.Context,
	fn any,
	result *temporaljobs.TenantRunResult,
	next *temporaljobs.TenantWorkItem,
	err error,
) (*temporaljobs.TenantRunResult, error) {
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, fn, &FanOutInput{After: next})
	}
	return result, nil
}

func CarrierIntelSweepWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities
	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Carrier intelligence sweep",
		Concurrency: temporaljobs.DefaultTenantDispatchLimit,
		ListPage: func(wctx workflow.Context, after *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			return listPage(wctx, after)
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var sweep SweepTenantResult
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, sweepActivityOptions),
				a.SweepTenantActivity,
				&TenantPayload{TenantWorkItem: tenant},
			).Get(wctx, &sweep); err != nil {
				return 0, err
			}
			return sweep.Total(), nil
		},
	})
	return continueIfNeeded(ctx, CarrierIntelSweepWorkflow, result, next, err)
}

func CarrierIntelReconcileWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities
	driftCheck := workflow.Now(ctx).UTC().Weekday() == sundayDriftCheckDay
	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Carrier intelligence reconcile",
		Concurrency: temporaljobs.DefaultTenantDispatchLimit,
		ListPage: func(wctx workflow.Context, after *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			return listPage(wctx, after)
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var reconcile carrierintelservice.ReconcileResult
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, sweepActivityOptions),
				a.ReconcileTenantActivity,
				&ReconcileTenantPayload{TenantWorkItem: tenant, DriftCheck: driftCheck},
			).Get(wctx, &reconcile); err != nil {
				return 0, err
			}
			return reconcile.Desired, nil
		},
	})
	return continueIfNeeded(ctx, CarrierIntelReconcileWorkflow, result, next, err)
}

func CarrierIntelDigestWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities
	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Carrier intelligence digest",
		Concurrency: temporaljobs.DefaultTenantDispatchLimit,
		ListPage: func(wctx workflow.Context, after *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			return listPage(wctx, after)
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var digest carrierintelservice.DigestResult
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, shortActivityOptions),
				a.DigestTenantActivity,
				&TenantPayload{TenantWorkItem: tenant},
			).Get(wctx, &digest); err != nil {
				return 0, err
			}
			return digest.Events, nil
		},
	})
	return continueIfNeeded(ctx, CarrierIntelDigestWorkflow, result, next, err)
}

func CarrierIntelMaintenanceWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities
	if input.After == nil {
		var maintenance carrierintelservice.MaintenanceResult
		if err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, sweepActivityOptions),
			a.GlobalMaintenanceActivity,
		).Get(ctx, &maintenance); err != nil {
			workflow.GetLogger(ctx).
				Error("Carrier intelligence global maintenance failed", "error", err)
		}
	}

	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Carrier intelligence prune",
		Concurrency: temporaljobs.DefaultTenantDispatchLimit,
		ListPage: func(wctx workflow.Context, after *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			return listPage(wctx, after)
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var pruned int
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, shortActivityOptions),
				a.PruneTenantActivity,
				&TenantPayload{TenantWorkItem: tenant},
			).Get(wctx, &pruned); err != nil {
				return 0, err
			}
			return pruned, nil
		},
	})
	return continueIfNeeded(ctx, CarrierIntelMaintenanceWorkflow, result, next, err)
}
