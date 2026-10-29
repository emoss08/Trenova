package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/commoditytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/documenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/equipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/exceptiontools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/locationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/partytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
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

var _ partytools.CarrierKeeper = (*carrierservice.Service)(nil)

var _ partytools.CustomerKeeper = (*customerservice.Service)(nil)

var _ commoditytools.CommodityKeeper = (*commodityservice.Service)(nil)

var _ partytools.CapacityKeeper = (*carriercapacityservice.Service)(nil)

var _ commoditytools.HazmatKeeper = (*hazardousmaterialservice.Service)(nil)

var _ locationtools.LocationKeeper = (*locationservice.Service)(nil)

var _ equipmenttools.TractorKeeper = (*tractorservice.Service)(nil)

var _ equipmenttools.TrailerKeeper = (*trailerservice.Service)(nil)

var _ documenttools.DocumentKeeper = (*documentservice.Service)(nil)

var _ insighttools.InsightRestorer = (*insightservice.Service)(nil)

var _ alerttools.AlertKeeper = (*tablechangealertservice.Service)(nil)

var _ exceptiontools.WatchtowerDismisser = (*watchtowerservice.Dismisser)(nil)

var _ documenttools.CaptureKeeper = (*captureservice.Service)(nil)

var _ masterdatakit.StateLookup = usstateports.UsStateRepository(nil)

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
		provideAssignExceptionTool,
		provideSnoozeExceptionTool,
		provideSetExceptionDueTool,
		provideResolveExceptionTool,
		provideReopenExceptionTool,
		provideCommentOnExceptionTool,
		provideFileCaptureItemsTool,
		provideDiscardCaptureItemTool,
		provideDiscardCaptureBatchTool,
	}
}

func provideCreateCarrierTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewCreateCarrierTool(carriers, states)
}

func provideUpdateCarrierTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewUpdateCarrierTool(carriers, states)
}

func provideUpdateCarrierStatusTool(
	carriers *carrierservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewUpdateCarrierStatusTool(carriers, states)
}

func provideCorrectChargeCodeTool(
	billing billingqueueports.BillingQueueService,
	customers *customerservice.Service,
) agenttoolports.AgentTool {
	return invoicetools.NewCorrectChargeCodeTool(billing, customers)
}

func provideCreateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewCreateCustomerTool(customers, states)
}

func provideUpdateCustomerTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewUpdateCustomerTool(customers, states)
}

func provideUpdateCustomerStatusTool(
	customers *customerservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewUpdateCustomerStatusTool(customers, states)
}

func provideCreateCommodityTool(commodities *commodityservice.Service) agenttoolports.AgentTool {
	return commoditytools.NewCreateCommodityTool(commodities)
}

func provideUpdateCommodityTool(commodities *commodityservice.Service) agenttoolports.AgentTool {
	return commoditytools.NewUpdateCommodityTool(commodities)
}

func provideUpdateCommodityStatusTool(
	commodities *commodityservice.Service,
) agenttoolports.AgentTool {
	return commoditytools.NewUpdateCommodityStatusTool(commodities)
}

func provideCreateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agenttoolports.AgentTool {
	return commoditytools.NewCreateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialTool(
	materials *hazardousmaterialservice.Service,
) agenttoolports.AgentTool {
	return commoditytools.NewUpdateHazardousMaterialTool(materials)
}

func provideUpdateHazardousMaterialStatusTool(
	materials *hazardousmaterialservice.Service,
) agenttoolports.AgentTool {
	return commoditytools.NewUpdateHazardousMaterialStatusTool(materials)
}

func provideUpdateLocationTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return locationtools.NewUpdateLocationTool(locations, states)
}

func provideUpdateLocationStatusTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return locationtools.NewUpdateLocationStatusTool(locations, states)
}

func provideCreateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return equipmenttools.NewCreateTractorTool(tractors, states)
}

func provideUpdateTractorTool(
	tractors *tractorservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return equipmenttools.NewUpdateTractorTool(tractors, states)
}

func provideLocateTractorTool(tractors *tractorservice.Service) agenttoolports.AgentTool {
	return equipmenttools.NewLocateTractorTool(tractors)
}

func provideCreateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return equipmenttools.NewCreateTrailerTool(trailers, states)
}

func provideUpdateTrailerTool(
	trailers *trailerservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return equipmenttools.NewUpdateTrailerTool(trailers, states)
}

func provideLocateTrailerTool(trailers *trailerservice.Service) agenttoolports.AgentTool {
	return equipmenttools.NewLocateTrailerTool(trailers)
}

func provideDeleteDocumentsTool(documents *documentservice.Service) agenttoolports.AgentTool {
	return documenttools.NewDeleteDocumentsTool(documents)
}

func provideRestoreDocumentVersionTool(
	documents *documentservice.Service,
) agenttoolports.AgentTool {
	return documenttools.NewRestoreDocumentVersionTool(documents)
}

func provideRestoreInsightTool(insights *insightservice.Service) agenttoolports.AgentTool {
	return insighttools.NewRestoreInsightTool(insights)
}

func provideUpdateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agenttoolports.AgentTool {
	return alerttools.NewUpdateTableChangeAlertTool(alerts)
}

func provideSetTableChangeAlertStatusTool(
	alerts *tablechangealertservice.Service,
) agenttoolports.AgentTool {
	return alerttools.NewSetTableChangeAlertStatusTool(alerts)
}

func provideDeleteTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agenttoolports.AgentTool {
	return alerttools.NewDeleteTableChangeAlertTool(alerts)
}

func provideAssignExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewAssignExceptionTool(exceptions)
}

func provideSnoozeExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewSnoozeExceptionTool(exceptions)
}

func provideSetExceptionDueTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewSetExceptionDueTool(exceptions)
}

func provideResolveExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewResolveExceptionTool(exceptions)
}

func provideReopenExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewReopenExceptionTool(exceptions)
}

func provideCommentOnExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewCommentOnExceptionTool(exceptions)
}

func provideDismissWatchtowerItemTool(
	items *watchtowerservice.Dismisser,
) agenttoolports.AgentTool {
	return exceptiontools.NewDismissWatchtowerItemTool(items)
}

func provideFileCaptureItemsTool(captures *captureservice.Service) agenttoolports.AgentTool {
	return documenttools.NewFileCaptureItemsTool(captures)
}

func provideDiscardCaptureItemTool(captures *captureservice.Service) agenttoolports.AgentTool {
	return documenttools.NewDiscardCaptureItemTool(captures)
}

func provideDiscardCaptureBatchTool(captures *captureservice.Service) agenttoolports.AgentTool {
	return documenttools.NewDiscardCaptureBatchTool(captures)
}

func provideCreateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewCreateCarrierCapacityPostingTool(postings, states)
}

func provideUpdateCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return partytools.NewUpdateCarrierCapacityPostingTool(postings, states)
}

func provideDeleteCarrierCapacityPostingTool(
	postings *carriercapacityservice.Service,
) agenttoolports.AgentTool {
	return partytools.NewDeleteCarrierCapacityPostingTool(postings)
}
