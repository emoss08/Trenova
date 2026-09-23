package modelcall

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

const (
	// heartbeatEvery is how often a model call says it is alive, whether or
	// not tokens are arriving. Heartbeating only on output meant a model
	// thinking silently for longer than the heartbeat timeout was killed
	// mid-thought.
	heartbeatEvery = 10 * time.Second

	// HeartbeatTimeout is how long silence means the worker is gone, so the
	// call moves to another worker quickly rather than after the full call
	// timeout.
	HeartbeatTimeout = 45 * time.Second
)

// Heartbeat tells Temporal the activity is alive on a timer rather than on
// output, and stops when the returned function is called.
func Heartbeat(ctx context.Context) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(heartbeatEvery)
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
