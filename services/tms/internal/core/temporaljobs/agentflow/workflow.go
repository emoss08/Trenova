package agentflow

import (
	"fmt"
	"time"

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
	fx := &workflowEffects{ctx: ctx, run: rc, events: events}
	result, err := runtime.Drive(runtime.RestoreTurn(rc.request(), state), fx)

	return &Outcome{Result: result, Artifacts: fx.outcome.Artifacts, Events: fx.outcome.Events}, err
}

// Publish sends one event to the run's reader from workflow code.
func Publish(events *workflowstreams.WorkflowTopicHandle, event serviceports.StreamEvent) error {
	return events.Publish(StreamItem{Event: event.Event, Data: event.Data})
}

type workflowEffects struct {
	ctx     workflow.Context
	run     RunContext
	events  *workflowstreams.WorkflowTopicHandle
	outcome Outcome
}

func (fx *workflowEffects) Supports(change string) bool {
	return workflow.GetVersion(fx.ctx, change, workflow.DefaultVersion, 1) == 1
}

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
		// A tool that could not be run at all, after its retries, is still an
		// answer the model can work with. Failing the turn over it would throw
		// away everything the turn had already done.
		return agentruntime.ToolOutcome{
			Content: fmt.Sprintf("Tool %q could not be run just now: %s. "+
				"Tell the person it did not happen; do not claim it did.", call.Call.Name, err),
			Failed: true,
		}
	}

	fx.outcome.Artifacts = append(fx.outcome.Artifacts, result.Artifacts...)

	return result.Outcome
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
	if event.Event != serviceports.AssistantEventDelta &&
		event.Event != serviceports.AssistantEventReasoning {
		fx.outcome.Events = append(fx.outcome.Events, item)
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
