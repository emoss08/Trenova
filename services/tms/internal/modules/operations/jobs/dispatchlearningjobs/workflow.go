package dispatchlearningjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchlearningservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	LearnDispatchStyleWorkflowName = "LearnDispatchStyleWorkflow"

	tenantPageSize = 100
)

var learnActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
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

type ActivitiesParams struct {
	fx.In

	Learning *dispatchlearningservice.Service
	Logger   *zap.Logger
}

type Activities struct {
	learning *dispatchlearningservice.Service
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		learning: p.Learning,
		l:        p.Logger.Named("temporal.dispatch-learning"),
	}
}

func LearnDispatchStyleWorkflow(
	ctx workflow.Context,
	input *FanOutInput,
) (*temporaljobs.TenantRunResult, error) {
	if input == nil {
		input = &FanOutInput{}
	}
	var a *Activities

	result, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Dispatch style learning",
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
				workflow.WithActivityOptions(wctx, learnActivityOptions),
				a.ListDispatchStyleTenantsActivity,
				&ListTenantsPayload{After: after, Limit: tenantPageSize},
			).Get(wctx, &page)
			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			var learned dispatchlearningports.LearnResult
			if err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(wctx, learnActivityOptions),
				a.LearnDispatchStyleActivity,
				&TenantPayload{TenantWorkItem: tenant},
			).Get(wctx, &learned); err != nil {
				return 0, err
			}
			return learned.Learned, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(
			ctx,
			LearnDispatchStyleWorkflow,
			&FanOutInput{After: next},
		)
	}

	return result, nil
}

func (a *Activities) ListDispatchStyleTenantsActivity(
	ctx context.Context,
	payload *ListTenantsPayload,
) (*temporaljobs.TenantPage, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, tenantPageSize)
	req := &dispatchlearningports.ListTenantsRequest{Limit: limit}
	if payload.After != nil {
		req.After = payload.After.TenantInfo()
	}

	tenants, err := a.learning.ListLearningTenants(ctx, req)
	if err != nil {
		return nil, err
	}

	return &temporaljobs.TenantPage{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
		HasMore: len(tenants) == limit,
	}, nil
}

func (a *Activities) LearnDispatchStyleActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*dispatchlearningports.LearnResult, error) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, "learning")
	}

	result, err := a.learning.Learn(ctx, &dispatchlearningports.LearnRequest{
		TenantInfo: payload.TenantInfo(),
	})
	if err != nil {
		return nil, err
	}
	if result.Learned > 0 {
		a.l.Info("dispatch style learned",
			zap.String("organizationId", payload.OrganizationID.String()),
			zap.Int("decisions", result.Learned),
			zap.Int("overrides", result.Overrides),
			zap.Int("declines", result.Declines))
	}

	return result, nil
}
