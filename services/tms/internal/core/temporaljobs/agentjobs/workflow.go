package agentjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	minRunTimeout     = 60 * time.Second
	prepareTimeout    = 2 * time.Minute
	persistTimeout    = 2 * time.Minute
	sweepStartTimeout = time.Minute
	sweepListTimeout  = time.Minute
	// reconcileTimeout bounds one reconcile of every agent's schedule, a
	// read of every scheduled agent and a call per schedule.
	reconcileTimeout = 10 * time.Minute
	// runActivityRetries is how many times a run may be attempted.
	//
	// Two was the old ceiling, and it was low because a retry was dangerous:
	// the second attempt re-ran every write the first had made. With the step
	// ledger claiming each tool call before it runs, a retry redoes the
	// reasoning and none of the writing, so a transient failure is worth
	// another try rather than a failed run somebody has to notice.
	runActivityRetries = 3
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

var reconcileOptions = workflow.ActivityOptions{
	StartToCloseTimeout: reconcileTimeout,
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

// background is the work that runs without anybody watching: the runs, the
// schedules that start them, and the periodic sweeps.
func (w *Workflows) background() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        AgentRunWorkflowName,
			Fn:          w.AgentRunWorkflow,
			Description: "Run one agent definition against its subject and wait for decisions on what it proposes",
		},
		{
			Name:        AgentScheduledRunWorkflowName,
			Fn:          AgentScheduledRunWorkflow,
			Description: "Start a scheduled or continuous agent's run on the slot its schedule fired for",
		},
		{
			Name:        ReconcileSchedulesWorkflowName,
			Fn:          ReconcileDefinitionSchedulesWorkflow,
			Description: "Keep one schedule behind every scheduled or continuous agent",
		},
		{
			Name:        ExpireStaleProposalsWorkflowName,
			Fn:          ExpireStaleProposalsWorkflow,
			Description: "Mark pending agent proposals whose decision window has closed as expired",
		},
		{
			Name:        DeleteStaleAskThreadsWorkflowName,
			Fn:          DeleteStaleAskThreadsWorkflow,
			Description: "Remove quick questions nobody kept once they have gone quiet for a month",
		},
		{
			Name:        CaptureEvalCaseCandidatesWorkflowName,
			Fn:          CaptureEvalCaseCandidatesWorkflow,
			Description: "Capture decided proposals as candidate evaluation cases",
		},
		{
			Name:        PurgeEvalCasesWorkflowName,
			Fn:          PurgeEvalCasesWorkflow,
			Description: "Purge expired evaluation cases and those of deleted conversations",
		},
	}
}

// heavy is the work that costs as much as a run and is never urgent, kept
// apart so an evaluation sweep cannot fill the queue a scheduled run needs.
func (w *Workflows) heavy() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        AgentEvaluationWorkflowName,
			Fn:          w.AgentEvaluationWorkflow,
			Description: "Replay a recorded agent run against the agent as it is now, writes simulated, and compare the outcome",
		},
	}
}

// drain is the same set, on the queue everything used before the split.
//
// A workflow's task queue is fixed when it starts, so every run and evaluation
// already in flight when the split deployed still dispatches to agent-queue.
// Without a worker polling it they would hang until their timeouts fired. It
// goes once Temporal shows no open executions there.
func (w *Workflows) drain() []registry.WorkflowDefinition {
	return append(w.background(), w.heavy()...)
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

// Workflows are the agent's run and evaluation workflows. They hold the agent
// runtime because the agent loop runs in workflow code, and the loop is the
// runtime's.
type Workflows struct {
	runtime *agentruntime.Service
}

func NewWorkflows(runtime *agentruntime.Service) *Workflows {
	return &Workflows{runtime: runtime}
}

// AgentRunWorkflow runs one agent definition against its subject and waits for
// decisions on what it proposed.
//
// A run started before the agent loop moved into workflow code runs to the end
// on the code it started on: those are the executions the recorded histories
// in testdata replay, most of them parked in a day-long wait for a decision.
// Every run started since runs the loop here, one activity per model call and
// per tool call.
func (w *Workflows) AgentRunWorkflow(ctx workflow.Context, payload *AgentRunPayload) error {
	if workflow.GetVersion(ctx, agentflowChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return runInOneActivity(ctx, payload)
	}

	var a *Activities
	tenant := payload.tenantInfo()

	prepareCtx := workflow.WithActivityOptions(ctx, withPriority(prepareActivityOptions, payload,
		agentflow.PriorityBackground))
	var prepared PrepareRunResult
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareRunActivity, payload).
		Get(prepareCtx, &prepared); err != nil {
		_ = failRun(ctx, a, payload.RunID, tenant, err)
		return err
	}

	// The definition's run timeout bounds the whole loop, as it bounded the
	// one activity the loop used to be.
	runCtx, cancelRun := workflow.WithCancel(ctx)
	defer cancelRun()
	outOfTime := false
	workflow.Go(ctx, func(timerCtx workflow.Context) {
		if workflow.Sleep(timerCtx, runTimeout(prepared.RunTimeoutSeconds)) == nil {
			outOfTime = true
			cancelRun()
		}
	})

	openCtx := workflow.WithActivityOptions(runCtx, withPriority(prepareActivityOptions, payload,
		agentflow.PriorityBackground))
	var opened OpenRunResult
	if err := workflow.ExecuteActivity(openCtx, a.OpenRunActivity, &OpenRunInput{
		Payload:    payload,
		Definition: prepared.Definition,
		Subject:    prepared.Subject,
		Shadow:     prepared.ShadowMode,
	}).Get(openCtx, &opened); err != nil {
		_ = failRun(ctx, a, payload.RunID, tenant, err)
		return err
	}

	outcome, runErr := agentflow.Run(runCtx, w.runtime, nil, opened.Run, opened.Turn)
	if runErr != nil && outOfTime {
		runErr = temporal.NewApplicationError(
			fmt.Sprintf("the run did not finish within %s", runTimeout(prepared.RunTimeoutSeconds)),
			"RunTimedOut",
		)
	}

	finishCtx := workflow.WithActivityOptions(ctx, withPriority(persistActivityOptions, payload,
		agentflow.PriorityBackground))
	var finished FinishRunResult
	if err := workflow.ExecuteActivity(finishCtx, a.FinishRunActivity, &FinishRunInput{
		Payload:    payload,
		Definition: opened.Run.Definition,
		Subject:    prepared.Subject,
		Run:        outcome.Result,
		Failure:    modelcall.FailureOf(runErr),
		Events:     outcome.Events,
	}).Get(finishCtx, &finished); err != nil {
		_ = failRun(ctx, a, payload.RunID, tenant, err)
		return err
	}

	if runErr != nil {
		// What ran is recorded; a proposal made before the failure is not left
		// waiting on a run that will never hear its decision.
		_ = expireProposals(ctx, a, payload.RunID, tenant)
		_ = failRun(ctx, a, payload.RunID, tenant, runErr)
		return runErr
	}

	if prepared.ShadowMode {
		return completeRun(ctx, a, payload.RunID, agent.RunStatusShadowCompleted, tenant)
	}

	if finished.PendingProposals == 0 {
		return completeRun(ctx, a, payload.RunID, agent.RunStatusCompleted, tenant)
	}

	if timedOut := awaitEveryDecision(ctx, a, payload, prepared.DecisionTimeoutSeconds); timedOut {
		return expireProposals(ctx, a, payload.RunID, tenant)
	}

	return completeRun(ctx, a, payload.RunID, agent.RunStatusCompleted, tenant)
}

// awaitEveryDecision waits until none of the run's proposals is still pending,
// or the decision window closes. It reports whether it closed.
//
// A decision signal says something was decided, and the ledger says what is
// left: a plan decides several steps under one signal, and a proposal decided
// twice over in a race sends two. Counting what is still pending after each
// signal is right in both cases, where counting signals is right in neither.
func awaitEveryDecision(
	ctx workflow.Context,
	a *Activities,
	payload *AgentRunPayload,
	timeoutSeconds int,
) bool {
	if timeoutSeconds <= 0 {
		timeoutSeconds = agentdefinition.DefaultDecisionTimeoutSeconds
	}

	decisions := workflow.GetSignalChannel(ctx, AgentDecisionSignalName)
	deadline := workflow.Now(ctx).Add(time.Duration(timeoutSeconds) * time.Second)
	countCtx := workflow.WithActivityOptions(ctx, withPriority(persistActivityOptions, payload,
		agentflow.PriorityBackground))

	for {
		remaining := deadline.Sub(workflow.Now(ctx))
		if remaining <= 0 {
			return true
		}

		var decision DecisionSignal
		if received, _ := decisions.ReceiveWithTimeout(ctx, remaining, &decision); !received {
			return true
		}

		var pending int
		if err := workflow.ExecuteActivity(countCtx, a.PendingProposalsActivity,
			&PendingProposalsInput{RunID: payload.RunID, TenantInfo: payload.tenantInfo()},
		).Get(countCtx, &pending); err != nil {
			workflow.GetLogger(ctx).Warn("could not count the run's pending proposals",
				"runId", payload.RunID.String(), "error", err.Error())
			continue
		}
		if pending == 0 {
			return false
		}
	}
}

func withPriority(
	options workflow.ActivityOptions,
	payload *AgentRunPayload,
	priorityKey int,
) workflow.ActivityOptions {
	options.Priority = temporal.Priority{
		PriorityKey: priorityKey,
		FairnessKey: payload.OrganizationID.String(),
	}

	return options
}

func expireProposals(
	ctx workflow.Context,
	a *Activities,
	runID pulid.ID,
	tenant pagination.TenantInfo,
) error {
	expireCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	return workflow.ExecuteActivity(expireCtx, a.ExpireProposalsActivity, &ExpireProposalsInput{
		RunID:      runID,
		TenantInfo: tenant,
	}).Get(expireCtx, nil)
}

// runInOneActivity is the run as it was before the loop moved into workflow
// code: the whole loop inside one activity, and the first decision ending the
// wait. It runs only executions that started on it, and must not change: a
// change to what it schedules breaks every one of them mid-wait.
func runInOneActivity(ctx workflow.Context, payload *AgentRunPayload) error {
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

// AgentEvaluationWorkflow replays one run against its agent as it is now, with
// every write simulated, and stores the comparison.
//
// An evaluation started before the loop moved into workflow code finishes on
// the one activity it started on. Every one since drives the loop here.
func (w *Workflows) AgentEvaluationWorkflow(
	ctx workflow.Context,
	payload *AgentEvaluationPayload,
) error {
	if workflow.GetVersion(ctx, agentflowChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return replayInOneActivity(ctx, payload)
	}

	var a *Activities
	options := prepareActivityOptions
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityEvaluation,
		FairnessKey: payload.OrganizationID.String(),
	}

	openCtx := workflow.WithActivityOptions(ctx, options)
	var opened OpenReplayResult
	if err := workflow.ExecuteActivity(openCtx, a.OpenReplayActivity, payload).
		Get(openCtx, &opened); err != nil {
		failEvaluation(ctx, a, payload, err)
		return err
	}
	if opened.Done {
		return nil
	}

	replayCtx, cancelReplay := workflow.WithCancel(ctx)
	defer cancelReplay()
	workflow.Go(ctx, func(timerCtx workflow.Context) {
		if workflow.Sleep(timerCtx, runTimeout(evaluationTimeoutSeconds)) == nil {
			cancelReplay()
		}
	})

	outcome, err := agentflow.Run(replayCtx, w.runtime, nil, opened.Run, opened.Turn)
	if err != nil {
		failEvaluation(ctx, a, payload, err)
		return err
	}

	finishCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	if err = workflow.ExecuteActivity(finishCtx, a.FinishReplayActivity, &FinishReplayInput{
		Payload:   payload,
		Originals: opened.Originals,
		Run:       outcome.Result,
	}).Get(finishCtx, nil); err != nil {
		failEvaluation(ctx, a, payload, err)
		return err
	}

	return nil
}

// failEvaluation records why a replay did not finish, so the evaluation never
// sits at Running with nothing to say.
func failEvaluation(
	ctx workflow.Context,
	a *Activities,
	payload *AgentEvaluationPayload,
	cause error,
) {
	failCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	if err := workflow.ExecuteActivity(failCtx, a.FailEvaluationActivity, &FailEvaluationInput{
		EvaluationID: payload.EvaluationID,
		Error:        cause.Error(),
		TenantInfo:   payload.tenantInfo(),
	}).Get(failCtx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not record why an evaluation failed",
			"evaluationId", payload.EvaluationID.String(), "error", err.Error())
	}
}

// replayInOneActivity is the replay as it was before the loop moved into
// workflow code. It runs only evaluations that started on it, and must not
// change.
func replayInOneActivity(ctx workflow.Context, payload *AgentEvaluationPayload) error {
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

// AgentScheduledRunWorkflow is what a definition's schedule starts on each of
// its slots: one activity that starts the run when the definition may run now.
// The slot is the time the schedule fired for, so a slot starts at most one
// run however often this is retried.
func AgentScheduledRunWorkflow(
	ctx workflow.Context,
	payload *ScheduledRunPayload,
) (*StartScheduledRunResult, error) {
	var a *Activities

	slot := workflow.Now(ctx)
	if scheduled, ok := workflow.GetTypedSearchAttributes(ctx).
		GetTime(scheduledStartTime); ok {
		slot = scheduled
	}

	fired := *payload
	fired.Slot = slot.Unix()

	startCtx := workflow.WithActivityOptions(ctx, sweepStartOptions)
	var started StartScheduledRunResult
	if err := workflow.ExecuteActivity(startCtx, a.StartScheduledRunActivity, &fired).
		Get(startCtx, &started); err != nil {
		return nil, err
	}

	return &started, nil
}

// scheduledStartTime is the slot a schedule fired for, which Temporal sets on
// every workflow a schedule starts.
var scheduledStartTime = temporal.NewSearchAttributeKeyTime("TemporalScheduledStartTime")

// ReconcileDefinitionSchedulesWorkflow makes every agent's schedule match the
// agent. It runs when a worker starts and every quarter hour after.
func ReconcileDefinitionSchedulesWorkflow(ctx workflow.Context) (*ReconcileResult, error) {
	var a *Activities

	reconcileCtx := workflow.WithActivityOptions(ctx, reconcileOptions)
	var result ReconcileResult
	if err := workflow.ExecuteActivity(reconcileCtx, a.ReconcileDefinitionSchedulesActivity).
		Get(reconcileCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
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
