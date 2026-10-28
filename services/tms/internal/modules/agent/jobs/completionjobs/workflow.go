package completionjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/completionjobs/completioncontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// StructuredCompletionWorkflow asks the model one structured question, retried
// the way the provider's answer says, within the time its caller waits.
func StructuredCompletionWorkflow(
	ctx workflow.Context,
	payload *completioncontract.StructuredCompletionPayload,
) (*llmports.StructuredCompletionResult, error) {
	var a *Activities
	callCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, callAttempts, "Ask the model",
	))

	var result llmports.StructuredCompletionResult
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
	payload *completioncontract.TestAIProviderPayload,
) (*aiproviderports.TestAIProviderResult, error) {
	var a *Activities
	testCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, 1, "Test the provider",
	))

	var result aiproviderports.TestAIProviderResult
	if err := workflow.ExecuteActivity(testCtx, a.TestAIProviderActivity, payload).
		Get(testCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAIProviderModelsWorkflow asks an endpoint once which models it serves.
// The list is read while an administrator picks a model, so a failure is
// shown to them rather than retried behind their back.
func ListAIProviderModelsWorkflow(
	ctx workflow.Context,
	payload *completioncontract.ListAIProviderModelsPayload,
) ([]aiproviderports.AIProviderModelOption, error) {
	var a *Activities
	listCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, 1, "List the endpoint's models",
	))

	var result []aiproviderports.AIProviderModelOption
	if err := workflow.ExecuteActivity(listCtx, a.ListAIProviderModelsActivity, payload).
		Get(listCtx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TestAIProviderDraftWorkflow probes an unsaved endpoint once, for the same
// reason a saved provider's test is not retried.
func TestAIProviderDraftWorkflow(
	ctx workflow.Context,
	payload *completioncontract.TestAIProviderDraftPayload,
) (*aiproviderports.AIProviderDraftTestResult, error) {
	var a *Activities
	testCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, 1, "Test the unsaved provider",
	))

	var result aiproviderports.AIProviderDraftTestResult
	if err := workflow.ExecuteActivity(testCtx, a.TestAIProviderDraftActivity, payload).
		Get(testCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// WriteBriefingWorkflow writes a day's briefing for a person who asked for it
// again.
func WriteBriefingWorkflow(
	ctx workflow.Context,
	payload *completioncontract.WriteBriefingPayload,
) (*briefingports.WriteBriefingResult, error) {
	var a *Activities
	writeCtx := workflow.WithActivityOptions(ctx, waitedOptions(
		ctx, payload.Request.TenantInfo.OrgID, writeAttempts, "Write the briefing",
	))

	var result briefingports.WriteBriefingResult
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
		budget = completioncontract.DefaultWait
	}

	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: budget,
		StartToCloseTimeout:    budget,
		HeartbeatTimeout:       min(modelcall.HeartbeatTimeout, budget),
		RetryPolicy:            modelcall.RetryPolicy(attempts),
		Summary:                summary,
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityOneShot,
			FairnessKey: organizationID.String(),
		},
	}
}
