package temporaljobs

import (
	"go.temporal.io/sdk/workflow"
)

type TenantPage struct {
	Tenants []TenantWorkItem `json:"tenants"`
	HasMore bool             `json:"hasMore"`
}

type FanOutOptions struct {
	Label       string
	Concurrency int
	ListPage    func(ctx workflow.Context, after *TenantWorkItem) (*TenantPage, error)
	RunTenant   func(ctx workflow.Context, tenant TenantWorkItem) (int, error)
	MaxPages    int
}

func RunTenantFanOut(
	ctx workflow.Context,
	opts FanOutOptions,
) (*TenantRunResult, *TenantWorkItem, error) {
	logger := workflow.GetLogger(ctx)
	concurrency := NormalizeLimit(opts.Concurrency, DefaultTenantDispatchLimit)
	maxPages := NormalizeLimit(opts.MaxPages, 5)
	result := new(TenantRunResult)

	var after *TenantWorkItem
	for range maxPages {
		page, err := opts.ListPage(ctx, after)
		if err != nil {
			return result, after, err
		}
		if len(page.Tenants) == 0 {
			return result, nil, nil
		}
		result.TenantsScanned += len(page.Tenants)

		selector := workflow.NewSelector(ctx)
		inFlight := 0
		for _, tenant := range page.Tenants {
			for inFlight >= concurrency {
				selector.Select(ctx)
				inFlight--
			}
			future, settable := workflow.NewFuture(ctx)
			workflow.Go(ctx, func(gctx workflow.Context) {
				processed, runErr := opts.RunTenant(gctx, tenant)
				settable.Set(processed, runErr)
			})
			inFlight++
			selector.AddFuture(future, func(f workflow.Future) {
				var processed int
				if getErr := f.Get(ctx, &processed); getErr != nil {
					logger.Error(opts.Label+" tenant failed",
						"orgId", tenant.OrganizationID.String(),
						"buId", tenant.BusinessUnitID.String(),
						"error", getErr,
					)
					result.AddFailure(tenant, getErr)
					return
				}
				result.AddTenantResult(processed, 0)
			})
		}
		for inFlight > 0 {
			selector.Select(ctx)
			inFlight--
		}

		last := page.Tenants[len(page.Tenants)-1]
		after = &last
		if !page.HasMore {
			return result, nil, nil
		}
	}

	logger.Info(opts.Label+" fan-out paused for continue-as-new",
		"tenantsScanned", result.TenantsScanned)
	return result, after, nil
}
