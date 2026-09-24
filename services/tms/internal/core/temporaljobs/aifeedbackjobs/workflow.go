package aifeedbackjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const organizationConcurrency = 4

var maintenanceRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy:         maintenanceRetryPolicy,
}

var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy:         maintenanceRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AIFeedbackMaintenanceWorkflowName,
			Fn:          AIFeedbackMaintenanceWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Purge expired AI feedback and suggest agent memories from it, per organization",
		},
	}
}

func AIFeedbackMaintenanceWorkflow(
	ctx workflow.Context,
	input *AIFeedbackMaintenanceInput,
) (*AIFeedbackMaintenanceResult, error) {
	if input == nil {
		input = &AIFeedbackMaintenanceInput{}
	}
	now := input.Now
	if now == 0 {
		now = workflow.Now(ctx).Unix()
	}

	var a *Activities
	result := newAIFeedbackMaintenanceResult()
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "AI feedback maintenance",
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
				a.ListAIFeedbackOrganizationsActivity,
				&ListAIFeedbackOrganizationsInput{
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
			var maintained OrganizationAIFeedbackResult
			if err := workflow.ExecuteActivity(
				activityCtx,
				a.MaintainOrganizationAIFeedbackActivity,
				&OrganizationAIFeedbackInput{TenantWorkItem: tenant, Now: now},
			).Get(activityCtx, &maintained); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.absorb(&maintained)

			return maintained.Suggested, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, AIFeedbackMaintenanceWorkflow,
			&AIFeedbackMaintenanceInput{After: next, Now: now})
	}

	return result, nil
}
