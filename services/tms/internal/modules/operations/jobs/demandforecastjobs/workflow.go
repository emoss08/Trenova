package demandforecastjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/demandforecastjobs/demandforecastcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const tenantPageSize = 100

var fitActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    2 * time.Minute,
		MaximumAttempts:    3,
	},
}

type FanOutInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

type ListTenantsPayload struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type TenantPayload struct {
	temporaljobs.TenantWorkItem
}

type FitResult struct {
	Status      string  `json:"status"`
	LoadsTested int64   `json:"loadsTested"`
	Skill       float64 `json:"skill"`
}

type ActivitiesParams struct {
	fx.In

	Forecasts demandforecastports.Service
	Tenants   demandforecastports.TenantLister
	Logger    *zap.Logger
}

type Activities struct {
	forecasts demandforecastports.Service
	tenants   demandforecastports.TenantLister
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		forecasts: p.Forecasts,
		tenants:   p.Tenants,
		l:         p.Logger.Named("temporal.demand-forecast"),
	}
}

func FitDemandForecastsWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities

	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Freight forecast fitting",
		Concurrency: temporaljobs.DefaultTenantDispatchLimit,
		ListPage: func(
			wctx workflow.Context,
			after *temporaljobs.TenantWorkItem,
		) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			var page temporaljobs.TenantPage
			err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, fitActivityOptions),
				a.ListForecastTenantsActivity,
				&ListTenantsPayload{After: after, Limit: tenantPageSize},
			).Get(wctx, &page)
			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var fitted FitResult
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, fitActivityOptions),
				a.FitDemandForecastActivity,
				&TenantPayload{TenantWorkItem: tenant},
			).Get(wctx, &fitted); err != nil {
				return 0, err
			}
			return int(fitted.LoadsTested), nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(
			ctx,
			FitDemandForecastsWorkflow,
			&FanOutInput{After: next},
		)
	}

	return result, nil
}

func FitDemandForecastWorkflow(
	ctx workflow.Context,
	payload *demandforecastcontract.FitPayload,
) (*FitResult, error) {
	var a *Activities
	var fitted FitResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, fitActivityOptions),
		a.FitDemandForecastActivity,
		&TenantPayload{TenantWorkItem: temporaljobs.TenantWorkItem{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
		}},
	).Get(ctx, &fitted)

	return &fitted, err
}

func (a *Activities) ListForecastTenantsActivity(
	ctx context.Context,
	payload *ListTenantsPayload,
) (*temporaljobs.TenantPage, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, tenantPageSize)
	req := &demandforecastports.ListTenantsRequest{Limit: limit}
	if payload.After != nil {
		req.After = payload.After.TenantInfo()
	}

	tenants, err := a.tenants.ListTenants(ctx, req)
	if err != nil {
		return nil, err
	}

	return &temporaljobs.TenantPage{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
		HasMore: len(tenants) == limit,
	}, nil
}

func (a *Activities) FitDemandForecastActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*FitResult, error) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, "fitting")
	}

	quality, err := a.forecasts.Fit(ctx, payload.TenantInfo())
	if err != nil {
		return nil, err
	}
	a.l.Info("freight forecast fitted",
		zap.String("organizationId", payload.OrganizationID.String()),
		zap.String("status", string(quality.Status)),
		zap.Int32("historyWeeks", quality.HistoryWeeks),
		zap.Float64("skill", quality.Skill),
		zap.Float64("intervalCoverage", quality.IntervalCoverage))

	return &FitResult{
		Status:      string(quality.Status),
		LoadsTested: quality.LoadsTested,
		Skill:       quality.Skill,
	}, nil
}
