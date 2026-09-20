package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-query-tool-service",
	fx.Provide(
		fx.Annotate(newGetShipmentTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchShipmentsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetWorkerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchWorkerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newListExpiringCredentialsTool, fx.ResultTags(`group:"agent_query_tools"`)),
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
		fx.Annotate(provideListReportsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideRunReportTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(provideGetReportRunTool, fx.ResultTags(`group:"agent_query_tools"`)),
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
