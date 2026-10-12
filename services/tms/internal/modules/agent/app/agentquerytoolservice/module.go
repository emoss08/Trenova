package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
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
		ledgerToolProviders(),
		settlementToolProviders(),
		receivableToolProviders(),
		driverPayToolProviders(),
		ratingToolProviders(),
		orderToolProviders(),
		ediToolProviders(),
		oversightToolProviders(),
		importDraftToolProviders(),
		formulaToolProviders(),
		tenderingToolProviders(),
		operationsQueryToolProviders(),
		fuelRateReadProviders(),
		workforceQueryToolProviders(),
		masterDataReadProviders(),
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
		provideListAccountingDriftFindingsTool,
		provideListAccountingMappingGapsTool,
		provideGetAccountingMappingTool,
		newGetShipmentTool,
		provideSearchShipmentsTool,
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
		newListCarrierCapacityPostingsTool,
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
		newGetInvoicesTool,
		newListBillingQueueItemsTool,
		newGetBillingQueueItemTool,
		newGetBillingQueueItemsTool,
		newListBillingTransferCandidatesTool,
		provideGetDetentionOccurrenceTool,
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
		provideReadAttachmentTool,
		provideFindShipmentRecordsTool,
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

func provideListReportsTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newListReportsTool(reports)
}

func provideRunReportTool(
	reports *reporting.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newRunReportTool(reports, permissions)
}

func provideGetReportRunTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newGetReportRunTool(reports)
}

func provideListReportRunsTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newListReportRunsTool(reports)
}

func provideCompareReportRunsTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newCompareReportRunsTool(reports)
}

func provideDescribeReportTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newDescribeReportTool(reports)
}

func provideListReportDatasetsTool(
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListReportDatasetsTool(permissions)
}

func provideDescribeReportDatasetTool(
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newDescribeReportDatasetTool(permissions)
}

func providePreviewReportTool(reports *reporting.Service) agentports.AgentQueryTool {
	return newPreviewReportTool(reports)
}

func provideListTimeOffTool(pto workerports.WorkerPTOService) agentports.AgentQueryTool {
	return newListTimeOffTool(pto)
}

// The monitoring tools read the board, the telematics feed, service failures,
// the detention desk and the weather feed through the narrow interfaces each
// declares; the concrete services are what the graph holds.

func provideGetShipmentTrackingTool(
	shipments shipmentports.ShipmentRepository,
	console shipmentports.DispatchConsoleRepository,
	telematics *telematicsservice.Service,
) agentports.AgentQueryTool {
	return newGetShipmentTrackingTool(shipments, console, telematics)
}

func provideListVehiclePositionsTool(
	telematics *telematicsservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListVehiclePositionsTool(telematics, permissions)
}

func provideGetWorkerHOSTool(telematics *telematicsservice.Service) agentports.AgentQueryTool {
	return newGetWorkerHOSTool(telematics)
}

func provideGetDispatchBoardTool(
	board agentports.DispatchConsoleService,
) agentports.AgentQueryTool {
	return newGetDispatchBoardTool(board)
}

func provideListServiceFailuresTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentQueryTool {
	return newListServiceFailuresTool(failures)
}

func provideListServiceFailureReasonCodesTool(
	codes servicefailureports.ServiceFailureReasonCodeService,
) agentports.AgentQueryTool {
	return newListServiceFailureReasonCodesTool(codes)
}

func provideListDetentionDeskTool(desk *detentionservice.Service) agentports.AgentQueryTool {
	return newListDetentionDeskTool(desk)
}

func provideListWeatherAlertsTool(
	weather weatheralertports.WeatherAlertService,
) agentports.AgentQueryTool {
	return newListWeatherAlertsTool(weather)
}

func provideListEmailProfilesTool(profiles *emailservice.Service) agentports.AgentQueryTool {
	return newListEmailProfilesTool(profiles)
}

func provideGetDocumentSummaryTool(
	documents documentports.DocumentRepository,
	content documentports.DocumentContentService,
	permissions permissionports.PermissionEngine,
	threads agentports.ThreadOwnerRepository,
) agentports.AgentQueryTool {
	return newGetDocumentSummaryTool(documents, content, permissions, threads)
}

//nolint:gocritic // fx.In parameter structs are passed by value
func provideFindShipmentRecordsTool(p ShipmentRecordsToolParams) agentports.AgentQueryTool {
	return newFindShipmentRecordsTool(&p)
}

func provideReadAttachmentTool(
	documents documentports.DocumentRepository,
	content documentports.DocumentContentService,
) agentports.AgentQueryTool {
	return newReadAttachmentTool(documents, content)
}

func provideGetShipmentDraftTool(
	content documentports.DocumentContentService,
) agentports.AgentQueryTool {
	return newGetShipmentDraftTool(content)
}

func provideQuoteShipmentTool(
	quotes *ratequoteservice.Service,
	locations locationports.LocationRepository,
) agentports.AgentQueryTool {
	return newQuoteShipmentTool(quotes, locations)
}

func provideShopCarriersTool(
	quotes *ratequoteservice.Service,
	carriers carrierports.CarrierRepository,
) agentports.AgentQueryTool {
	return newShopCarriersTool(quotes, carriers)
}

func provideRankMoveCandidatesTool(
	console agentports.DispatchConsoleService,
) agentports.AgentQueryTool {
	return newRankMoveCandidatesTool(console)
}

func providePlanDispatchTool(
	planner agentports.DispatchAutoAssignService,
) agentports.AgentQueryTool {
	return newPlanDispatchTool(planner)
}

func provideListInsightsTool(insights *insightservice.Service) agentports.AgentQueryTool {
	return newListInsightsTool(insights)
}

func provideGetInsightTool(insights *insightservice.Service) agentports.AgentQueryTool {
	return newGetInsightTool(insights)
}

func provideListBankReceiptExceptionsTool(
	receipts *bankreceiptservice.Service,
	items bankreceiptworkitemports.BankReceiptWorkItemRepository,
) agentports.AgentQueryTool {
	return newListBankReceiptExceptionsTool(receipts, items)
}

func provideGetBankReceiptTool(
	receipts *bankreceiptservice.Service,
	items bankreceiptworkitemports.BankReceiptWorkItemRepository,
) agentports.AgentQueryTool {
	return newGetBankReceiptTool(receipts, items)
}

func provideListCustomerPaymentsTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentQueryTool {
	return newListCustomerPaymentsTool(payments)
}

func provideGetWorkerCredentialTool(
	repo workerports.WorkerCredentialRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetWorkerCredentialTool(repo, permissions)
}

func provideGetWorkerTool(
	repo workerports.WorkerRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetWorkerTool(repo, permissions)
}

func provideGetInboundMessageTool(
	messages *inboundmessageservice.Service,
) agentports.AgentQueryTool {
	return newGetInboundMessageTool(messages)
}

func provideListInboundMessagesTool(
	messages *inboundmessageservice.Service,
) agentports.AgentQueryTool {
	return newListInboundMessagesTool(messages)
}

func provideSearchDocumentsTool(
	searcher agentports.RetrievalSearcher,
	permissions permissionports.PermissionEngine,
	threads agentports.ThreadOwnerRepository,
) agentports.AgentQueryTool {
	return newSearchDocumentsTool(searcher, permissions, threads)
}

func provideSearchInboundMessagesTool(
	searcher agentports.RetrievalSearcher,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newSearchInboundMessagesTool(searcher, permissions)
}
