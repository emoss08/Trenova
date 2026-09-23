package briefingjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var briefingRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

// perOrganizationChange marks where the sweep stopped writing every due tenant
// in one activity. A sweep started before it finishes on that activity.
const perOrganizationChange = "daily-briefing-per-organization"

// organizationConcurrency bounds how many organizations are written at once.
// Each gathers a day of figures across eight repositories, so the bound is
// what keeps the hour's sweep off the database the product runs on.
const organizationConcurrency = 4

// briefingActivityOptions serve only sweeps started before the sweep fanned
// out: one activity walking every tenant, generous and heartbeating.
var briefingActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 45 * time.Minute,
	HeartbeatTimeout:    3 * time.Minute,
	RetryPolicy:         briefingRetryPolicy,
}

// dueOptions bound deciding which organizations are due: a control row and a
// cached organization per tenant.
var dueOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         briefingRetryPolicy,
}

// organizationOptions bound one organization's morning: its figures and a
// model call per role. The heartbeat runs on a timer, so a slow model call is
// not mistaken for a lost worker.
var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy:         briefingRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        DailyBriefingWorkflowName,
			Fn:          DailyBriefingWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write the morning briefing for every organization whose hour has come",
		},
		{
			Name:        WriteOrganizationBriefingWorkflowName,
			Fn:          WriteOrganizationBriefingWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write one organization's morning briefing",
		},
		{
			Name:        BriefingRetentionWorkflowName,
			Fn:          BriefingRetentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Remove briefings older than the window a reader can page back through",
		},
	}
}

// DailyBriefingWorkflow writes the morning for every organization whose hour
// has come, one child workflow each.
//
// Which organizations are due is decided up front, as of one instant, so
// every page written in the hour describes the same moment. Each is then its
// own child, keyed for fairness by organization: one tenant's unreadable
// table fails that tenant's child, retried and visible on its own, and costs
// no other tenant its morning.
func DailyBriefingWorkflow(ctx workflow.Context) (*DailyBriefingResult, error) {
	if workflow.GetVersion(ctx, perOrganizationChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return writeInOneActivity(ctx)
	}

	now := workflow.Now(ctx).Unix()

	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, dueOptions)
	var due DueOrganizations
	if err := workflow.ExecuteActivity(listCtx, a.ListDueOrganizationsActivity,
		&DueOrganizationsInput{Now: now},
	).Get(listCtx, &due); err != nil {
		return nil, err
	}

	result := &DailyBriefingResult{
		OrganizationsDue:    len(due.Due),
		FailedOrganizations: append(make([]string, 0, len(due.Failed)), due.Failed...),
	}
	_, _, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Daily briefing",
		Concurrency: organizationConcurrency,
		ListPage: func(workflow.Context, *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			return &temporaljobs.TenantPage{Tenants: due.Due}, nil
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			childCtx := workflow.WithChildOptions(wctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("daily-briefing/%s/%d", tenant.OrganizationID, now),
				TaskQueue:  temporaltype.TaskQueueSystem.String(),
				Priority:   temporal.Priority{FairnessKey: tenant.OrganizationID.String()},
			})

			var written OrganizationBriefingResult
			if err := workflow.ExecuteChildWorkflow(childCtx, WriteOrganizationBriefingWorkflow,
				&OrganizationBriefingInput{TenantWorkItem: tenant, Now: now},
			).Get(childCtx, &written); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations, tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.BriefingsWritten += written.Written
			result.BriefingsNarrated += written.Narrated

			return written.Written, nil
		},
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// WriteOrganizationBriefingWorkflow writes one organization's morning.
func WriteOrganizationBriefingWorkflow(
	ctx workflow.Context,
	input *OrganizationBriefingInput,
) (*OrganizationBriefingResult, error) {
	ctx = workflow.WithActivityOptions(ctx, organizationOptions)

	var a *Activities
	var result OrganizationBriefingResult
	if err := workflow.ExecuteActivity(ctx, a.WriteOrganizationBriefingActivity, input).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// writeInOneActivity is the sweep as it was before it fanned out. It runs only
// sweeps that started on it, and must not change.
func writeInOneActivity(ctx workflow.Context) (*DailyBriefingResult, error) {
	ctx = workflow.WithActivityOptions(ctx, briefingActivityOptions)

	var a *Activities
	result := new(DailyBriefingResult)
	if err := workflow.ExecuteActivity(ctx, a.WriteDueBriefingsActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Daily briefing workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func BriefingRetentionWorkflow(ctx workflow.Context) (*BriefingRetentionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, briefingActivityOptions)

	var a *Activities
	result := new(BriefingRetentionResult)
	if err := workflow.ExecuteActivity(ctx, a.BriefingRetentionActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Briefing retention workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}
