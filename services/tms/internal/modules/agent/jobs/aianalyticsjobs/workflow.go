package aianalyticsjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const organizationConcurrency = 4

var refreshRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy:         refreshRetryPolicy,
}

var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy:         refreshRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RefreshAIAnalyticsWorkflowName,
			Fn:          RefreshAIAnalyticsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Work out each organization's AI analytics from its newest runs, turns and calls",
		},
	}
}

func RefreshAIAnalyticsWorkflow(
	ctx workflow.Context,
	input *RefreshAIAnalyticsInput,
) (*RefreshAIAnalyticsResult, error) {
	if input == nil {
		input = &RefreshAIAnalyticsInput{}
	}

	var a *Activities
	result := newRefreshAIAnalyticsResult()
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "AI analytics",
		Concurrency: organizationConcurrency,
		ListPage: func(
			wctx workflow.Context,
			after *temporaljobs.TenantWorkItem,
		) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			listCtx := workflow.WithActivityOptions(wctx, listOptions)
			var page temporaljobs.TenantPage
			err := workflow.ExecuteActivity(
				listCtx,
				a.ListAIAnalyticsOrganizationsActivity,
				&ListAIAnalyticsOrganizationsInput{
					After: after,
					Limit: temporaljobs.DefaultOrganizationPageSize,
				},
			).Get(listCtx, &page)

			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			options := organizationOptions
			options.Priority = temporal.Priority{FairnessKey: tenant.OrganizationID.String()}
			activityCtx := workflow.WithActivityOptions(wctx, options)
			var refreshed OrganizationAIAnalyticsResult
			if err := workflow.ExecuteActivity(
				activityCtx,
				a.RefreshOrganizationAIAnalyticsActivity,
				&OrganizationAIAnalyticsInput{TenantWorkItem: tenant},
			).Get(activityCtx, &refreshed); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.OrganizationsProcessed++
			result.Units += refreshed.Units

			return refreshed.Units, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, RefreshAIAnalyticsWorkflow,
			&RefreshAIAnalyticsInput{After: next})
	}

	return result, nil
}
