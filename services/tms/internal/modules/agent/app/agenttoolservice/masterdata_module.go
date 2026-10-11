package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carriercapacityservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
)

var _ masterdatatools.CarrierKeeper = (*carrierservice.Service)(nil)

var _ masterdatatools.CustomerKeeper = (*customerservice.Service)(nil)

var _ masterdatatools.CommodityKeeper = (*commodityservice.Service)(nil)

var _ masterdatatools.CapacityKeeper = (*carriercapacityservice.Service)(nil)

var _ masterdatatools.HazmatKeeper = (*hazardousmaterialservice.Service)(nil)

var _ masterdatatools.LocationKeeper = (*locationservice.Service)(nil)

var _ masterdatatools.TractorKeeper = (*tractorservice.Service)(nil)

var _ masterdatatools.TrailerKeeper = (*trailerservice.Service)(nil)

var _ receivabletools.DocumentKeeper = (*documentservice.Service)(nil)

var _ insighttools.InsightRestorer = (*insightservice.Service)(nil)

var _ alerttools.AlertKeeper = (*tablechangealertservice.Service)(nil)

var _ receivabletools.WatchtowerDismisser = (*watchtowerservice.Dismisser)(nil)

var _ receivabletools.CaptureKeeper = (*captureservice.Service)(nil)

var _ masterdatatools.StateLookup = usstateports.UsStateRepository(nil)

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
	return masterdatatools.NewCreateCarrierTool(carriers, states)
}

func provideUpdateCarrierTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCarrierTool(carriers, states)
}

func provideUpdateCarrierStatusTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCarrierStatusTool(carriers, states)
}

func provideCreateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewCreateCustomerTool(customers, states)
}

func provideUpdateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCustomerTool(customers, states)
}

func provideUpdateCustomerStatusTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCustomerStatusTool(customers, states)
}

func provideCreateCommodityTool(commodities *commodityservice.Service) agentports.AgentTool {
	return masterdatatools.NewCreateCommodityTool(commodities)
}

func provideUpdateCommodityTool(commodities *commodityservice.Service) agentports.AgentTool {
	return masterdatatools.NewUpdateCommodityTool(commodities)
}

func provideUpdateCommodityStatusTool(
	commodities *commodityservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCommodityStatusTool(commodities)
}

func provideCreateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewCreateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewUpdateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialStatusTool(
	materials *hazardousmaterialservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewUpdateHazardousMaterialStatusTool(materials)
}

func provideUpdateLocationTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateLocationTool(locations, states)
}

func provideUpdateLocationStatusTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateLocationStatusTool(locations, states)
}

func provideCreateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewCreateTractorTool(tractors, states)
}

func provideUpdateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateTractorTool(tractors, states)
}

func provideLocateTractorTool(tractors *tractorservice.Service) agentports.AgentTool {
	return masterdatatools.NewLocateTractorTool(tractors)
}

func provideCreateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewCreateTrailerTool(trailers, states)
}

func provideUpdateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateTrailerTool(trailers, states)
}

func provideLocateTrailerTool(trailers *trailerservice.Service) agentports.AgentTool {
	return masterdatatools.NewLocateTrailerTool(trailers)
}

func provideDeleteDocumentsTool(documents *documentservice.Service) agentports.AgentTool {
	return receivabletools.NewDeleteDocumentsTool(documents)
}

func provideRestoreDocumentVersionTool(
	documents *documentservice.Service,
) agentports.AgentTool {
	return receivabletools.NewRestoreDocumentVersionTool(documents)
}

func provideRestoreInsightTool(insights *insightservice.Service) agentports.AgentTool {
	return insighttools.NewRestoreInsightTool(insights)
}

func provideUpdateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return alerttools.NewUpdateTableChangeAlertTool(alerts)
}

func provideSetTableChangeAlertStatusTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return alerttools.NewSetTableChangeAlertStatusTool(alerts)
}

func provideDeleteTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return alerttools.NewDeleteTableChangeAlertTool(alerts)
}

func provideDismissWatchtowerItemTool(
	items *watchtowerservice.Dismisser,
) agentports.AgentTool {
	return receivabletools.NewDismissWatchtowerItemTool(items)
}

func provideFileCaptureItemsTool(captures *captureservice.Service) agentports.AgentTool {
	return receivabletools.NewFileCaptureItemsTool(captures)
}

func provideDiscardCaptureItemTool(captures *captureservice.Service) agentports.AgentTool {
	return receivabletools.NewDiscardCaptureItemTool(captures)
}

func provideDiscardCaptureBatchTool(captures *captureservice.Service) agentports.AgentTool {
	return receivabletools.NewDiscardCaptureBatchTool(captures)
}

func provideCreateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewCreateCarrierCapacityPostingTool(postings, states)
}

func provideUpdateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return masterdatatools.NewUpdateCarrierCapacityPostingTool(postings, states)
}

func provideDeleteCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewDeleteCarrierCapacityPostingTool(postings)
}
