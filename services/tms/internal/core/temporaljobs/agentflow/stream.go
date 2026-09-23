package agentflow

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	// DrainWindow is how long a finished run waits for its reader to say it
	// has the last event. A reader that is there says so within a poll or
	// two; one that is not is not coming, and the run closes without it.
	DrainWindow = 15 * time.Second

	// flushTimeout bounds an activity handing its last events to the stream.
	flushTimeout = 10 * time.Second
)

// Stream is the Workflow Stream a run's workflow hosts, which its reader
// follows from the run's first event to its last.
type Stream struct {
	stream  *workflowstreams.WorkflowStream
	events  *workflowstreams.WorkflowTopicHandle
	drained bool
}

// HostStream opens the run's stream and listens for its reader saying it has
// the last event. It exists as soon as the workflow does, so a reader can
// never attach ahead of it.
func HostStream(ctx workflow.Context) (*Stream, error) {
	stream, err := workflowstreams.NewWorkflowStream(ctx, nil)
	if err != nil {
		return nil, err
	}

	host := &Stream{stream: stream, events: stream.Topic(EventsTopic)}
	workflow.Go(ctx, func(ctx workflow.Context) {
		workflow.GetSignalChannel(ctx, temporaltype.SignalStreamDrained).Receive(ctx, nil)
		host.drained = true
	})

	return host, nil
}

// Events is the topic a run's events are published on.
func (h *Stream) Events() *workflowstreams.WorkflowTopicHandle { return h.events }

// Publish sends one event to the reader from workflow code. A publish that
// fails costs the reader one event, not the run.
func (h *Stream) Publish(ctx workflow.Context, item StreamItem) {
	if err := h.events.Publish(item); err != nil {
		workflow.GetLogger(ctx).Warn("could not publish a run event",
			"event", item.Event,
			"error", err.Error(),
		)
	}
}

// Close hands the last events to the reader before the workflow ends.
//
// A stream is read by polling the workflow, so a workflow that has closed can
// no longer be read. It waits, bounded, for the reader to say it has the last
// event, then releases any poll still waiting and lets its handlers finish.
func (h *Stream) Close(ctx workflow.Context) {
	if _, err := workflow.AwaitWithTimeout(ctx, DrainWindow, func() bool {
		return h.drained
	}); err != nil {
		workflow.GetLogger(ctx).Warn("waiting for the reader was cut short",
			"error", err.Error())
	}

	h.stream.DetachPollers()
	if err := workflow.Await(ctx, func() bool {
		return workflow.AllHandlersFinished(ctx)
	}); err != nil {
		workflow.GetLogger(ctx).Warn("waiting for the run's readers was cut short",
			"error", err.Error())
	}
}

// OpenStream opens the run's stream from an activity, batching what it
// publishes every streamBatchInterval.
func OpenStream(
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

// CloseStream flushes what is still buffered. It rides a context cancellation
// cannot reach: a stopped reply's last words should still reach the reader who
// stopped it.
func CloseStream(ctx context.Context, stream *workflowstreams.Client, l *zap.Logger) {
	if stream == nil {
		return
	}

	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), flushTimeout)
	defer cancel()

	if err := stream.Close(flushCtx); err != nil {
		l.Warn("could not flush the last of a run's stream", zap.Error(err))
	}
}
