package worker

import (
	"go.uber.org/fx"
)

var TemporalWorkerModule = fx.Module("temporal-workers",
	fx.Invoke(NewTemporalWorkers),
)

var Module = fx.Module("workers",
	TemporalWorkerModule,
)
