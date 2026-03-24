package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type HealthStatus struct {
	ready      atomic.Bool
	sinkStatus atomic.Value
}

func NewHealthStatus() *HealthStatus {
	h := &HealthStatus{}
	h.sinkStatus.Store(make(map[string]bool))
	return h
}

func (h *HealthStatus) SetReady(ready bool) {
	h.ready.Store(ready)
}

func (h *HealthStatus) IsReady() bool {
	return h.ready.Load()
}

func (h *HealthStatus) UpdateSinkStatus(statuses map[string]bool) {
	h.sinkStatus.Store(statuses)
}

func (h *HealthStatus) SinkStatuses() map[string]bool {
	v := h.sinkStatus.Load()
	if v == nil {
		return make(map[string]bool)
	}
	return v.(map[string]bool)
}

type SinkHealthChecker interface {
	HealthCheck(ctx context.Context) map[string]bool
	IsReady() bool
}

func StartHealthMonitor(
	status *HealthStatus,
	checker SinkHealthChecker,
	interval time.Duration,
) func() {
	done := make(chan struct{})
	var closeOnce sync.Once

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					results := checker.HealthCheck(ctx)
					cancel()

					allHealthy := true
					for _, healthy := range results {
						if !healthy {
							allHealthy = false
						}
					}

					status.UpdateSinkStatus(results)
					status.SetReady(checker.IsReady() && allHealthy && len(results) > 0)
				}
			}
		}()

	return func() {
		closeOnce.Do(func() { close(done) })
	}
}
