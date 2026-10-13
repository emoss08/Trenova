//revive:disable-next-line:var-naming
package api

import (
	"github.com/emoss08/trenova/internal/api"
	"go.uber.org/fx"
)

var MonitoringServerModule = fx.Module("monitoring-server",
	fx.Provide(api.NewMonitoringServer),
	fx.Invoke(func(_ *api.MonitoringServer) {}),
)

// WorkerMonitoringServerModule serves the worker's metrics only when
// monitoring.metrics is enabled. The API always runs its monitoring server (it
// also answers health checks); the worker's would take the same port, so in
// local development, where both run on one host with metrics off, it stays
// down. In production each process has its own container.
var WorkerMonitoringServerModule = fx.Module("worker-monitoring-server",
	fx.Invoke(func(p api.MonitoringServerParams) {
		if p.Config.Monitoring.Metrics.Enabled {
			api.NewMonitoringServer(p)
		}
	}),
)
