package agentwaitjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// recordChangeGroup is the consumer group the workers share on the realtime
// stream: each change to a watched record reaches one worker.
const recordChangeGroup = "agent-waits"

type FollowerParams struct {
	fx.In

	LC       fx.Lifecycle
	Feed     realtimeports.RecordWatchFeed
	Notifier agentwaitports.RecordChangeNotifier
	Logger   *zap.Logger
}

// followRecordChanges reads the realtime stream for the records open waits
// watch, for as long as the worker runs, and hands each change to the waits.
func followRecordChanges(p FollowerParams) {
	l := p.Logger.Named("agent-wait.record-changes")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				defer close(done)
				if err := p.Feed.Follow(ctx, recordChangeGroup, p.Notifier.NotifyRecordChanges); err != nil {
					l.Error("stopped following changes to watched records", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(stop context.Context) error {
			cancel()
			select {
			case <-done:
			case <-stop.Done():
			}
			return nil
		},
	})
}

// WorkerModule follows the change stream in a worker process. The API never
// follows it: a write there only publishes.
var WorkerModule = fx.Module("agent-wait-jobs-worker",
	fx.Invoke(followRecordChanges),
)
