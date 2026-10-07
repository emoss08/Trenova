package aituneupjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const organizationConcurrency = 4

var computeRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy:         computeRetryPolicy,
}

var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy:         computeRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ComputeAITuneUpsWorkflowName,
			Fn:          ComputeAITuneUpsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Work out each organization's AI tune-ups from the last 30 days of runs",
		},
	}
}

func ComputeAITuneUpsWorkflow(
	ctx workflow.Context,
	input *ComputeAITuneUpsInput,
) (*ComputeAITuneUpsResult, error) {
	if input == nil {
		input = &ComputeAITuneUpsInput{}
	}

	var a *Activities
	result := newComputeAITuneUpsResult()
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "AI tune-ups",
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
				a.ListAITuneUpOrganizationsActivity,
				&ListAITuneUpOrganizationsInput{
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
			var computed OrganizationAITuneUpsResult
			if err := workflow.ExecuteActivity(
				activityCtx,
				a.ComputeOrganizationAITuneUpsActivity,
				&OrganizationAITuneUpsInput{TenantWorkItem: tenant},
			).Get(activityCtx, &computed); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.OrganizationsProcessed++
			result.Suggested += computed.Suggested

			return computed.Suggested, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, ComputeAITuneUpsWorkflow,
			&ComputeAITuneUpsInput{After: next})
	}

	return result, nil
}
