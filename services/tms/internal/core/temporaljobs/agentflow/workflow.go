package agentflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
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
	fx := &workflowEffects{ctx: ctx, runtime: runtime, run: rc, events: events}
	result, err := runtime.Drive(runtime.RestoreTurn(rc.request(), state), fx)

	return &Outcome{Result: result, Artifacts: fx.outcome.Artifacts, Events: fx.outcome.Events}, err
}

// Publish sends one event to the run's reader from workflow code.
func Publish(events *workflowstreams.WorkflowTopicHandle, event serviceports.StreamEvent) error {
	return events.Publish(StreamItem{Event: event.Event, Data: event.Data})
}

type workflowEffects struct {
	ctx     workflow.Context
	runtime *agentruntime.Service
	run     RunContext
	events  *workflowstreams.WorkflowTopicHandle
	outcome Outcome
}

func (fx *workflowEffects) Supports(change string) bool {
	return workflow.GetVersion(fx.ctx, change, workflow.DefaultVersion, 1) == 1
}

func (fx *workflowEffects) Now() int64 { return workflow.Now(fx.ctx).Unix() }

func (fx *workflowEffects) Complete(
	_ *agentruntime.Turn,
	req *serviceports.ChatCompletionRequest,
) (agentruntime.ModelReply, error) {
	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.modelOptions())

	var reply agentruntime.ModelReply
	err := workflow.ExecuteActivity(ctx, a.ModelCallActivity, &ModelCallInput{
		Request: req,
		Stream:  fx.events != nil,
		Scope:   fx.run.Scope(),
	}).Get(ctx, &reply)

	return reply, err
}

func (fx *workflowEffects) Dispatch(
	_ *agentruntime.Turn,
	call agentruntime.DispatchCall,
) agentruntime.ToolOutcome {
	ctx := workflow.WithActivityOptions(fx.ctx, fx.toolOptions(call.Call.Name))

	var result ToolResult
	err := workflow.ExecuteActivity(ctx, call.Call.Name, &ToolInput{Run: fx.run, Call: call}).
		Get(ctx, &result)
	if err != nil {
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
func unsettledToolOutcome(name string, err error) agentruntime.ToolOutcome {
	if temporal.IsCanceledError(err) {
		return agentruntime.ToolOutcome{
			Content: fmt.Sprintf("Tool %q was stopped before it reported back. It may or may "+
				"not have taken effect. Tell the person plainly that this one is unconfirmed, "+
				"and that it should be checked before anything that depends on it.", name),
			Failed: true,
		}
	}

	return agentruntime.ToolOutcome{
		Content: fmt.Sprintf("Tool %q did not report back: %s. It may or may not have taken "+
			"effect. Tell the person it is unconfirmed; do not claim it happened, and do not "+
			"claim it did not.", name, err),
		Failed: true,
	}
}

func (fx *workflowEffects) Find(t *agentruntime.Turn, arguments map[string]any) string {
	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.findOptions())

	var result FindToolsResult
	err := workflow.ExecuteActivity(ctx, a.FindToolsActivity, &FindToolsInput{
		Run:       fx.run,
		Tools:     t.ToolsState(),
		Arguments: arguments,
	}).Get(ctx, &result)
	if err != nil {
		return "Tools could not be searched just now. Use the ones you have."
	}

	t.LoadTools(result.Loaded)

	return result.Content
}

func (fx *workflowEffects) Emit(event serviceports.StreamEvent) {
	event, shown := fx.run.Scope().Tag(event)
	if !shown {
		return
	}
	item := StreamItem{Event: event.Event, Data: event.Data}
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
	case serviceports.AssistantEventDelta,
		serviceports.AssistantEventReasoning,
		serviceports.AssistantEventDelegateDelta,
		serviceports.AssistantEventDelegateReasoning:
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
	call *serviceports.ToolCall,
	outcome agentruntime.ToolOutcome,
) agentruntime.ToolOutcome {
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
		return agentruntime.ToolOutcome{
			Content: fmt.Sprintf("Tool %q could not keep the document just now. "+
				"Put the text in your reply instead.", call.Name),
			Failed: true,
		}
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

// Delegate opens another agent's turn on the task, in an activity, and drives
// it here, in this workflow, with the same effects: each of its model and tool
// calls is an activity of this execution, its events reach this run's reader
// tagged with the task, and a stop that cancels this run cancels them too.
func (fx *workflowEffects) Delegate(
	_ *agentruntime.Turn,
	call agentruntime.DelegateCall,
) agentruntime.DelegateRun {
	var a *Activities
	ctx := workflow.WithActivityOptions(fx.ctx, fx.openDelegateOptions(call.Delegate.Name))

	var opened DelegateOpening
	err := workflow.ExecuteActivity(ctx, a.OpenDelegateActivity, &OpenDelegateInput{
		Run:  fx.run,
		Call: call,
	}).Get(ctx, &opened)
	if err != nil {
		return declinedDelegate(call, err)
	}

	sub := &workflowEffects{
		ctx:     fx.ctx,
		runtime: fx.runtime,
		run:     opened.Run,
		events:  fx.events,
	}
	turn := fx.runtime.RestoreTurn(opened.Run.request(), opened.Turn)
	turn.ReserveCallIDs(call.CallIDs)
	result, err := fx.runtime.Drive(turn, sub)

	fx.outcome.Artifacts = append(fx.outcome.Artifacts, sub.outcome.Artifacts...)
	fx.outcome.Events = append(fx.outcome.Events, sub.outcome.Events...)

	run := agentruntime.DelegateRun{
		Definition: opened.Run.Definition,
		Result:     result,
		Documents:  publishedDocuments(sub.outcome.Artifacts),
	}
	if err != nil {
		failure := modelcall.FailureOf(err)
		run.Stopped = failure.Stopped
		if !failure.Stopped {
			run.Failure = delegateFailureReason(call.Delegate.Name, failure)
		}
	}

	return run
}

// declinedDelegate is a task whose agent's turn could not be opened. A
// refusal is passed on as it was written; anything else never reached the
// agent, so nothing it could have written happened.
func declinedDelegate(call agentruntime.DelegateCall, err error) agentruntime.DelegateRun {
	if temporal.IsCanceledError(err) {
		return agentruntime.DelegateRun{Stopped: true}
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == ErrTypeDelegateDeclined {
		return agentruntime.DelegateRun{Declined: appErr.Message()}
	}

	return agentruntime.DelegateRun{
		Declined: call.Delegate.Name + " could not be started just now. Nothing was asked " +
			"of it; try again, or tell the person.",
	}
}

// delegateFailureReason says why another agent's turn ended partway, in words
// the asking agent can pass on, without the provider's own message.
func delegateFailureReason(name string, failure *modelcall.Failure) string {
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
) []serviceports.DelegateDocument {
	documents := make([]serviceports.DelegateDocument, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil || artifact.Kind != assistantartifact.KindDocument {
			continue
		}
		documents = append(documents, serviceports.DelegateDocument{
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
