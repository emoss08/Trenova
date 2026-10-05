package cloudlifecyclejobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/cloud/lifecycle/cloudlifecycleservice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var lifecycleRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    5 * time.Minute,
	MaximumAttempts:    5,
}

var sweepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         lifecycleRetryPolicy,
}

var checkActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         lifecycleRetryPolicy,
}

var purgeRowsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Hour,
	HeartbeatTimeout:    5 * time.Minute,
	RetryPolicy:         lifecycleRetryPolicy,
}

var purgeStepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Hour,
	HeartbeatTimeout:    5 * time.Minute,
	RetryPolicy:         lifecycleRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        CloudSubscriptionSweepWorkflowName,
			Fn:          CloudSubscriptionSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Move cloud trials to read-only and expired, and start the purge of expired organizations",
		},
		{
			Name:        CloudTenantPurgeWorkflowName,
			Fn:          CloudTenantPurgeWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Delete every row, stored object and user an expired cloud organization owns",
		},
	}
}

func CloudSubscriptionSweepWorkflow(ctx workflow.Context, input *SweepInput) (*SweepResult, error) {
	now := workflow.Now(ctx).UTC()
	if input == nil {
		input = &SweepInput{}
	}
	if input.Now <= 0 {
		input.Now = now.Unix()
	}

	var a *Activities
	sweep := new(cloudlifecycleservice.SweepResult)
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, sweepActivityOptions),
		a.SweepCloudSubscriptionsActivity,
		input,
	).Get(ctx, sweep); err != nil {
		return nil, err
	}

	result := &SweepResult{SweepResult: *sweep}
	day := now.Format("20060102")
	for _, target := range sweep.PurgeTargets {
		started, err := startPurge(ctx, target, day)
		if err != nil {
			workflow.GetLogger(ctx).Error("failed to start a cloud tenant purge",
				"organizationId", target.OrganizationID.String(),
				"error", err,
			)
			result.Failed++
			result.Failures = append(result.Failures, target.OrganizationID.String())
			continue
		}
		if started {
			result.PurgesStarted++
		} else {
			result.PurgesRunning++
		}
	}

	return result, nil
}

func PurgeWorkflowID(organizationID pulid.ID, day string) string {
	return fmt.Sprintf("%s/%s/%s", purgeWorkflowIDPrefix, organizationID, day)
}

func startPurge(
	ctx workflow.Context,
	target cloudlifecycleservice.TenantRef,
	day string,
) (bool, error) {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:            PurgeWorkflowID(target.OrganizationID, day),
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		StaticSummary:         "Purge expired cloud organization " + target.OrganizationID.String(),
	})

	future := workflow.ExecuteChildWorkflow(childCtx, CloudTenantPurgeWorkflow, &PurgePayload{
		OrganizationID: target.OrganizationID,
		BusinessUnitID: target.BusinessUnitID,
	})

	var execution workflow.Execution
	if err := future.GetChildWorkflowExecution().Get(ctx, &execution); err != nil {
		if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func CloudTenantPurgeWorkflow(ctx workflow.Context, payload *PurgePayload) (*PurgeResult, error) {
	var a *Activities
	result := &PurgeResult{OrganizationID: payload.OrganizationID}

	eligibility := new(PurgeEligibility)
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, checkActivityOptions),
		a.CheckCloudTenantPurgeActivity,
		payload,
	).Get(ctx, eligibility); err != nil {
		return nil, err
	}
	if !eligibility.Eligible {
		result.Skipped = true
		return result, nil
	}

	result.Rows = new(PurgeRowsResult)
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, purgeRowsActivityOptions),
		a.PurgeCloudTenantRowsActivity,
		payload,
	).Get(ctx, result.Rows); err != nil {
		return nil, err
	}

	stepCtx := workflow.WithActivityOptions(ctx, purgeStepActivityOptions)

	result.Storage = new(PurgeStorageResult)
	if err := workflow.ExecuteActivity(
		stepCtx,
		a.PurgeCloudTenantStorageActivity,
		payload,
	).Get(ctx, result.Storage); err != nil {
		return nil, err
	}

	result.Users = new(cloudlifecycleservice.PurgeUsersResult)
	if err := workflow.ExecuteActivity(
		stepCtx,
		a.PurgeCloudTenantUsersActivity,
		&PurgeUsersInput{PurgePayload: *payload, UserIDs: memberIDs(eligibility.Members)},
	).Get(ctx, result.Users); err != nil {
		return nil, err
	}

	result.Tenant = new(repositories.DeleteTenantResult)
	if err := workflow.ExecuteActivity(
		stepCtx,
		a.FinalizeCloudTenantPurgeActivity,
		payload,
	).Get(ctx, result.Tenant); err != nil {
		return nil, err
	}

	return result, nil
}

func memberIDs(members []*repositories.TenantMember) []pulid.ID {
	ids := make([]pulid.ID, 0, len(members))
	for _, member := range members {
		if member != nil && member.UserID.IsNotNil() {
			ids = append(ids, member.UserID)
		}
	}

	return ids
}
