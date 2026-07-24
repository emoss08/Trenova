package agentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var deterministicRetry = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    30 * time.Second,
	MaximumAttempts:    3,
}

var gatherActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         deterministicRetry,
}

var diagnoseActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

var persistActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         deterministicRetry,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        BillingExceptionAgentWorkflowName,
			Fn:          BillingExceptionAgentWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Diagnose a blocked billing queue item and propose resolutions",
		},
	}
}

func BillingExceptionAgentWorkflow(ctx workflow.Context, payload *AgentRunPayload) error {
	var a *Activities
	tenant := payload.tenantInfo()

	gatherCtx := workflow.WithActivityOptions(ctx, gatherActivityOptions)
	var gathered GatherContextResult
	if err := workflow.ExecuteActivity(
		gatherCtx,
		a.GatherContextActivity,
		payload,
	).Get(gatherCtx, &gathered); err != nil {
		_ = completeRun(ctx, a, payload.RunID, agent.RunStatusFailed, tenant)
		return err
	}

	diagnoseCtx := workflow.WithActivityOptions(ctx, diagnoseActivityOptions)
	var diagnosis DiagnoseActivityResult
	if err := workflow.ExecuteActivity(diagnoseCtx, a.DiagnoseActivity, &DiagnoseActivityInput{
		RunID:         payload.RunID,
		PromptVersion: payload.PromptVersion,
		TenantInfo:    tenant,
		Context:       gathered.Context,
	}).Get(diagnoseCtx, &diagnosis); err != nil {
		_ = completeRun(ctx, a, payload.RunID, agent.RunStatusFailed, tenant)
		return err
	}

	persistCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	if err := workflow.ExecuteActivity(persistCtx, a.PersistDiagnosisActivity, &PersistDiagnosisInput{
		RunID:           payload.RunID,
		SubjectType:     payload.SubjectType,
		SubjectID:       payload.SubjectID,
		ModelIdentifier: diagnosis.ModelIdentifier,
		TenantInfo:      tenant,
		Proposals:       diagnosis.Proposals,
		Exceptions:      diagnosis.Exceptions,
	}).Get(persistCtx, nil); err != nil {
		_ = completeRun(ctx, a, payload.RunID, agent.RunStatusFailed, tenant)
		return err
	}

	if payload.ShadowMode {
		return completeRun(ctx, a, payload.RunID, agent.RunStatusShadowCompleted, tenant)
	}

	decision, timedOut := awaitDecision(ctx, payload.DecisionTimeoutSeconds)
	_ = decision

	if timedOut {
		expireCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
		return workflow.ExecuteActivity(expireCtx, a.ExpireProposalsActivity, &ExpireProposalsInput{
			RunID:      payload.RunID,
			TenantInfo: tenant,
		}).Get(expireCtx, nil)
	}

	return completeRun(ctx, a, payload.RunID, agent.RunStatusCompleted, tenant)
}

func awaitDecision(ctx workflow.Context, timeoutSeconds int) (DecisionSignal, bool) {
	ch := workflow.GetSignalChannel(ctx, AgentDecisionSignalName)
	timer := workflow.NewTimer(ctx, time.Duration(timeoutSeconds)*time.Second)

	var decision DecisionSignal
	var timedOut bool

	sel := workflow.NewSelector(ctx)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &decision)
	})
	sel.AddFuture(timer, func(workflow.Future) {
		timedOut = true
	})
	sel.Select(ctx)

	return decision, timedOut
}

func completeRun(
	ctx workflow.Context,
	a *Activities,
	runID pulid.ID,
	status agent.RunStatus,
	tenant pagination.TenantInfo,
) error {
	completeCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	return workflow.ExecuteActivity(completeCtx, a.CompleteRunActivity, &CompleteRunInput{
		RunID:      runID,
		Status:     status,
		TenantInfo: tenant,
	}).Get(completeCtx, nil)
}
