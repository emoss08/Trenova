package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-query-tool-service",
	fx.Provide(
		fx.Annotate(newGetShipmentTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchShipmentsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetWorkerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchWorkerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListExpiringCredentialsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newRecallMemoryTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newComposeTableViewTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newExplainRateTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListWorkersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListShipmentsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListTractorsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListTrailersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListCustomersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListLocationsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListInvoicesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListCarriersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListEquipmentTypesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListFleetCodesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListServiceTypesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListShipmentTypesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListCommoditiesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListHazardousMaterialsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListAccessorialChargesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListDocumentTypesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListLocationCategoriesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListHoldReasonsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetCustomerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetCarrierTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetTractorTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetTrailerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetInvoiceTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetDetentionOccurrenceTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetCarrierIntelEventTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetCustomerUpdatePreferencesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetAgentRunTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetServiceFailureTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetWorkerCredentialTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListTimeOffTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListReportsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideRunReportTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetReportRunTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideDescribeReportTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListReportDatasetsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideDescribeReportDatasetTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(providePreviewReportTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetShipmentTrackingTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListVehiclePositionsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetWorkerHOSTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetDispatchBoardTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListServiceFailuresTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListServiceFailureReasonCodesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListDetentionDeskTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListWeatherAlertsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListEmailProfilesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetShipmentDraftTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetDocumentSummaryTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideQuoteShipmentTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideShopCarriersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideRankMoveCandidatesTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(providePlanDispatchTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListInsightsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetInsightTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListBankReceiptExceptionsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetBankReceiptTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideListCustomerPaymentsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		NewRegistry,
	),
)

// The report tools take the narrow reportRunner interface so they can be tested
// without the whole reporting stack; fx holds the concrete service, so the
// widening happens here rather than in the tools.

func provideListReportsTool(reports *reporting.Service) services.AgentQueryTool {
	return newListReportsTool(reports)
}

func provideRunReportTool(
	reports *reporting.Service,
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newRunReportTool(reports, permissions)
}

func provideGetReportRunTool(reports *reporting.Service) services.AgentQueryTool {
	return newGetReportRunTool(reports)
}

func provideDescribeReportTool(reports *reporting.Service) services.AgentQueryTool {
	return newDescribeReportTool(reports)
}

func provideListReportDatasetsTool(permissions services.PermissionEngine) services.AgentQueryTool {
	return newListReportDatasetsTool(permissions)
}

func provideDescribeReportDatasetTool(
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newDescribeReportDatasetTool(permissions)
}

func providePreviewReportTool(reports *reporting.Service) services.AgentQueryTool {
	return newPreviewReportTool(reports)
}

func provideListTimeOffTool(pto services.WorkerPTOService) services.AgentQueryTool {
	return newListTimeOffTool(pto)
}

// The monitoring tools read the board, the telematics feed, service failures,
// the detention desk and the weather feed through the narrow interfaces each
// declares; the concrete services are what the graph holds.

func provideGetShipmentTrackingTool(
	shipments repositories.ShipmentRepository,
	console repositories.DispatchConsoleRepository,
	telematics *telematicsservice.Service,
) services.AgentQueryTool {
	return newGetShipmentTrackingTool(shipments, console, telematics)
}

func provideListVehiclePositionsTool(
	telematics *telematicsservice.Service,
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newListVehiclePositionsTool(telematics, permissions)
}

func provideGetWorkerHOSTool(telematics *telematicsservice.Service) services.AgentQueryTool {
	return newGetWorkerHOSTool(telematics)
}

func provideGetDispatchBoardTool(board services.DispatchConsoleService) services.AgentQueryTool {
	return newGetDispatchBoardTool(board)
}

func provideListServiceFailuresTool(
	failures services.ServiceFailureService,
) services.AgentQueryTool {
	return newListServiceFailuresTool(failures)
}

func provideListServiceFailureReasonCodesTool(
	codes services.ServiceFailureReasonCodeService,
) services.AgentQueryTool {
	return newListServiceFailureReasonCodesTool(codes)
}

func provideListDetentionDeskTool(desk *detentionservice.Service) services.AgentQueryTool {
	return newListDetentionDeskTool(desk)
}

func provideListWeatherAlertsTool(weather services.WeatherAlertService) services.AgentQueryTool {
	return newListWeatherAlertsTool(weather)
}

func provideListEmailProfilesTool(profiles *emailservice.Service) services.AgentQueryTool {
	return newListEmailProfilesTool(profiles)
}

func provideGetDocumentSummaryTool(
	documents repositories.DocumentRepository,
	content services.DocumentContentService,
) services.AgentQueryTool {
	return newGetDocumentSummaryTool(documents, content)
}

func provideGetShipmentDraftTool(content services.DocumentContentService) services.AgentQueryTool {
	return newGetShipmentDraftTool(content)
}

func provideQuoteShipmentTool(
	quotes *ratequoteservice.Service,
	locations repositories.LocationRepository,
) services.AgentQueryTool {
	return newQuoteShipmentTool(quotes, locations)
}

func provideShopCarriersTool(quotes *ratequoteservice.Service) services.AgentQueryTool {
	return newShopCarriersTool(quotes)
}

func provideRankMoveCandidatesTool(console services.DispatchConsoleService) services.AgentQueryTool {
	return newRankMoveCandidatesTool(console)
}

func providePlanDispatchTool(planner services.DispatchAutoAssignService) services.AgentQueryTool {
	return newPlanDispatchTool(planner)
}

func provideListInsightsTool(insights *insightservice.Service) services.AgentQueryTool {
	return newListInsightsTool(insights)
}

func provideGetInsightTool(insights *insightservice.Service) services.AgentQueryTool {
	return newGetInsightTool(insights)
}

func provideListBankReceiptExceptionsTool(
	receipts *bankreceiptservice.Service,
	items repositories.BankReceiptWorkItemRepository,
) services.AgentQueryTool {
	return newListBankReceiptExceptionsTool(receipts, items)
}

func provideGetBankReceiptTool(
	receipts *bankreceiptservice.Service,
	items repositories.BankReceiptWorkItemRepository,
) services.AgentQueryTool {
	return newGetBankReceiptTool(receipts, items)
}

func provideListCustomerPaymentsTool(payments services.CustomerPaymentService) services.AgentQueryTool {
	return newListCustomerPaymentsTool(payments)
}

func provideGetWorkerCredentialTool(
	repo repositories.WorkerCredentialRepository,
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newGetWorkerCredentialTool(repo, permissions)
}

func provideGetWorkerTool(
	repo repositories.WorkerRepository,
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newGetWorkerTool(repo, permissions)
}
