package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carriercapacityservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
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
	_ stateLookup         = usstateports.UsStateRepository(nil)
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
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newCreateCarrierTool(carriers, states)
}

func provideUpdateCarrierTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateCarrierTool(carriers, states)
}

func provideUpdateCarrierStatusTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateCarrierStatusTool(carriers, states)
}

func provideCreateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newCreateCustomerTool(customers, states)
}

func provideUpdateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateCustomerTool(customers, states)
}

func provideUpdateCustomerStatusTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateCustomerStatusTool(customers, states)
}

func provideCreateCommodityTool(commodities *commodityservice.Service) agentports.AgentTool {
	return newCreateCommodityTool(commodities)
}

func provideUpdateCommodityTool(commodities *commodityservice.Service) agentports.AgentTool {
	return newUpdateCommodityTool(commodities)
}

func provideUpdateCommodityStatusTool(
	commodities *commodityservice.Service,
) agentports.AgentTool {
	return newUpdateCommodityStatusTool(commodities)
}

func provideCreateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return newCreateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return newUpdateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialStatusTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return newUpdateHazardousMaterialStatusTool(materials)
}

func provideUpdateLocationTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateLocationTool(locations, states)
}

func provideUpdateLocationStatusTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateLocationStatusTool(locations, states)
}

func provideCreateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newCreateTractorTool(tractors, states)
}

func provideUpdateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateTractorTool(tractors, states)
}

func provideLocateTractorTool(tractors *tractorservice.Service) agentports.AgentTool {
	return newLocateTractorTool(tractors)
}

func provideCreateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newCreateTrailerTool(trailers, states)
}

func provideUpdateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateTrailerTool(trailers, states)
}

func provideLocateTrailerTool(trailers *trailerservice.Service) agentports.AgentTool {
	return newLocateTrailerTool(trailers)
}

func provideDeleteDocumentsTool(documents *documentservice.Service) agentports.AgentTool {
	return newDeleteDocumentsTool(documents)
}

func provideRestoreDocumentVersionTool(
	documents *documentservice.Service,
) agentports.AgentTool {
	return newRestoreDocumentVersionTool(documents)
}

func provideRestoreInsightTool(insights *insightservice.Service) agentports.AgentTool {
	return newRestoreInsightTool(insights)
}

func provideUpdateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return newUpdateTableChangeAlertTool(alerts)
}

func provideSetTableChangeAlertStatusTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return newSetTableChangeAlertStatusTool(alerts)
}

func provideDeleteTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return newDeleteTableChangeAlertTool(alerts)
}

func provideDismissWatchtowerItemTool(
	items *watchtowerservice.Dismisser,
) agentports.AgentTool {
	return newDismissWatchtowerItemTool(items)
}

func provideFileCaptureItemsTool(captures *captureservice.Service) agentports.AgentTool {
	return newFileCaptureItemsTool(captures)
}

func provideDiscardCaptureItemTool(captures *captureservice.Service) agentports.AgentTool {
	return newDiscardCaptureItemTool(captures)
}

func provideDiscardCaptureBatchTool(captures *captureservice.Service) agentports.AgentTool {
	return newDiscardCaptureBatchTool(captures)
}

func provideCreateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newCreateCarrierCapacityPostingTool(postings, states)
}

func provideUpdateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newUpdateCarrierCapacityPostingTool(postings, states)
}

func provideDeleteCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
) agentports.AgentTool {
	return newDeleteCarrierCapacityPostingTool(postings)
}
