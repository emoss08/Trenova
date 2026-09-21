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
	// evaluationTimeoutSeconds bounds a replay the way the longest run is
	// bounded: an hour, the ceiling a definition may set.
	evaluationTimeoutSeconds = 3600
	// reminderAfter is how long a proposal waits before its deciders are told
	// a second time. Long enough that a busy morning does not nag; short
	// enough that a proposal is not stale by the time anyone hears twice.
	reminderAfter      = 4 * time.Hour
	reminderBatchLimit = 500
	reminderTimeout    = 5 * time.Minute
)

var reminderOptions = workflow.ActivityOptions{
	StartToCloseTimeout: reminderTimeout,
	RetryPolicy:         deterministicRetry,
}

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
			Name:        AgentEvaluationWorkflowName,
			Fn:          AgentEvaluationWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Replay a recorded agent run against the agent as it is now, writes simulated, and compare the outcome",
		},
		{
			Name:        AgentSweepWorkflowName,
			Fn:          AgentSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Start every scheduled or continuous agent whose slot has come",
		},
		{
			Name:        ExpireStaleProposalsWorkflowName,
			Fn:          ExpireStaleProposalsWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Mark pending agent proposals whose decision window has closed as expired",
		},
		{
			Name:        DeleteStaleAskThreadsWorkflowName,
			Fn:          DeleteStaleAskThreadsWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgent.String(),
			Description: "Remove quick questions nobody kept once they have gone quiet for a month",
		},
	}
}

// askThreadRetention is how long an unkept quick question stays. A person
// who wants one keeps it, which lists it; the rest are noise the rail
// already hides and the database need not carry.
const askThreadRetention = 30 * 24 * time.Hour

// deleteStaleAskBatch bounds one activity call, so the sweep never holds a
// long transaction; the workflow loops until a batch comes back short.
const deleteStaleAskBatch = 200

// DeleteStaleAskThreadsWorkflow removes quick questions nobody kept.
func DeleteStaleAskThreadsWorkflow(ctx workflow.Context) (*DeleteStaleAskThreadsResult, error) {
	var a *Activities
	result := &DeleteStaleAskThreadsResult{}

	before := workflow.Now(ctx).Add(-askThreadRetention).Unix()
	deleteCtx := workflow.WithActivityOptions(ctx, sweepListOptions)
	for {
		var batch DeleteStaleAskThreadsResult
		if err := workflow.ExecuteActivity(deleteCtx, a.DeleteStaleAskThreadsActivity, &DeleteStaleAskThreadsInput{
			Before: before,
		}).Get(deleteCtx, &batch); err != nil {
			return nil, err
		}
		result.Deleted += batch.Deleted
		if batch.Deleted < deleteStaleAskBatch {
			return result, nil
		}
	}
}

// ExpireStaleProposalsWorkflow closes the window on proposals nobody decided.
//
// The decision service refuses an expired proposal on its own, so this is not
// what keeps a stale proposal from running. It is what keeps the thread
// honest: a card that still says "awaiting" a month later is a lie the sweeper
// corrects.
func ExpireStaleProposalsWorkflow(ctx workflow.Context) (*ExpireStaleProposalsResult, error) {
	var a *Activities
	result := &ExpireStaleProposalsResult{}

	now := workflow.Now(ctx).Unix()
	expireCtx := workflow.WithActivityOptions(ctx, sweepListOptions)
	if err := workflow.ExecuteActivity(expireCtx, a.ExpireStaleProposalsActivity, &ExpireStaleProposalsInput{
		Now: now,
	}).Get(expireCtx, result); err != nil {
		return nil, err
	}

	// Reminders ride the same sweep: what is still pending after expiry ran
	// is exactly what somebody should hear about again.
	remindCtx := workflow.WithActivityOptions(ctx, reminderOptions)
	var reminded RemindPendingProposalsResult
	if err := workflow.ExecuteActivity(remindCtx, a.RemindPendingProposalsActivity, &RemindPendingProposalsInput{
		Now:              now,
		OlderThanSeconds: int64(reminderAfter.Seconds()),
	}).Get(remindCtx, &reminded); err != nil {
		return nil, err
	}
	result.Reminded = reminded.Reminded

	return result, nil
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

// AgentEvaluationWorkflow replays one run. The replay is a single activity
// the size of a run; a failure marks the evaluation failed rather than
// leaving it Running for ever.
func AgentEvaluationWorkflow(ctx workflow.Context, payload *AgentEvaluationPayload) error {
	var a *Activities

	replayCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: runTimeout(evaluationTimeoutSeconds),
		HeartbeatTimeout:    prepareTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    runActivityRetries,
		},
	})
	var outcome ReplayRunResult
	if err := workflow.ExecuteActivity(replayCtx, a.ReplayRunActivity, payload).
		Get(replayCtx, &outcome); err != nil {
		failCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
		_ = workflow.ExecuteActivity(failCtx, a.FailEvaluationActivity, &FailEvaluationInput{
			EvaluationID: payload.EvaluationID,
			Error:        err.Error(),
			TenantInfo:   payload.tenantInfo(),
		}).Get(failCtx, nil)

		return err
	}

	return nil
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
