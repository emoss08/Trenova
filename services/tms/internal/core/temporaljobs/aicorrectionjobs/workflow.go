package aicorrectionjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const organizationConcurrency = 4

var retentionRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy:         retentionRetryPolicy,
}

var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy:         retentionRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AICorrectionRetentionWorkflowName,
			Fn:          AICorrectionRetentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Purge AI corrections older than each organization's retention period",
		},
	}
}

func AICorrectionRetentionWorkflow(
	ctx workflow.Context,
	input *AICorrectionRetentionInput,
) (*AICorrectionRetentionResult, error) {
	if input == nil {
		input = &AICorrectionRetentionInput{}
	}
	now := input.Now
	if now == 0 {
		now = workflow.Now(ctx).Unix()
	}

	var a *Activities
	result := newAICorrectionRetentionResult()
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "AI correction retention",
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
				a.ListAICorrectionOrganizationsActivity,
				&ListAICorrectionOrganizationsInput{
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
			var purged OrganizationAICorrectionResult
			if err := workflow.ExecuteActivity(
				activityCtx,
				a.PurgeOrganizationAICorrectionsActivity,
				&OrganizationAICorrectionInput{TenantWorkItem: tenant, Now: now},
			).Get(activityCtx, &purged); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.absorb(&purged)

			return int(purged.Purged), nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, AICorrectionRetentionWorkflow,
			&AICorrectionRetentionInput{After: next, Now: now})
	}

	return result, nil
}
