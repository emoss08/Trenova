package shipmentjobs

import "go.uber.org/fx"

var Module = fx.Module("shipment-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
