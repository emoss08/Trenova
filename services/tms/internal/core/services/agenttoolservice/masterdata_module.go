package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/internal/core/services/carriercapacityservice"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/internal/core/services/commodityservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowerservice"
)

var (
	_ carrierKeeper       = (*carrierservice.Service)(nil)
	_ customerKeeper      = (*customerservice.Service)(nil)
	_ commodityKeeper     = (*commodityservice.Service)(nil)
	_ capacityKeeper      = (*carriercapacityservice.Service)(nil)
	_ hazmatKeeper        = (*hazardousmaterialservice.Service)(nil)
	_ locationKeeper      = (*locationservice.Service)(nil)
	_ tractorKeeper       = (*tractorservice.Service)(nil)
	_ trailerKeeper       = (*trailerservice.Service)(nil)
	_ documentKeeper      = (*documentservice.Service)(nil)
	_ insightRestorer     = (*insightservice.Service)(nil)
	_ alertKeeper         = (*tablechangealertservice.Service)(nil)
	_ watchtowerDismisser = (*watchtowerservice.Dismisser)(nil)
	_ captureKeeper       = (*captureservice.Service)(nil)
	_ stateLookup         = repositories.UsStateRepository(nil)
)

func masterDataToolProviders() []any {
	return []any{
		provideCreateCarrierTool,
		provideUpdateCarrierTool,
		provideUpdateCarrierStatusTool,
		provideCreateCustomerTool,
		provideUpdateCustomerTool,
		provideUpdateCustomerStatusTool,
		provideCreateCommodityTool,
		provideUpdateCommodityTool,
		provideUpdateCommodityStatusTool,
		provideCreateCarrierCapacityPostingTool,
		provideUpdateCarrierCapacityPostingTool,
		provideDeleteCarrierCapacityPostingTool,
		provideCreateHazardousMaterialTool,
		provideUpdateHazardousMaterialTool,
		provideUpdateHazardousMaterialStatusTool,
		provideUpdateLocationTool,
		provideUpdateLocationStatusTool,
		provideCreateTractorTool,
		provideUpdateTractorTool,
		provideLocateTractorTool,
		provideCreateTrailerTool,
		provideUpdateTrailerTool,
		provideLocateTrailerTool,
		provideDeleteDocumentsTool,
		provideRestoreDocumentVersionTool,
		provideRestoreInsightTool,
		provideUpdateTableChangeAlertTool,
		provideSetTableChangeAlertStatusTool,
		provideDeleteTableChangeAlertTool,
		provideDismissWatchtowerItemTool,
		provideFileCaptureItemsTool,
		provideDiscardCaptureItemTool,
		provideDiscardCaptureBatchTool,
	}
}

func provideCreateCarrierTool(
	carriers *carrierservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newCreateCarrierTool(carriers, states)
}

func provideUpdateCarrierTool(
	carriers *carrierservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateCarrierTool(carriers, states)
}

func provideUpdateCarrierStatusTool(
	carriers *carrierservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateCarrierStatusTool(carriers, states)
}

func provideCreateCustomerTool(
	customers *customerservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newCreateCustomerTool(customers, states)
}

func provideUpdateCustomerTool(
	customers *customerservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateCustomerTool(customers, states)
}

func provideUpdateCustomerStatusTool(
	customers *customerservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateCustomerStatusTool(customers, states)
}

func provideCreateCommodityTool(commodities *commodityservice.Service) serviceports.AgentTool {
	return newCreateCommodityTool(commodities)
}

func provideUpdateCommodityTool(commodities *commodityservice.Service) serviceports.AgentTool {
	return newUpdateCommodityTool(commodities)
}

func provideUpdateCommodityStatusTool(
	commodities *commodityservice.Service,
) serviceports.AgentTool {
	return newUpdateCommodityStatusTool(commodities)
}

func provideCreateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) serviceports.AgentTool {
	return newCreateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) serviceports.AgentTool {
	return newUpdateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialStatusTool(
	materials *hazardousmaterialservice.Service,
) serviceports.AgentTool {
	return newUpdateHazardousMaterialStatusTool(materials)
}

func provideUpdateLocationTool(
	locations *locationservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateLocationTool(locations, states)
}

func provideUpdateLocationStatusTool(
	locations *locationservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateLocationStatusTool(locations, states)
}

func provideCreateTractorTool(
	tractors *tractorservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newCreateTractorTool(tractors, states)
}

func provideUpdateTractorTool(
	tractors *tractorservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateTractorTool(tractors, states)
}

func provideLocateTractorTool(tractors *tractorservice.Service) serviceports.AgentTool {
	return newLocateTractorTool(tractors)
}

func provideCreateTrailerTool(
	trailers *trailerservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newCreateTrailerTool(trailers, states)
}

func provideUpdateTrailerTool(
	trailers *trailerservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateTrailerTool(trailers, states)
}

func provideLocateTrailerTool(trailers *trailerservice.Service) serviceports.AgentTool {
	return newLocateTrailerTool(trailers)
}

func provideDeleteDocumentsTool(documents *documentservice.Service) serviceports.AgentTool {
	return newDeleteDocumentsTool(documents)
}

func provideRestoreDocumentVersionTool(
	documents *documentservice.Service,
) serviceports.AgentTool {
	return newRestoreDocumentVersionTool(documents)
}

func provideRestoreInsightTool(insights *insightservice.Service) serviceports.AgentTool {
	return newRestoreInsightTool(insights)
}

func provideUpdateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) serviceports.AgentTool {
	return newUpdateTableChangeAlertTool(alerts)
}

func provideSetTableChangeAlertStatusTool(
	alerts *tablechangealertservice.Service,
) serviceports.AgentTool {
	return newSetTableChangeAlertStatusTool(alerts)
}

func provideDeleteTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) serviceports.AgentTool {
	return newDeleteTableChangeAlertTool(alerts)
}

func provideDismissWatchtowerItemTool(
	items *watchtowerservice.Dismisser,
) serviceports.AgentTool {
	return newDismissWatchtowerItemTool(items)
}

func provideFileCaptureItemsTool(captures *captureservice.Service) serviceports.AgentTool {
	return newFileCaptureItemsTool(captures)
}

func provideDiscardCaptureItemTool(captures *captureservice.Service) serviceports.AgentTool {
	return newDiscardCaptureItemTool(captures)
}

func provideDiscardCaptureBatchTool(captures *captureservice.Service) serviceports.AgentTool {
	return newDiscardCaptureBatchTool(captures)
}

func provideCreateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newCreateCarrierCapacityPostingTool(postings, states)
}

func provideUpdateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newUpdateCarrierCapacityPostingTool(postings, states)
}

func provideDeleteCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
) serviceports.AgentTool {
	return newDeleteCarrierCapacityPostingTool(postings)
}
