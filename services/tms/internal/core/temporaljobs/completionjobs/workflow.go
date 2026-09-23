package completionjobs

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// StructuredCompletionWorkflow asks the model one structured question, retried
// the way the provider's answer says, within the time its caller waits.
func StructuredCompletionWorkflow(
	ctx workflow.Context,
	payload *StructuredCompletionPayload,
) (*serviceports.StructuredCompletionResult, error) {
	var a *Activities
	callCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, callAttempts, "Ask the model",
	))

	var result serviceports.StructuredCompletionResult
	if err := workflow.ExecuteActivity(callCtx, a.CompleteStructuredActivity, payload.Request).
		Get(callCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TestAIProviderWorkflow probes a provider once. A test is not retried: the
// administrator is asking whether the connection works now, and a retry that
// hid a first failure would answer a different question.
func TestAIProviderWorkflow(
	ctx workflow.Context,
	payload *TestAIProviderPayload,
) (*serviceports.TestAIProviderResult, error) {
	var a *Activities
	testCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, 1, "Test the provider",
	))

	var result serviceports.TestAIProviderResult
	if err := workflow.ExecuteActivity(testCtx, a.TestAIProviderActivity, payload).
		Get(testCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// WriteBriefingWorkflow writes a day's briefing for a person who asked for it
// again.
func WriteBriefingWorkflow(
	ctx workflow.Context,
	payload *WriteBriefingPayload,
) (*serviceports.WriteBriefingResult, error) {
	var a *Activities
	writeCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, writeAttempts, "Write the briefing",
	))

	var result serviceports.WriteBriefingResult
	if err := workflow.ExecuteActivity(writeCtx, a.WriteBriefingActivity, payload).
		Get(writeCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// waitedOptions spend the whole of the workflow's time on its one activity:
// the caller stops waiting when the workflow's execution timeout passes, so
// no attempt may outlast it.
func waitedOptions(
	ctx workflow.Context,
	organizationID pulid.ID,
	attempts int32,
	summary string,
) workflow.ActivityOptions {
	budget := workflow.GetInfo(ctx).WorkflowExecutionTimeout
	if budget <= 0 {
		budget = defaultWait
	}

	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: budget,
		StartToCloseTimeout:    budget,
		HeartbeatTimeout:       min(modelcall.HeartbeatTimeout, budget),
		RetryPolicy:            modelcall.RetryPolicy(attempts),
		Summary:                summary,
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityOneShot,
			FairnessKey: organizationID.String(),
		},
	}
}
