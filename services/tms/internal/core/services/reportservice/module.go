package reportservice

import "go.uber.org/fx"

var Module = fx.Module("report-service",
	fx.Provide(NewService),
)
