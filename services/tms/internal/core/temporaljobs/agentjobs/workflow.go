package agentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	sweepDueLimit      = 200
	minRunTimeout      = 60 * time.Second
	prepareTimeout     = 2 * time.Minute
	persistTimeout     = 2 * time.Minute
	sweepStartTimeout  = time.Minute
	sweepListTimeout   = time.Minute
	runActivityRetries = 2
)

var deterministicRetry = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    30 * time.Second,
	MaximumAttempts:    3,
}

var prepareActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: prepareTimeout,
	RetryPolicy:         deterministicRetry,
}

var persistActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: persistTimeout,
	RetryPolicy:         deterministicRetry,
}

var sweepListOptions = workflow.ActivityOptions{
	StartToCloseTimeout: sweepListTimeout,
	RetryPolicy:         deterministicRetry,
}

var sweepStartOptions = workflow.ActivityOptions{
	StartToCloseTimeout: sweepStartTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    2,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AgentRunWorkflowName,
			Fn:          AgentRunWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Run one agent definition against its subject and wait for decisions on what it proposes",
		},
		{
			Name:        AgentSweepWorkflowName,
			Fn:          AgentSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Start every scheduled or continuous agent whose slot has come",
		},
	}
}

func AgentRunWorkflow(ctx workflow.Context, payload *AgentRunPayload) error {
	var a *Activities
	tenant := payload.tenantInfo()

	prepareCtx := workflow.WithActivityOptions(ctx, prepareActivityOptions)
	var prepared PrepareRunResult
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareRunActivity, payload).
		Get(prepareCtx, &prepared); err != nil {
		_ = failRun(ctx, a, payload.RunID, tenant, err)
		return err
	}

	runCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: runTimeout(prepared.RunTimeoutSeconds),
		HeartbeatTimeout:    prepareTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    runActivityRetries,
		},
	})
	var outcome RunAgentResult
	if err := workflow.ExecuteActivity(runCtx, a.RunAgentActivity, &RunAgentInput{
		Payload:    payload,
		Definition: prepared.Definition,
		Subject:    prepared.Subject,
	}).Get(runCtx, &outcome); err != nil {
		_ = failRun(ctx, a, payload.RunID, tenant, err)
		return err
	}

	if prepared.ShadowMode {
		return completeRun(ctx, a, payload.RunID, agent.RunStatusShadowCompleted, tenant)
	}

	if outcome.PendingProposals == 0 {
		return completeRun(ctx, a, payload.RunID, agent.RunStatusCompleted, tenant)
	}

	if _, timedOut := awaitDecision(ctx, prepared.DecisionTimeoutSeconds); timedOut {
		expireCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
		return workflow.ExecuteActivity(expireCtx, a.ExpireProposalsActivity, &ExpireProposalsInput{
			RunID:      payload.RunID,
			TenantInfo: tenant,
		}).Get(expireCtx, nil)
	}

	return completeRun(ctx, a, payload.RunID, agent.RunStatusCompleted, tenant)
}

func AgentSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	result := &SweepResult{}

	listCtx := workflow.WithActivityOptions(ctx, sweepListOptions)
	var due ListDueDefinitionsResult
	if err := workflow.ExecuteActivity(listCtx, a.ListDueDefinitionsActivity, &ListDueDefinitionsInput{
		Now:   workflow.Now(ctx).Unix(),
		Limit: sweepDueLimit,
	}).Get(listCtx, &due); err != nil {
		return nil, err
	}
	result.Found = len(due.Due)

	startCtx := workflow.WithActivityOptions(ctx, sweepStartOptions)
	for i := range due.Due {
		var started StartDueRunResult
		if err := workflow.ExecuteActivity(startCtx, a.StartDueRunActivity, &due.Due[i]).
			Get(startCtx, &started); err != nil {
			result.Failed++
			workflow.GetLogger(ctx).Warn("agent sweep: start failed",
				"definitionId", due.Due[i].DefinitionID.String(),
				"error", err.Error(),
			)
			continue
		}
		if started.Started {
			result.Started++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

func runTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = agentdefinition.DefaultRunTimeoutSeconds
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout < minRunTimeout {
		return minRunTimeout
	}

	return timeout
}

func awaitDecision(ctx workflow.Context, timeoutSeconds int) (DecisionSignal, bool) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = agentdefinition.DefaultDecisionTimeoutSeconds
	}

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

func failRun(
	ctx workflow.Context,
	a *Activities,
	runID pulid.ID,
	tenant pagination.TenantInfo,
	cause error,
) error {
	completeCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	return workflow.ExecuteActivity(completeCtx, a.CompleteRunActivity, &CompleteRunInput{
		RunID:      runID,
		Status:     agent.RunStatusFailed,
		Error:      cause.Error(),
		TenantInfo: tenant,
	}).Get(completeCtx, nil)
}
