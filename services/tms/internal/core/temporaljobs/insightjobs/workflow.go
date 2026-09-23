package insightjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// perOrganizationChange marks where the sweep stopped walking every tenant
// in one activity. A sweep started before it finishes on that activity.
const perOrganizationChange = "insight-refresh-per-organization"

// organizationConcurrency bounds how many organizations refresh at once. Each
// runs month-wide aggregates, so the bound is what keeps a sweep from
// saturating the database the product runs on.
const organizationConcurrency = 4

var insightRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

// The timeout is generous because the sweep walks every organization and each
// one runs several month-wide aggregates plus a model call. It serves only
// sweeps started before the sweep fanned out.
var insightActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         insightRetryPolicy,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy:         insightRetryPolicy,
}

// organizationOptions bound one organization's refresh: its aggregates and a
// model call per batch of findings. The heartbeat runs on a timer, so a slow
// model call is not mistaken for a lost worker.
var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy:         insightRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        InsightRefreshWorkflowName,
			Fn:          InsightRefreshWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Recompute operational insights for every organization",
		},
		{
			Name:        RefreshOrganizationInsightsWorkflowName,
			Fn:          RefreshOrganizationInsightsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Recompute one organization's operational insights",
		},
	}
}

// InsightRefreshWorkflow recomputes insights for every organization, one child
// workflow each.
//
// A child per organization means one tenant's failure is that child's, retried
// on its own and visible by name in the Temporal UI, rather than a line in a
// thirty-minute activity. The children are keyed for fairness by
// organization, and the sweep is anchored to one instant, carried across
// continue-as-new, so every tenant describes the same window.
func InsightRefreshWorkflow(
	ctx workflow.Context,
	input *InsightRefreshInput,
) (*InsightRefreshResult, error) {
	if workflow.GetVersion(ctx, perOrganizationChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return refreshInOneActivity(ctx)
	}

	if input == nil {
		input = &InsightRefreshInput{}
	}
	now := input.Now
	if now == 0 {
		now = workflow.Now(ctx).Unix()
	}

	var a *Activities
	result := newInsightRefreshResult()
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Insight refresh",
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
				a.ListOrganizationsActivity,
				&ListOrganizationsInput{
					After: after,
					Limit: temporaljobs.DefaultOrganizationPageSize,
				},
			).Get(listCtx, &page)

			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			childCtx := workflow.WithChildOptions(wctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("insight-refresh/%s/%d", tenant.OrganizationID, now),
				TaskQueue:  temporaltype.TaskQueueSystem.String(),
				Priority:   temporal.Priority{FairnessKey: tenant.OrganizationID.String()},
			})

			var refreshed OrganizationInsightsResult
			if err := workflow.ExecuteChildWorkflow(childCtx, RefreshOrganizationInsightsWorkflow,
				&OrganizationInsightsInput{TenantWorkItem: tenant, Now: now},
			).Get(childCtx, &refreshed); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations, tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.absorb(&refreshed)

			return refreshed.Created, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, InsightRefreshWorkflow,
			&InsightRefreshInput{After: next, Now: now})
	}

	return result, nil
}

// RefreshOrganizationInsightsWorkflow recomputes one organization's insights.
func RefreshOrganizationInsightsWorkflow(
	ctx workflow.Context,
	input *OrganizationInsightsInput,
) (*OrganizationInsightsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, organizationOptions)

	var a *Activities
	var result OrganizationInsightsResult
	if err := workflow.ExecuteActivity(ctx, a.RefreshOrganizationInsightsActivity, input).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// refreshInOneActivity is the sweep as it was before it fanned out. It runs
// only sweeps that started on it, and must not change.
func refreshInOneActivity(ctx workflow.Context) (*InsightRefreshResult, error) {
	ctx = workflow.WithActivityOptions(ctx, insightActivityOptions)

	var a *Activities
	result := new(InsightRefreshResult)
	if err := workflow.ExecuteActivity(ctx, a.RefreshInsightsActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Insight refresh workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}
