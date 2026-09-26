package accountingsyncjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const maxRefreshFailureLength = 1000

var healthActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

var referenceActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    2 * time.Minute,
	},
}

var markActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    5,
		MaximumInterval:    30 * time.Second,
	},
}

var modelPassActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy:         modelcall.RetryPolicy(modelPassAttempts),
}

var sweepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return append([]temporaltype.WorkflowDefinition{
		{
			Name:        CheckAccountingConnectionsWorkflowName,
			Fn:          CheckAccountingConnectionsWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Refresh accounting authorizations that are about to expire and check each connection answers",
		},
		{
			Name:        RefreshAccountingReferenceWorkflowName,
			Fn:          RefreshAccountingReferenceWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Pull an accounting system's reference data and re-score the mappings that are still open",
		},
		{
			Name:        RefreshAllAccountingReferenceWorkflowName,
			Fn:          RefreshAllAccountingReferenceWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Start the daily reference data refresh for every active accounting connection",
		},
	}, append(append(syncWorkflows(), changesWorkflows()...), driftWorkflows()...)...)
}

func CheckAccountingConnectionsWorkflow(ctx workflow.Context) (*HealthSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, healthActivityOptions)

	var a *Activities
	result := new(HealthSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.CheckAccountingConnectionsActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Accounting connection check failed", "error", err)
		return nil, err
	}

	return result, nil
}

func withFairness(base *workflow.ActivityOptions, orgID pulid.ID) workflow.ActivityOptions {
	options := *base
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityBackground,
		FairnessKey: orgID.String(),
	}
	return options
}

func RefreshAccountingReferenceWorkflow(
	ctx workflow.Context,
	payload *RefreshReferencePayload,
) (*RefreshReferenceResult, error) {
	var a *Activities
	logger := workflow.GetLogger(ctx)
	markCtx := workflow.WithActivityOptions(
		ctx,
		withFairness(&markActivityOptions, payload.OrganizationID),
	)

	if err := workflow.ExecuteActivity(markCtx, a.MarkAccountingReferenceRefreshStartedActivity, payload).
		Get(markCtx, nil); err != nil {
		return nil, err
	}

	result := &RefreshReferenceResult{}
	refreshErr := refreshReference(ctx, payload, result)

	failure := ""
	if refreshErr != nil {
		failure = refreshFailureMessage(refreshErr)
	}
	finishCtx, _ := workflow.NewDisconnectedContext(markCtx)
	if err := workflow.ExecuteActivity(finishCtx, a.MarkAccountingReferenceRefreshFinishedActivity, payload, failure).
		Get(finishCtx, nil); err != nil {
		logger.Error("Could not record the reference refresh outcome", "error", err)
	}

	if refreshErr != nil {
		return result, refreshErr
	}
	return result, nil
}

func refreshReference(
	ctx workflow.Context,
	payload *RefreshReferencePayload,
	result *RefreshReferenceResult,
) error {
	var a *Activities
	logger := workflow.GetLogger(ctx)
	pullCtx := workflow.WithActivityOptions(
		ctx,
		withFairness(&referenceActivityOptions, payload.OrganizationID),
	)

	for _, kind := range accountingsync.AllReferenceKinds() {
		var pulled ReferenceKindResult
		if err := workflow.ExecuteActivity(pullCtx, a.PullAccountingReferenceActivity, payload, kind).
			Get(pullCtx, &pulled); err != nil {
			return err
		}
		result.Kinds = append(result.Kinds, pulled)
	}

	var rescored RescoreReferenceResult
	if err := workflow.ExecuteActivity(pullCtx, a.RescoreAccountingMappingsActivity, payload).
		Get(pullCtx, &rescored); err != nil {
		return err
	}
	result.Targets = rescored.Targets
	result.Created = rescored.Created
	result.Updated = rescored.Updated
	result.Proposed = rescored.Proposed

	if len(rescored.NeedsModel) == 0 {
		return nil
	}
	modelCtx := workflow.WithActivityOptions(
		ctx,
		withFairness(&modelPassActivityOptions, payload.OrganizationID),
	)
	var applied int
	if err := workflow.ExecuteActivity(modelCtx, a.AccountingMappingModelPassActivity, payload, rescored.NeedsModel).
		Get(modelCtx, &applied); err != nil {
		logger.Warn(
			"The mapping model pass failed; the deterministic proposals stand",
			"error",
			err,
		)
		return nil
	}
	result.ModelProposed = applied
	result.Proposed += applied
	return nil
}

func refreshFailureMessage(err error) string {
	message := err.Error()
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Message() != "" {
		message = appErr.Message()
	}
	if runes := []rune(message); len(runes) > maxRefreshFailureLength {
		message = string(runes[:maxRefreshFailureLength])
	}
	return message
}

func RefreshAllAccountingReferenceWorkflow(
	ctx workflow.Context,
	payload *ReferenceSweepPayload,
) (*ReferenceSweepResult, error) {
	if payload == nil {
		payload = &ReferenceSweepPayload{}
	}
	var a *Activities
	logger := workflow.GetLogger(ctx)
	listCtx := workflow.WithActivityOptions(ctx, sweepActivityOptions)

	var page ReferenceSweepPage
	if err := workflow.ExecuteActivity(listCtx, a.ListAccountingReferenceConnectionsActivity, payload.AfterID).
		Get(listCtx, &page); err != nil {
		return nil, err
	}

	result := &ReferenceSweepResult{Started: payload.Started}
	for idx := range page.Connections {
		conn := &page.Connections[idx]
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            ReferenceWorkflowID(conn.ConnectionID),
			TaskQueue:             temporaltype.IntegrationTaskQueue,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		})
		child := workflow.ExecuteChildWorkflow(childCtx, RefreshAccountingReferenceWorkflow, conn)
		if err := child.GetChildWorkflowExecution().Get(childCtx, nil); err != nil {
			if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
				result.Skipped++
				continue
			}
			logger.Error(
				"Could not start a reference refresh",
				"connectionId",
				conn.ConnectionID,
				"error",
				err,
			)
			result.Skipped++
			continue
		}
		result.Started++
	}

	if page.More && !page.LastID.IsNil() {
		return nil, workflow.NewContinueAsNewError(
			ctx,
			RefreshAllAccountingReferenceWorkflow,
			&ReferenceSweepPayload{
				AfterID: page.LastID,
				Started: result.Started,
			},
		)
	}
	return result, nil
}
