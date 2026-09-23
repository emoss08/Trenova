package agentflow

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ToolObserver turns what a tool call returned into what a person sees beside
// the run, such as a chat's artifacts, and keeps a document the model
// published. It runs inside an activity, where the call's raw result exists.
// It answers with what the person now sees, for the model to be told, and with
// what it kept, so the run can account for it when the turn is saved.
type ToolObserver interface {
	ObserveTool(
		ctx context.Context,
		run RunContext,
		observation serviceports.ToolObservation,
		emit serviceports.AssistantStreamEmitter,
	) (*serviceports.ShownArtifact, []*assistantartifact.Artifact, error)
}

type ActivitiesParams struct {
	fx.In

	Logger   *zap.Logger
	Runtime  *agentruntime.Service
	Steps    serviceports.RunStepLedger
	Observer ToolObserver `optional:"true"`
}

// Activities are a turn's effects, one activity each.
type Activities struct {
	l        *zap.Logger
	runtime  *agentruntime.Service
	steps    serviceports.RunStepLedger
	observer ToolObserver
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		l:        p.Logger.Named("agentflow"),
		runtime:  p.Runtime,
		steps:    p.Steps,
		observer: p.Observer,
	}
}

// ModelCallActivity asks the model for one reply and streams it to the run's
// reader as it arrives.
func (a *Activities) ModelCallActivity(
	ctx context.Context,
	in *ModelCallInput,
) (*agentruntime.ModelReply, error) {
	emit := func(serviceports.StreamEvent) {}
	if in.Stream {
		stream, events, err := OpenStream(ctx)
		if err != nil {
			return nil, err
		}
		defer CloseStream(ctx, stream, a.l)

		// Whatever an earlier attempt streamed is already in front of the
		// reader. The retry starts the reply over, and the reader has to be
		// told so before the first word of the new one arrives, or the two
		// run together.
		if attempt := activity.GetInfo(ctx).Attempt; attempt > 1 {
			events.Publish(StreamItem{
				Event: serviceports.AssistantEventRetrying,
				Data: serviceports.AssistantRetryingEvent{
					Attempt: int(attempt) - 1,
					Reason:  "The reply is being written again after an interruption.",
					Kind:    serviceports.RetryKindRestart,
				},
			}, true)
		}

		emit = func(event serviceports.StreamEvent) {
			events.Publish(StreamItem{Event: event.Event, Data: event.Data}, false)
		}
	}

	stopBeating := modelcall.Heartbeat(ctx)
	defer stopBeating()

	reply, err := a.runtime.StreamCompletion(ctx, in.Request, emit)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return &reply, nil
}

// FindToolsActivity answers find_tools for a turn held as data.
func (a *Activities) FindToolsActivity(
	ctx context.Context,
	in *FindToolsInput,
) (*FindToolsResult, error) {
	content, loaded := a.runtime.FindFor(ctx, in.Run.request(), in.Tools, in.Arguments)

	return &FindToolsResult{Content: content, Loaded: loaded}, nil
}

// RegisterDynamic registers the tool call as the worker's dynamic activity. The
// worker's registry registers the struct itself for the model call, the tool
// search and the publish. This is not a method because the SDK refuses a
// registered struct with any exported method that is not an activity.
func RegisterDynamic(w worker.ActivityRegistry, a *Activities) {
	w.RegisterDynamicActivity(a.runTool, activity.DynamicRegisterOptions{})
}

// runTool runs one tool call. It is the worker's dynamic activity, so it is
// scheduled under the tool's own name and each tool reads as itself in the UI
// and in the SDK's metrics. It is unexported so registering the struct does not
// also register it under a name of its own.
//
// Being the dynamic activity also means it receives any activity type the
// worker has no registration for. A name that is not a tool is refused loudly
// and never retried, so a typo in an activity name fails where it happened
// instead of being quietly answered as a missing tool.
func (a *Activities) runTool(
	ctx context.Context,
	args converter.EncodedValues,
) (*ToolResult, error) {
	name := activity.GetInfo(ctx).ActivityType.Name
	if !a.runtime.KnowsTool(name) {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("activity %q is not registered and is not a tool", name),
			ErrTypeUnknownTool, nil,
		)
	}

	var in ToolInput
	if err := args.Get(&in); err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("tool %q received input it could not read: %v", name, err),
			ErrTypeBadToolInput, err,
		)
	}
	if in.Call.Call.Name != name {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("tool activity %q was sent a call to %q", name, in.Call.Call.Name),
			ErrTypeBadToolInput, nil,
		)
	}

	req := in.Run.request()
	req.Steps = a.steps
	req.Attempt = int(activity.GetInfo(ctx).Attempt)

	outcome := a.runtime.DispatchStep(ctx, req, in.Call)
	observe, kept, done := a.observing(ctx, &in.Run)
	defer done()
	outcome = a.runtime.ObserveCall(observe, &in.Call.Call, outcome)

	return &ToolResult{Outcome: outcome, Artifacts: *kept}, nil
}

// PublishArtifactActivity keeps a document the model published, and says what
// the model reads about it.
func (a *Activities) PublishArtifactActivity(
	ctx context.Context,
	in *PublishInput,
) (*ToolResult, error) {
	observe, kept, done := a.observing(ctx, &in.Run)
	defer done()
	outcome := a.runtime.PublishStep(observe, &in.Call)

	return &ToolResult{Outcome: outcome, Artifacts: *kept}, nil
}

// observing is the observer for one activity, with where what it keeps is
// collected and a function that flushes what it published. A run with nothing
// beside it, a desk's most often, has no observer.
func (a *Activities) observing(
	ctx context.Context,
	run *RunContext,
) (serviceports.ToolObserver, *[]*assistantartifact.Artifact, func()) {
	kept := new([]*assistantartifact.Artifact)
	if a.observer == nil || run.ThreadID.IsNil() {
		return nil, kept, func() {}
	}

	stream, events, openErr := OpenStream(ctx)
	if openErr != nil {
		// The call still counts; the reader misses its announcement and sees
		// the artifact when the conversation is read.
		a.l.Warn("a run's stream could not be opened for what a call showed",
			zap.String("run", run.ThreadID.String()), zap.Error(openErr))
	}
	emit := func(event serviceports.StreamEvent) {
		if events != nil {
			events.Publish(StreamItem{Event: event.Event, Data: event.Data}, true)
		}
	}

	observe := func(observation serviceports.ToolObservation) (*serviceports.ShownArtifact, error) {
		shown, artifacts, err := a.observer.ObserveTool(ctx, *run, observation, emit)
		*kept = append(*kept, artifacts...)

		return shown, err
	}

	return observe, kept, func() {
		if stream != nil {
			CloseStream(ctx, stream, a.l)
		}
	}
}
