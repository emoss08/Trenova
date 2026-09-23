package agentflow

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ToolObserver turns what a tool call returned into what a person sees beside
// the run, such as a chat's artifacts. It runs inside the tool activity, the
// only place the tool's raw result exists, and returns what it made so the
// run can account for it when the turn is saved.
type ToolObserver interface {
	ObserveTool(
		ctx context.Context,
		run RunContext,
		observation serviceports.ToolObservation,
		emit serviceports.AssistantStreamEmitter,
	) []*assistantartifact.Artifact
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
	stream, events, err := openStream(ctx)
	if err != nil {
		return nil, err
	}
	defer closeStream(ctx, stream, a.l)

	// Whatever an earlier attempt streamed is already in front of the reader.
	// The retry starts the reply over, and the reader has to be told so before
	// the first word of the new one arrives, or the two run together.
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

	stopBeating := heartbeat(ctx)
	defer stopBeating()

	reply, err := a.runtime.StreamCompletion(ctx, in.Request, func(event serviceports.StreamEvent) {
		events.Publish(StreamItem{Event: event.Event, Data: event.Data}, false)
	})
	if err != nil {
		return nil, retryPolicyFor(err)
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

// RegisterActivities puts a turn's activities on a worker: the model call and
// the tool search under their own names, and the tool call as the worker's
// dynamic activity. It is not a method because the SDK refuses a registered
// struct with any exported method that is not an activity.
func RegisterActivities(w worker.ActivityRegistry, a *Activities) {
	w.RegisterActivity(a)
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
	result := &ToolResult{Outcome: outcome}

	if a.observer != nil && !in.Run.ThreadID.IsNil() {
		stream, events, err := openStream(ctx)
		if err != nil {
			a.l.Warn("tool ran, but its result could not be shown beside the run",
				zap.String("tool", name), zap.Error(err))

			return result, nil
		}
		defer closeStream(ctx, stream, a.l)

		result.Artifacts = a.observer.ObserveTool(ctx, in.Run, serviceports.ToolObservation{
			Call:   in.Call.Call,
			Data:   outcome.Data,
			Failed: outcome.Failed,
			Action: outcome.Action,
		}, func(event serviceports.StreamEvent) {
			events.Publish(StreamItem{Event: event.Event, Data: event.Data}, true)
		})
	}

	return result, nil
}

func openStream(
	ctx context.Context,
) (*workflowstreams.Client, *workflowstreams.TopicHandle, error) {
	stream, err := workflowstreams.NewClientFromActivity(ctx, workflowstreams.Options{
		BatchInterval: streamBatchInterval,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open the run's stream: %w", err)
	}

	return stream, stream.Topic(EventsTopic), nil
}

// closeStream flushes what is still buffered. It rides a context cancellation
// cannot reach: a stopped reply's last words should still reach the reader who
// stopped it.
func closeStream(ctx context.Context, stream *workflowstreams.Client, l *zap.Logger) {
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if err := stream.Close(flushCtx); err != nil {
		l.Warn("could not flush the last of a run's stream", zap.Error(err))
	}
}

// heartbeat tells Temporal the activity is alive on a timer rather than on
// output, and stops when the returned function is called.
func heartbeat(ctx context.Context) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(modelHeartbeatEvery)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			}
		}
	}()

	return func() { close(done) }
}
