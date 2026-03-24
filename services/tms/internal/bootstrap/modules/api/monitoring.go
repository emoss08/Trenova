package api

import (
	"github.com/emoss08/trenova/internal/api"
	"go.uber.org/fx"
)

var MonitoringServerModule = fx.Module("monitoring-server",
	fx.Provide(api.NewMonitoringServer),
	fx.Invoke(func(_ *api.MonitoringServer) {}),
)
