package agentflow

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Run drives one turn from workflow code and returns what it came to. events
// is the run's stream, or nil for a run nobody reads live. The turn was built
// by an activity, which may read permissions and history; from here on nothing
// is read but the turn itself and what its activities return.
//
// An error from the model ends the turn and is returned with what the turn had
// already done, so the caller can keep the record of any write it made. A tool
// that fails does not: its failure is the model's to read, and the turn goes
// on.
func Run(
	ctx workflow.Context,
	runtime *agentruntime.Service,
	events *workflowstreams.WorkflowTopicHandle,
	rc RunContext,
	state agentruntime.TurnState,
) (*Outcome, error) {
	fx := &workflowEffects{
		ctx:     ctx,
		runtime: runtime,
		run:     rc,
		events:  events,
		steers:  workflow.GetSignalChannel(ctx, agentflowcontract.SteerSignal),
	}
	result, err := runtime.Drive(runtime.RestoreTurn(rc.request(), state), fx)

	return &Outcome{
		Result:    result,
		Artifacts: fx.outcome.Artifacts,
		Events:    fx.outcome.Events,
		Steered:   fx.outcome.Steered,
	}, err
}

// Publish sends one event to the run's reader from workflow code.
func Publish(events *workflowstreams.WorkflowTopicHandle, event kernelports.StreamEvent) error {
	return events.Publish(StreamItem{Event: event.Event, Data: event.Data})
}

type workflowEffects struct {
	ctx     workflow.Context
	runtime *agentruntime.Service
	run     RunContext
	events  *workflowstreams.WorkflowTopicHandle
	// steers is where the person's words reach the turn while it works. Nil
	// for another agent's turn on a task, which nobody steers.
	steers  workflow.ReceiveChannel
	outcome Outcome
}

func (fx *workflowEffects) Now() int64 { return workflow.Now(fx.ctx).Unix() }

// Replaying reports the loop re-running over recorded history, during which
// whatever it counts was already counted the first time.
func (fx *workflowEffects) Replaying() bool { return workflow.IsReplaying(fx.ctx) }

// Interject reads, between steps, what the person said while the turn worked
// and what changed under it.
func (fx *workflowEffects) Interject(
	_ *agentruntime.Turn,
	check *worldwatch.Check,
) worldwatch.Interjections {
	pending := fx.steers != nil && fx.steers.Len() > 0
	if !pending && check == nil {
		return worldwatch.Interjections{}
	}

	var got worldwatch.Interjections
	for pending {
		var steer worldwatch.Steer
		if !fx.steers.ReceiveAsync(&steer) {
			break
		}
		if steer.ID.IsNotNil() && !slices.Contains(fx.outcome.Steered, steer.ID) {
			fx.outcome.Steered = append(fx.outcome.Steered, steer.ID)
			got.Steers = append(got.Steers, steer)
		}
	}
	if check != nil {
		got.World = fx.checkWorld(check)
	}

	return got
}

// checkWorld reads the changes as a local activity: a read of the realtime
// stream on the worker running the turn, with no queue between steps. A read
// that fails leaves the cursor where it was, so the next step looks again.
func (fx *workflowEffects) checkWorld(check *worldwatch.Check) *realtimeports.RecordChanges {
	if fx.run.Actor == nil {
		return nil
	}
	var a *Activities
	ctx := workflow.WithLocalActivityOptions(fx.ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: worldCheckScheduleWait,
		StartToCloseTimeout:    worldCheckTimeout,
		Summary:                "Check the records in use for changes",
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 2},
	})

	var found realtimeports.RecordChanges
	err := workflow.ExecuteLocalActivity(ctx, a.CheckWorldActivity, &WorldCheckInput{
		OrganizationID: fx.run.Actor.OrganizationID,
		BusinessUnitID: fx.run.Actor.BusinessUnitID,
		Check:          *check,
	}).Get(ctx, &found)
	if err != nil {
		workflow.GetLogger(fx.ctx).Warn("could not check the turn's records for changes",
			"error", err.Error())
		return nil
	}

	return &found
}

func (fx *workflowEffects) Complete(
	_ *agentruntime.Turn,
	req *llmports.ChatCompletionRequest,
) (modelstream.Reply, error) {
	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.modelOptions())

	var reply modelstream.Reply
	err := workflow.ExecuteActivity(ctx, a.ModelCallActivity, &ModelCallInput{
		Request: req,
		Stream:  fx.events != nil,
		Scope:   fx.run.Scope(),
	}).Get(ctx, &reply)

	return reply, err
}

func (fx *workflowEffects) Dispatch(
	_ *agentruntime.Turn,
	call toolcall.Call,
) toolcall.Outcome {
	return fx.settleTool(&call, fx.startTool(&call))
}

// DispatchReads schedules every read's activity before waiting on any, so
// they run at once, and settles them in the order given: the artifacts the
// turn keeps and the outcomes the loop reads come back in call order whatever
// order the activities finish in.
func (fx *workflowEffects) DispatchReads(
	_ *agentruntime.Turn,
	calls []toolcall.Call,
) []toolcall.Outcome {
	futures := make([]workflow.Future, 0, len(calls))
	for idx := range calls {
		futures = append(futures, fx.startTool(&calls[idx]))
	}

	outcomes := make([]toolcall.Outcome, 0, len(calls))
	for idx := range calls {
		outcomes = append(outcomes, fx.settleTool(&calls[idx], futures[idx]))
	}

	return outcomes
}

func (fx *workflowEffects) startTool(call *toolcall.Call) workflow.Future {
	ctx := workflow.WithActivityOptions(fx.ctx, fx.toolOptions(call.Call.Name))

	return workflow.ExecuteActivity(ctx, call.Call.Name, &ToolInput{Run: fx.run, Call: *call})
}

func (fx *workflowEffects) settleTool(
	call *toolcall.Call,
	future workflow.Future,
) toolcall.Outcome {
	var result ToolResult
	if err := future.Get(fx.ctx, &result); err != nil {
		// A tool that could not be run to an answer, after its retries, is
		// still something the model can work with. Failing the turn over it
		// would throw away everything the turn had already done.
		return unsettledToolOutcome(call.Call.Name, err)
	}

	fx.outcome.Artifacts = append(fx.outcome.Artifacts, result.Artifacts...)

	return result.Outcome
}

// unsettledToolOutcome is what the model is told about a tool that never
// reported back. A stopped turn, a timed-out attempt or a lost worker can each
// end the wait after the tool made its change and before it said so, so
// nothing here may tell the person the change did not happen.
func unsettledToolOutcome(name string, err error) toolcall.Outcome {
	if temporal.IsCanceledError(err) {
		return toolcall.Outcome{
			Content: fmt.Sprintf("Tool %q was stopped before it reported back. It may or may "+
				"not have taken effect. Tell the person plainly that this one is unconfirmed, "+
				"and that it should be checked before anything that depends on it.", name),
			Failed: true,
		}
	}

	return toolcall.Outcome{
		Content: fmt.Sprintf("Tool %q did not report back: %s. It may or may not have taken "+
			"effect. Tell the person it is unconfirmed; do not claim it happened, and do not "+
			"claim it did not.", name, err),
		Failed: true,
	}
}

func (fx *workflowEffects) Find(
	t *agentruntime.Turn,
	arguments map[string]any,
) toolaccess.FindAnswer {
	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.findOptions())

	var result FindToolsResult
	err := workflow.ExecuteActivity(ctx, a.FindToolsActivity, &FindToolsInput{
		Run:       fx.run,
		Tools:     t.ToolsState(),
		Arguments: arguments,
	}).Get(ctx, &result)
	if err != nil {
		return toolaccess.FindAnswer{
			Content: "Tools could not be searched just now. Use the ones you have.",
		}
	}

	t.LoadTools(result.Loaded)

	return toolaccess.FindAnswer{
		Content: result.Content,
		Found:   result.Found,
		HandOff: result.HandOff,
	}
}

func (fx *workflowEffects) Emit(event kernelports.StreamEvent) {
	event, shown := fx.run.Scope().Tag(event)
	if !shown {
		return
	}
	item := StreamItem{Event: event.Event, Data: event.Data, At: workflow.Now(fx.ctx).Unix()}
	if fx.events != nil {
		if err := fx.events.Publish(item); err != nil {
			workflow.GetLogger(fx.ctx).
				Warn("could not publish a run event", "event", event.Event, "error", err)
		}
	}

	// The reply's text is already in the transcript whole. Keeping each
	// streamed fragment of it as well would record the same words hundreds of
	// times over.
	if !streamedText(event.Event) {
		fx.outcome.Events = append(fx.outcome.Events, item)
	}
}

// streamedText reports an event carrying a piece of a reply or of thinking,
// which the transcript keeps whole.
func streamedText(event string) bool {
	switch event {
	case assistantstreamports.AssistantEventDelta,
		assistantstreamports.AssistantEventReasoning,
		assistantstreamports.AssistantEventDelegateDelta,
		assistantstreamports.AssistantEventDelegateReasoning:
		return true
	default:
		return false
	}
}

// Observe keeps a published document, in an activity of its own. Any other
// call was observed where it ran, in its activity, because only there does its
// raw result exist, and what it showed is already in the outcome.
func (fx *workflowEffects) Observe(
	_ *agentruntime.Turn,
	call *llmports.ToolCall,
	outcome toolcall.Outcome,
) toolcall.Outcome {
	if !outcome.Publishes {
		return outcome
	}

	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.publishOptions())

	var result ToolResult
	err := workflow.ExecuteActivity(ctx, a.PublishArtifactActivity, &PublishInput{
		Run:  fx.run,
		Call: *call,
	}).Get(ctx, &result)
	if err != nil {
		return runtimetools.UnkeptOutcome(call.Name)
	}

	fx.outcome.Artifacts = append(fx.outcome.Artifacts, result.Artifacts...)

	return result.Outcome
}

// NewCallID mints a call id once and records it, so a replay reads back the
// same id rather than minting another.
func (fx *workflowEffects) NewCallID() string {
	var id string
	encoded := workflow.SideEffect(fx.ctx, func(workflow.Context) any {
		return agentruntime.NewCallID()
	})
	if err := encoded.Get(&id); err != nil {
		workflow.GetLogger(fx.ctx).Error("could not read a recorded call id", "error", err)
	}

	return id
}

// Delegate opens each other agent's turn on its task, in an activity, and
// drives it here, in this workflow, with the same effects: each of its model
// and tool calls is an activity of this execution, its events reach this run's
// reader tagged with the task, and a stop that cancels this run cancels them
// too. Several tasks run side by side, each in its own coroutine, and what they
// kept is folded in in the order the calls were made.
func (fx *workflowEffects) Delegate(
	_ *agentruntime.Turn,
	calls []delegation.Call,
) []delegation.Run {
	runs := make([]delegation.Run, len(calls))
	outcomes := make([]Outcome, len(calls))
	if len(calls) == 1 {
		runs[0], outcomes[0] = fx.delegate(fx.ctx, &calls[0])
	} else {
		wg := workflow.NewWaitGroup(fx.ctx)
		for idx := range calls {
			wg.Add(1)
			workflow.Go(fx.ctx, func(ctx workflow.Context) {
				defer wg.Done()
				runs[idx], outcomes[idx] = fx.delegate(ctx, &calls[idx])
			})
		}
		wg.Wait(fx.ctx)
	}
	for idx := range outcomes {
		fx.outcome.Artifacts = append(fx.outcome.Artifacts, outcomes[idx].Artifacts...)
		fx.outcome.Events = append(fx.outcome.Events, outcomes[idx].Events...)
	}

	return runs
}

func (fx *workflowEffects) delegate(
	ctx workflow.Context,
	call *delegation.Call,
) (delegation.Run, Outcome) {
	var a *Activities
	openCtx := workflow.WithActivityOptions(ctx, fx.openDelegateOptions(call.Delegate.Name))

	var opened DelegateOpening
	err := workflow.ExecuteActivity(openCtx, a.OpenDelegateActivity, &OpenDelegateInput{
		Run:  fx.run,
		Call: *call,
	}).Get(openCtx, &opened)
	if err != nil {
		return declinedDelegate(call, err), Outcome{}
	}

	sub := &workflowEffects{
		ctx:     ctx,
		runtime: fx.runtime,
		run:     opened.Run,
		events:  fx.events,
	}
	turn := fx.runtime.RestoreTurn(opened.Run.request(), opened.Turn)
	turn.ReserveCallIDs(call.CallIDs)
	turn.CarryExternalContent(call.AfterExternalContent)
	result, err := fx.runtime.Drive(turn, sub)

	run := delegation.Run{
		Definition:      opened.Run.Definition,
		Result:          result,
		Documents:       publishedDocuments(sub.outcome.Artifacts),
		ExternalContent: turn.ReadExternalContent(),
	}
	if err != nil {
		failure := modelcallcontract.FailureOf(err)
		run.Stopped = failure.Stopped
		if !failure.Stopped {
			run.Failure = delegateFailureReason(call.Delegate.Name, failure)
		}
	}

	return run, sub.outcome
}

// declinedDelegate is a task whose agent's turn could not be opened. A
// refusal is passed on as it was written; anything else never reached the
// agent, so nothing it could have written happened.
func declinedDelegate(call *delegation.Call, err error) delegation.Run {
	if temporal.IsCanceledError(err) {
		return delegation.Run{Stopped: true}
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == ErrTypeDelegateDeclined {
		return delegation.Run{Declined: appErr.Message()}
	}

	return delegation.Run{
		Declined: call.Delegate.Name + " could not be started just now. Nothing was asked " +
			"of it; try again, or tell the person.",
	}
}

// delegateFailureReason says why another agent's turn ended partway, in words
// the asking agent can pass on, without the provider's own message.
func delegateFailureReason(name string, failure *modelcallcontract.Failure) string {
	switch {
	case failure == nil:
		return name + " stopped before it finished."
	case failure.Refusal != nil:
		return failure.Err().Error()
	case failure.NoProvider:
		return name + " has no model provider it may use."
	case failure.Resting:
		return "Every model provider " + name + " may use is paused after repeated failures."
	case failure.TimedOut:
		return "The model provider did not answer " + name + " in time."
	case failure.Status != 0:
		return fmt.Sprintf("The model provider could not answer %s (status %d).",
			name, failure.Status)
	default:
		return name + "'s model call failed before it finished."
	}
}

// publishedDocuments are the documents a delegate kept beside the
// conversation, for the account of what it did. Its tables and cards are
// shown to the person as they always are; they are not what it made.
func publishedDocuments(
	artifacts []*assistantartifact.Artifact,
) []assistantstreamports.DelegateDocument {
	documents := make([]assistantstreamports.DelegateDocument, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil || artifact.Kind != assistantartifact.KindDocument {
			continue
		}
		documents = append(documents, assistantstreamports.DelegateDocument{
			ID:    artifact.ID,
			Kind:  string(artifact.Kind),
			Title: artifact.Title,
		})
	}

	return documents
}

func (fx *workflowEffects) openDelegateOptions(name string) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: openDelegateTimeout,
		Priority:            fx.priority(),
		Summary:             "Hand a task to " + name,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        10 * time.Second,
			MaximumAttempts:        3,
			NonRetryableErrorTypes: []string{ErrTypeDelegateDeclined},
		},
	}
}

func (fx *workflowEffects) priority() temporal.Priority {
	priority := temporal.Priority{PriorityKey: fx.run.PriorityKey}
	if fx.run.Actor != nil {
		priority.FairnessKey = fx.run.Actor.OrganizationID.String()
	}

	return priority
}

func (fx *workflowEffects) modelOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: modelCallTimeout,
		HeartbeatTimeout:    modelcall.HeartbeatTimeout,
		WaitForCancellation: true,
		Priority:            fx.priority(),
		Summary:             "Ask the model",
		RetryPolicy:         modelcall.RetryPolicy(modelCallAttempts),
	}
}

func (fx *workflowEffects) toolOptions(name string) workflow.ActivityOptions {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: toolTimeout,
		Priority:            fx.priority(),
		Summary:             name,
		// A stopped turn waits for a running tool to finish rather than
		// walking away from it. The write lands either way; waiting is how
		// the turn learns what it did instead of guessing.
		WaitForCancellation: true,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
			// A write's retry is safe: the ledger claimed it before it ran,
			// so a second attempt is told the first one's answer, or that its
			// outcome is unknown, and never writes twice.
			MaximumAttempts:        3,
			NonRetryableErrorTypes: []string{ErrTypeUnknownTool, ErrTypeBadToolInput},
		},
	}
	if _, heavy := heavyTools[name]; heavy {
		options.TaskQueue = temporaltype.TaskQueueAgentHeavy.String()
		options.ScheduleToStartTimeout = heavyToolWait
	}

	return options
}

func (fx *workflowEffects) publishOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: findTimeout,
		Priority:            fx.priority(),
		Summary:             "Publish a document",
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	}
}

func (fx *workflowEffects) findOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: findTimeout,
		Priority:            fx.priority(),
		Summary:             "Search for tools",
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	}
}
