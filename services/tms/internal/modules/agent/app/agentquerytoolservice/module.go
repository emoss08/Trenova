// Package agentquerytoolservice registers the read-only tools the assistant may run
// without asking a human first.
//
// Everything here is read-only by construction rather than by convention: these
// types implement AgentQueryTool, which has no Execute method. A tool that needs
// to change something implements AgentTool instead and goes through the proposal
// and approval path.
package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/accountingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/agentrunquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/billingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/dispatchquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/documentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ediquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/formulaquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fuelquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/guidequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/homequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/importdraftquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/inboxquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/insightquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/intakequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/memoryquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/oversightquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ratequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/receivablequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/reportquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/scriptquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentreadquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentrecordquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/tableviewquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/webquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workforcequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
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
		accountingquerytools.Providers(),
		receivablequerytools.Providers(),
		ediquerytools.Providers(),
		oversightquerytools.Providers(),
		importdraftquerytools.Providers(),
		formulaquerytools.Providers(),
		operationsquerytools.TenderingToolProviders(),
		operationsquerytools.Providers(),
		fuelquerytools.RateReadProviders(),
		workforcequerytools.Providers(),
		masterdataquerytools.ReadProviders(),
		graphQLListProviders(),
		graphQLGetProviders(),
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
		shipmentreadquerytools.ProvideGetShipmentTool,
		shipmentquerytools.ProvideShipmentListTool,
		memoryquerytools.NewRecallMemoryTool,
		tableviewquerytools.NewComposeTableViewTool,
		ratequerytools.NewExplainRateTool,
		tableviewquerytools.NewListDashboardsTool,
		billingquerytools.NewListBillingQueueItemsTool,
		billingquerytools.NewGetBillingQueueItemTool,
		billingquerytools.NewListBillingTransferCandidatesTool,
		agentrunquerytools.NewGetAgentRunTool,
		provideListReportsTool,
		provideRunReportTool,
		provideGetReportRunTool,
		provideCompareReportRunsTool,
		provideDescribeReportTool,
		provideListReportDatasetsTool,
		provideDescribeReportDatasetTool,
		providePreviewReportTool,
		provideGetShipmentTrackingTool,
		provideGetDispatchBoardTool,
		provideListWeatherAlertsTool,
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
		inboxquerytools.ProvideListInboundMessagesTool,
		provideSearchDocumentsTool,
		homequerytools.NewListHomeWidgetsTool,
		guidequerytools.NewFindInTrenovaTool,
		guidequerytools.NewOpenPageTool,
		webquerytools.NewWebSearchTool,
		webquerytools.NewWebReadTool,
		webquerytools.NewListPortalSitesTool,
		webquerytools.NewReadPortalTool,
		webquerytools.NewListWebWatchesTool,
		provideRunScriptTool,
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

func provideListReportsTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewListReportsTool(reports)
}

func provideRunReportTool(
	reports *reporting.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return reportquerytools.NewRunReportTool(reports, permissions)
}

func provideRunScriptTool(
	reports *reporting.Service,
	runner scriptports.Runner,
	files documentports.ConversationFileReader,
	permissions permissionports.PermissionEngine,
	analyses analysisports.RunRecorder,
) agenttoolports.AgentQueryTool {
	return scriptquerytools.NewRunScriptTool(&scriptquerytools.RunScriptToolParams{
		Runner:      runner,
		Reports:     reports,
		Files:       files,
		Permissions: permissions,
		Analyses:    analyses,
	})
}

func provideGetReportRunTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewGetReportRunTool(reports)
}

func provideCompareReportRunsTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewCompareReportRunsTool(reports)
}

func provideDescribeReportTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewDescribeReportTool(reports)
}

func provideListReportDatasetsTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewListReportDatasetsTool(reports)
}

func provideDescribeReportDatasetTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewDescribeReportDatasetTool(reports)
}

func providePreviewReportTool(reports *reporting.Service) agenttoolports.AgentQueryTool {
	return reportquerytools.NewPreviewReportTool(reports)
}

// The monitoring tools read the board, the telematics feed and the weather
// feed through the narrow interfaces each declares; the concrete services are
// what the graph holds.

func provideGetShipmentTrackingTool(
	shipments shipmentports.ShipmentRepository,
	console dispatchboardports.DispatchConsoleRepository,
	telematics *telematicsservice.Service,
	stopEtas stopetaports.Repository,
) agenttoolports.AgentQueryTool {
	return shipmentreadquerytools.NewGetShipmentTrackingTool(
		shipments,
		console,
		telematics,
		stopEtas,
	)
}

func provideGetDispatchBoardTool(
	board dispatchconsoleports.DispatchConsoleService,
) agenttoolports.AgentQueryTool {
	return dispatchquerytools.NewGetDispatchBoardTool(board)
}

func provideListWeatherAlertsTool(
	weather weatheralertports.WeatherAlertService,
) agenttoolports.AgentQueryTool {
	return dispatchquerytools.NewListWeatherAlertsTool(weather)
}

func provideGetDocumentSummaryTool(
	documents documentports.DocumentRepository,
	content documentports.DocumentContentService,
	permissions permissionports.PermissionEngine,
	threads conversationports.ThreadOwnerRepository,
) agenttoolports.AgentQueryTool {
	return documentquerytools.NewGetDocumentSummaryTool(documents, content, permissions, threads)
}

//
//nolint:gocritic // fx.In parameter structs are passed by value
func provideFindShipmentRecordsTool(
	p shipmentrecordquerytools.ShipmentRecordsToolParams,
) agenttoolports.AgentQueryTool {
	return shipmentrecordquerytools.NewFindShipmentRecordsTool(&p)
}

func provideReadAttachmentTool(
	documents documentports.DocumentRepository,
	content documentports.DocumentContentService,
) agenttoolports.AgentQueryTool {
	return documentquerytools.NewReadAttachmentTool(documents, content)
}

func provideGetShipmentDraftTool(
	content documentports.DocumentContentService,
) agenttoolports.AgentQueryTool {
	return intakequerytools.NewGetShipmentDraftTool(content)
}

func provideQuoteShipmentTool(
	quotes *ratequoteservice.Service,
	locations locationports.LocationRepository,
) agenttoolports.AgentQueryTool {
	return intakequerytools.NewQuoteShipmentTool(quotes, locations)
}

func provideShopCarriersTool(
	quotes *ratequoteservice.Service,
	carriers carrierports.CarrierRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return intakequerytools.NewShopCarriersTool(quotes, carriers, permissions)
}

func provideRankMoveCandidatesTool(
	console dispatchconsoleports.DispatchConsoleService,
) agenttoolports.AgentQueryTool {
	return dispatchquerytools.NewRankMoveCandidatesTool(console)
}

func providePlanDispatchTool(
	planner dispatchconsoleports.DispatchAutoAssignService,
) agenttoolports.AgentQueryTool {
	return dispatchquerytools.NewPlanDispatchTool(planner)
}

func provideListInsightsTool(insights *insightservice.Service) agenttoolports.AgentQueryTool {
	return insightquerytools.NewListInsightsTool(insights)
}

func provideGetInsightTool(insights *insightservice.Service) agenttoolports.AgentQueryTool {
	return insightquerytools.NewGetInsightTool(insights)
}

func provideListBankReceiptExceptionsTool(
	receipts *bankreceiptservice.Service,
	items bankreceiptworkitemports.BankReceiptWorkItemRepository,
) agenttoolports.AgentQueryTool {
	return receivablequerytools.NewListBankReceiptExceptionsTool(receipts, items)
}

func provideGetBankReceiptTool(
	receipts *bankreceiptservice.Service,
	items bankreceiptworkitemports.BankReceiptWorkItemRepository,
) agenttoolports.AgentQueryTool {
	return receivablequerytools.NewGetBankReceiptTool(receipts, items)
}

func provideSearchDocumentsTool(
	searcher retrievalports.RetrievalSearcher,
	permissions permissionports.PermissionEngine,
	threads conversationports.ThreadOwnerRepository,
) agenttoolports.AgentQueryTool {
	return documentquerytools.NewSearchDocumentsTool(searcher, permissions, threads)
}
