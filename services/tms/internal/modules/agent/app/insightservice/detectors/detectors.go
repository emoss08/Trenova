package detectors

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// All is every insight rule a deployment runs, in the order a refresh runs
// them.
func All(metrics insightports.InsightMetricsRepository) []detector.Detector {
	return []detector.Detector{
		NewOnTimeDecline(metrics),
		NewUnbilledAging(metrics),
		NewUnbilledDetention(metrics),
		NewEmptyMiles(metrics),
		NewCredentialExpiry(metrics),
	}
}

func shipmentAndCustomer() []permission.Resource {
	return []permission.Resource{permission.ResourceShipment, permission.ResourceCustomer}
}

func shipmentAndLocation() []permission.Resource {
	return []permission.Resource{permission.ResourceShipment, permission.ResourceLocation}
}

func workerOnly() []permission.Resource {
	return []permission.Resource{permission.ResourceWorker}
}
