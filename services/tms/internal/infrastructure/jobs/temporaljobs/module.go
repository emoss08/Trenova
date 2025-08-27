package temporaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/shipmentjobs"
	"go.uber.org/fx"
)

var ActivitiesModule = fx.Module(
	"temporal-activities",
	fx.Provide(
		shipmentjobs.NewShipmentJobActivities,
	),
)

var ClientModule = fx.Module(
	"temporal-client",
	fx.Provide(
		NewTemporalClient,
	),
)

var WorkerModule = fx.Module(
	"temporal-worker",
	fx.Invoke(
		shipmentjobs.NewShipmentWorker,
	),
)
