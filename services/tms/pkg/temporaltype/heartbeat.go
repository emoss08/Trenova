package temporaltype

import (
	"context"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
)

func HeartbeatEvery(ctx context.Context, interval time.Duration) func() {
	if !activity.IsActivity(ctx) || interval <= 0 {
		return func() {}
	}
	activity.RecordHeartbeat(ctx)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		ticker := time.NewTicker(interval)
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
	})
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}
