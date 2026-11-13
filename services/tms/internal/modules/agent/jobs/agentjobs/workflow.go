package agentjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/runstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/contrib/workflowstreams"
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
			Name:        agentcontract.AgentRunWorkflowName,
			Fn:          w.AgentRunWorkflow,
			Description: "Run one agent definition against its subject and file what it proposes",
		},
		{
			Name:        agentcontract.AgentScheduledRunWorkflowName,
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
			Name:        AssistantRetentionWorkflowName,
			Fn:          AssistantRetentionWorkflow,
			Description: "Remove unsaved formula conversations and old palette question records",
		},
	}
}

// heavy is the work that costs as much as a run and is never urgent, kept
// apart so an evaluation sweep cannot fill the queue a scheduled run needs.
func (w *Workflows) heavy() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        agentcontract.AgentEvaluationWorkflowName,
			Fn:          w.AgentEvaluationWorkflow,
			Description: "Replay a recorded agent run against the agent as it is now, writes simulated, and compare the outcome",
		},
	}
}

// formulaDraftRetention is how long a formula conversation about a template
// that was never saved stays once it has gone quiet.
const formulaDraftRetention = 30 * 24 * time.Hour

// paletteQuestionRetention is how long the record of a palette question stays.
// The allowance counts this month's, so two months covers every month it is
// read for.
const paletteQuestionRetention = 62 * 24 * time.Hour

// deleteStaleDraftBatch bounds one activity call, so the sweep never holds a
// long transaction; the workflow loops until a batch comes back short.
const deleteStaleDraftBatch = 200

// AssistantRetentionWorkflow removes what the assistant keeps past its use:
// formula conversations about templates nobody saved, and the records of
// palette questions older than any allowance reads.
func AssistantRetentionWorkflow(ctx workflow.Context) (*RetentionResult, error) {
	var a *Activities
	now := workflow.Now(ctx)
	deleteCtx := workflow.WithActivityOptions(ctx, sweepListOptions)

	drafts, err := sweepBatches(deleteCtx, a.DeleteStaleFormulaDraftsActivity,
		now.Add(-formulaDraftRetention).Unix())
	if err != nil {
		return nil, err
	}
	questions, err := sweepBatches(deleteCtx, a.DeleteStalePaletteQuestionsActivity,
		now.Add(-paletteQuestionRetention).Unix())
	if err != nil {
		return nil, err
	}

	return &RetentionResult{FormulaDrafts: drafts, PaletteQuestions: questions}, nil
}

// sweepBatches runs one deleting activity until a batch comes back short, all
// against one cut-off.
func sweepBatches(ctx workflow.Context, activity any, before int64) (int, error) {
	deleted := 0
	for {
		var batch RetentionBatch
		if err := workflow.ExecuteActivity(ctx, activity, &RetentionInput{Before: before}).
			Get(ctx, &batch); err != nil {
			return deleted, err
		}
		deleted += batch.Deleted
		if batch.Deleted < deleteStaleDraftBatch {
			return deleted, nil
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
	}).
		Get(expireCtx, result); err != nil {
		return nil, err
	}

	// Reminders ride the same sweep: what is still pending after expiry ran
	// is exactly what somebody should hear about again.
	remindCtx := workflow.WithActivityOptions(ctx, reminderOptions)
	var reminded RemindPendingProposalsResult
	if err := workflow.ExecuteActivity(remindCtx, a.RemindPendingProposalsActivity, &RemindPendingProposalsInput{
		Now:              now,
		OlderThanSeconds: int64(reminderAfter.Seconds()),
	}).
		Get(remindCtx, &reminded); err != nil {
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

// AgentRunWorkflow runs one agent definition against its subject and files
// what it proposed. The loop runs here, one activity per model call and per
// tool call. It ends once the proposals are filed; the run then waits on
// decisions as a record.
func (w *Workflows) AgentRunWorkflow(
	ctx workflow.Context,
	payload *agentcontract.AgentRunPayload,
) error {
	if !payload.Watched {
		return w.run(ctx, payload, nil)
	}

	// A person is waiting on this run, so what the agent writes is streamed
	// to them as it is written. The stream's last word is how the run ended,
	// after everything it did has been filed.
	stream, err := runstream.Host(ctx)
	if err != nil {
		workflow.GetLogger(ctx).Warn("a watched run could not host its stream",
			"error", err.Error())

		return w.run(ctx, payload, nil)
	}

	runErr := w.run(ctx, payload, stream.Events())
	ending := temporaltype.StreamItem{
		Event: assistantstreamports.AssistantEventDone,
		Data:  map[string]any{},
		At:    workflow.Now(ctx).Unix(),
	}
	if runErr != nil {
		ending.Event = assistantstreamports.AssistantEventError
		ending.Data = map[string]any{"message": "The agent stopped before it finished."}
	}
	stream.Publish(ctx, ending)
	stream.Close(ctx)

	return runErr
}

func (w *Workflows) run(
	ctx workflow.Context,
	payload *agentcontract.AgentRunPayload,
	events *workflowstreams.WorkflowTopicHandle,
) error {
	var a *Activities
	tenant := payload.TenantInfo()
	keep, _ := workflow.NewDisconnectedContext(ctx)

	prepareCtx := workflow.WithActivityOptions(ctx, withPriority(prepareActivityOptions, payload))
	var prepared PrepareRunResult
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareRunActivity, payload).
		Get(prepareCtx, &prepared); err != nil {
		_ = failRun(keep, payload.RunID, tenant, err)
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

	openCtx := workflow.WithActivityOptions(runCtx, withPriority(prepareActivityOptions, payload))
	var opened OpenRunResult
	if err := workflow.ExecuteActivity(openCtx, a.OpenRunActivity, &OpenRunInput{
		Payload:    payload,
		Definition: prepared.Definition,
		Subject:    prepared.Subject,
		Shadow:     prepared.ShadowMode,
	}).Get(openCtx, &opened); err != nil {
		_ = failRun(keep, payload.RunID, tenant, err)
		return err
	}

	outcome, runErr := agentflow.Run(runCtx, w.runtime, events, opened.Run, opened.Turn)
	if runErr != nil && outOfTime {
		runErr = temporal.NewApplicationError(
			fmt.Sprintf("the run did not finish within %s", runTimeout(prepared.RunTimeoutSeconds)),
			"RunTimedOut",
		)
	}

	finishCtx := workflow.WithActivityOptions(keep, withPriority(persistActivityOptions, payload))
	var finished FinishRunResult
	if err := workflow.ExecuteActivity(finishCtx, a.FinishRunActivity, &FinishRunInput{
		Payload:    payload,
		Definition: opened.Run.Definition,
		Subject:    prepared.Subject,
		Run:        outcome.Result,
		Failure:    modelcallcontract.FailureOf(runErr),
		Events:     outcome.Events,
	}).Get(finishCtx, &finished); err != nil {
		_ = failRun(keep, payload.RunID, tenant, err)
		return err
	}

	if runErr != nil {
		// What ran is recorded; a proposal made before the failure is not left
		// waiting on a run that will never hear its decision.
		_ = expireProposals(keep, a, payload.RunID, tenant)
		_ = failRun(keep, payload.RunID, tenant, runErr)
		return runErr
	}

	if prepared.ShadowMode {
		return completeRun(keep, a, payload.RunID, agent.RunStatusShadowCompleted, tenant)
	}

	if finished.PendingProposals == 0 {
		return completeRun(keep, a, payload.RunID, agent.RunStatusCompleted, tenant)
	}

	// The run waits on decisions as a record, not as a workflow: the decision
	// that leaves nothing pending completes it, and the expiry sweep completes
	// it once the rest expired (agentrunclose).
	return nil
}

func withPriority(
	options workflow.ActivityOptions,
	payload *agentcontract.AgentRunPayload,
) workflow.ActivityOptions {
	options.Priority = temporal.Priority{
		PriorityKey: agentflowcontract.PriorityBackground,
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

// AgentEvaluationWorkflow replays one run against its agent as it is now, with
// every write simulated, and stores the comparison. The loop is driven here,
// one activity per model call and per tool call.
func (w *Workflows) AgentEvaluationWorkflow(
	ctx workflow.Context,
	payload *agentcontract.AgentEvaluationPayload,
) error {
	var a *Activities
	options := prepareActivityOptions
	options.Priority = temporal.Priority{
		PriorityKey: agentflowcontract.PriorityEvaluation,
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
	payload *agentcontract.AgentEvaluationPayload,
	cause error,
) {
	failCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	if err := workflow.ExecuteActivity(failCtx, a.FailEvaluationActivity, &FailEvaluationInput{
		EvaluationID: payload.EvaluationID,
		Error:        cause.Error(),
		TenantInfo:   payload.TenantInfo(),
	}).Get(failCtx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not record why an evaluation failed",
			"evaluationId", payload.EvaluationID.String(), "error", err.Error())
	}
}

// AgentScheduledRunWorkflow is what a definition's schedule starts on each of
// its slots: one activity that starts the run when the definition may run now.
// The slot is the time the schedule fired for, so a slot starts at most one
// run however often this is retried.
func AgentScheduledRunWorkflow(
	ctx workflow.Context,
	payload *agentcontract.ScheduledRunPayload,
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
func ReconcileDefinitionSchedulesWorkflow(
	ctx workflow.Context,
) (*agentcontract.ReconcileResult, error) {
	var a *Activities

	reconcileCtx := workflow.WithActivityOptions(ctx, reconcileOptions)
	var result agentcontract.ReconcileResult
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
	runID pulid.ID,
	tenant pagination.TenantInfo,
	cause error,
) error {
	var a *Activities
	completeCtx := workflow.WithActivityOptions(ctx, persistActivityOptions)
	return workflow.ExecuteActivity(completeCtx, a.CompleteRunActivity, &CompleteRunInput{
		RunID:      runID,
		Status:     agent.RunStatusFailed,
		Error:      cause.Error(),
		TenantInfo: tenant,
	}).Get(completeCtx, nil)
}
