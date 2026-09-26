package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-query-tool-service", fx.Provide(append(grouped(), NewRegistry)...))

// ToolProviders are the constructors of every tool this package registers.
// The module provides them into the group, and the description contract test
// builds each one to read what a model is shown.
func ToolProviders() []any {
	groups := [][]any{
		coreToolProviders(),
		accountingToolProviders(),
		settlementToolProviders(),
		ratingToolProviders(),
		orderToolProviders(),
		ediToolProviders(),
		oversightToolProviders(),
		importDraftToolProviders(),
		formulaToolProviders(),
		tenderingToolProviders(),
	}

	size := 0
	for _, group := range groups {
		size += len(group)
	}

	providers := make([]any, 0, size)
	for _, group := range groups {
		providers = append(providers, group...)
	}

	return providers
}

func coreToolProviders() []any {
	return []any{
		provideGetAccountingSyncStatusTool,
		provideListAccountingSyncRecordsTool,
		provideGetAccountingSyncRecordTool,
		provideGetRecordAccountingSyncStateTool,
		provideListAccountingInboundChangesTool,
		provideListAccountingMappingGapsTool,
		provideGetAccountingMappingTool,
		newGetShipmentTool,
		newSearchShipmentsTool,
		provideGetWorkerTool,
		newSearchWorkerTool,
		newListExpiringCredentialsTool,
		newRecallMemoryTool,
		newComposeTableViewTool,
		newExplainRateTool,
		newListDashboardsTool,
		newListWorkersTool,
		newListShipmentsTool,
		newListTractorsTool,
		newListTrailersTool,
		newListCustomersTool,
		newListLocationsTool,
		newListInvoicesTool,
		newListCarriersTool,
		newListEquipmentTypesTool,
		newListFleetCodesTool,
		newListServiceTypesTool,
		newListShipmentTypesTool,
		newListCommoditiesTool,
		newListHazardousMaterialsTool,
		newListAccessorialChargesTool,
		newListDocumentTypesTool,
		newListLocationCategoriesTool,
		newListHoldReasonsTool,
		newGetCustomerTool,
		newGetCarrierTool,
		newGetTractorTool,
		newGetTrailerTool,
		newGetInvoiceTool,
		newListBillingQueueItemsTool,
		newGetBillingQueueItemTool,
		newListBillingTransferCandidatesTool,
		newGetDetentionOccurrenceTool,
		newGetCarrierIntelEventTool,
		newGetCustomerUpdatePreferencesTool,
		newGetAgentRunTool,
		newGetServiceFailureTool,
		provideGetWorkerCredentialTool,
		provideListTimeOffTool,
		provideListReportsTool,
		provideRunReportTool,
		provideGetReportRunTool,
		provideListReportRunsTool,
		provideCompareReportRunsTool,
		provideDescribeReportTool,
		provideListReportDatasetsTool,
		provideDescribeReportDatasetTool,
		providePreviewReportTool,
		provideGetShipmentTrackingTool,
		provideListVehiclePositionsTool,
		provideGetWorkerHOSTool,
		provideGetDispatchBoardTool,
		provideListServiceFailuresTool,
		provideListServiceFailureReasonCodesTool,
		provideListDetentionDeskTool,
		provideListWeatherAlertsTool,
		provideListEmailProfilesTool,
		provideGetShipmentDraftTool,
		provideGetDocumentSummaryTool,
		provideQuoteShipmentTool,
		provideShopCarriersTool,
		provideRankMoveCandidatesTool,
		providePlanDispatchTool,
		provideListInsightsTool,
		provideGetInsightTool,
		provideListBankReceiptExceptionsTool,
		provideGetBankReceiptTool,
		provideListCustomerPaymentsTool,
		provideGetInboundMessageTool,
		provideListInboundMessagesTool,
		provideSearchDocumentsTool,
		provideSearchInboundMessagesTool,
		newGetMyHomeLayoutTool,
		newListHomeWidgetsTool,
		newFindInTrenovaTool,
		newOpenPageTool,
		newWebSearchTool,
		newWebReadTool,
	}
}

func grouped() []any {
	providers := ToolProviders()
	annotated := make([]any, 0, len(providers)+1)
	for _, provider := range providers {
		annotated = append(
			annotated,
			fx.Annotate(provider, fx.ResultTags(`group:"agent_query_tools"`)),
		)
	}

	return annotated
}

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

func provideListReportRunsTool(reports *reporting.Service) services.AgentQueryTool {
	return newListReportRunsTool(reports)
}

func provideCompareReportRunsTool(reports *reporting.Service) services.AgentQueryTool {
	return newCompareReportRunsTool(reports)
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
	permissions services.PermissionEngine,
	threads repositories.ThreadOwnerRepository,
) services.AgentQueryTool {
	return newGetDocumentSummaryTool(documents, content, permissions, threads)
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

func provideRankMoveCandidatesTool(
	console services.DispatchConsoleService,
) services.AgentQueryTool {
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

func provideListCustomerPaymentsTool(
	payments services.CustomerPaymentService,
) services.AgentQueryTool {
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

func provideGetInboundMessageTool(
	messages *inboundmessageservice.Service,
) services.AgentQueryTool {
	return newGetInboundMessageTool(messages)
}

func provideListInboundMessagesTool(
	messages *inboundmessageservice.Service,
) services.AgentQueryTool {
	return newListInboundMessagesTool(messages)
}

func provideSearchDocumentsTool(
	searcher services.RetrievalSearcher,
	permissions services.PermissionEngine,
	threads repositories.ThreadOwnerRepository,
) services.AgentQueryTool {
	return newSearchDocumentsTool(searcher, permissions, threads)
}

func provideSearchInboundMessagesTool(
	searcher services.RetrievalSearcher,
	permissions services.PermissionEngine,
) services.AgentQueryTool {
	return newSearchInboundMessagesTool(searcher, permissions)
}
