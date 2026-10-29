//revive:disable-next-line:var-naming
package api

import (
	graphqlapi "github.com/emoss08/trenova/internal/api/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/accountingloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/billingloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/documentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/ediloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/equipmentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/identityloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/kernelloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/operationsloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/partiesloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/placesloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/ratingloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/settlementloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/workforceloaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"go.uber.org/fx"
)

var graphQLLoaderModule = fx.Module("api-graphql-loaders", fx.Provide(
	loadercore.NewRegistry,
), loaderSources(
	agentloaders.NewFieldCeilingsFactory,
	identityloaders.NewOrganizationByIDLoaderFactory,
	placesloaders.NewLocationByIDLoaderFactory,
	operationsloaders.NewOrderByIDLoaderFactory,
	billingloaders.NewShipmentProfitabilityLoaderFactory,
	operationsloaders.NewShipmentEtaLoaderFactory,
	ediloaders.NewEDIPartnerByCustomerIDLoaderFactory,
	ratingloaders.NewFormulaTemplateStatsLoaderFactory,
	agentloaders.NewAgentDefinitionStatsLoaderFactory,
	settlementloaders.NewPayProfileActiveAssignmentCountLoaderFactory,
	workforceloaders.NewPTOPolicyOpenAssignmentCountLoaderFactory,
	workforceloaders.NewShiftTemplateActiveAssignmentCountLoaderFactory,
	agentloaders.NewAIProviderMonthSpendLoaderFactory,
	workforceloaders.NewWorkerChecklistTemplateOpenChecklistCountLoaderFactory,
	workforceloaders.NewWorkerCredentialTypeActiveCredentialCountLoaderFactory,
	workforceloaders.NewTrainingCourseOpenRecordCountLoaderFactory,
	workforceloaders.NewPerformanceReviewTemplateOpenReviewCountLoaderFactory,
	partiesloaders.NewCustomerByIDLoaderFactory,
	billingloaders.NewInvoiceByIDLoaderFactory,
	accountingloaders.NewGLAccountByIDLoaderFactory,
	operationsloaders.NewRoutingGuideWithEntriesByIDLoaderFactory,
	workforceloaders.NewWorkerDQFVerificationsByWorkerIDLoaderFactory,
	workforceloaders.NewWorkerLeaveEntriesByCaseIDLoaderFactory,
	workforceloaders.NewWorkerSafetyViolationsByEventIDLoaderFactory,
	accountingloaders.NewFiscalPeriodsByFiscalYearIDLoaderFactory,
	agentloaders.NewAgentDecisionsByProposalIDLoaderFactory,
	agentloaders.NewAgentPlanStepsByPlanIDLoaderFactory,
	agentloaders.NewSubsetLabelsLoaderFactory,
	agentloaders.NewAgentRunByIDLoaderFactory,
	agentloaders.NewSuggestedAgentByEventKindLoaderFactory,
	agentloaders.NewAgentRunTranscriptByIDLoaderFactory,
	agentloaders.NewAgentMemoryByIDLoaderFactory,
	agentloaders.NewAgentMemoryReplacementLoaderFactory,
	agentloaders.NewAgentDefinitionByIDLoaderFactory,
	agentloaders.NewAgentsPausedLoaderFactory,
	agentloaders.NewAgentPlanHoldLoaderFactory,
	agentloaders.NewThreadAgentByIDLoaderFactory,
	agentloaders.NewUsableAgentByIDLoaderFactory,
	agentloaders.NewAccessRolesByAgentIDLoaderFactory,
	agentloaders.NewAgentsByRoleIDLoaderFactory,
	agentloaders.NewToolTrustByAgentIDLoaderFactory,
	documentloaders.NewDocumentTemplateKindByTemplateIDLoaderFactory,
	settlementloaders.NewIFTAJurisdictionByIDLoaderFactory,
	settlementloaders.NewIFTAReturnByIDLoaderFactory,
	settlementloaders.NewFuelCardByIDLoaderFactory,
	equipmentloaders.NewTractorByIDLoaderFactory,
	ratingloaders.NewFormulaLibraryByIDLoaderFactory,
	ratingloaders.NewRateMatrixByIDLoaderFactory,
	ratingloaders.NewRateAgreementLaneCountLoaderFactory,
	ratingloaders.NewRateAgreementLanesLoaderFactory,
	operationsloaders.NewShipmentMoveJurisdictionMilesByMoveIDLoaderFactory,
	identityloaders.NewUserByIDLoaderFactory,
	documentloaders.NewDocumentByIDLoaderFactory,
	kernelloaders.NewUsStateByIDLoaderFactory,
	billingloaders.NewInvoicesByShipmentIDLoaderFactory,
	operationsloaders.NewChargeAllocationsByShipmentIDLoaderFactory,
	operationsloaders.NewChargeAllocationsByOrderChargeIDLoaderFactory,
	billingloaders.NewCustomerPaymentApplicationsByInvoiceIDLoaderFactory,
	billingloaders.NewCreditMemoApplicationsByInvoiceIDLoaderFactory,
	billingloaders.NewInvoiceDisputesByInvoiceIDLoaderFactory,
	settlementloaders.NewPayAdvanceByIDLoaderFactory,
	settlementloaders.NewEscrowAccountByIDLoaderFactory,
	settlementloaders.NewSettlementDisputesBySettlementIDLoaderFactory,
	settlementloaders.NewCarrierInvoiceMatchesByMoveIDLoaderFactory,
	billingloaders.NewFactoringCompanyByIDLoaderFactory,
	billingloaders.NewFactoringStateByInvoiceIDLoaderFactory,
	billingloaders.NewFactoringAgreementsByCompanyIDLoaderFactory,
	billingloaders.NewFactoringSubmissionItemsLoaderFactory,
	billingloaders.NewFactoringLiveItemByInvoiceIDLoaderFactory,
	billingloaders.NewFactoringSubmissionByIDLoaderFactory,
	billingloaders.NewLateChargeAssessmentsByInvoiceIDLoaderFactory,
	billingloaders.NewInvoiceEDISendPlanByInvoiceIDLoaderFactory,
	partiesloaders.NewCarrierIntelSnapshotByCarrierIDLoaderFactory,
	partiesloaders.NewCarrierIntelSnapshotByCustomerIDLoaderFactory,
	partiesloaders.NewCarrierIntelOpenEventCountLoaderFactory,
	documentloaders.NewInboundAttachmentCountLoaderFactory,
	documentloaders.NewInboundThreadCountLoaderFactory,
	documentloaders.NewInboundActivitiesByMessageIDLoaderFactory,
	documentloaders.NewDocumentTypeNameByIDLoaderFactory,
	documentloaders.NewTriageViewWaitingCountLoaderFactory,
	documentloaders.NewTriageRuleMonthMatchesLoaderFactory,
	documentloaders.NewTriageSampleByMessageIDLoaderFactory,
	documentloaders.NewInboundMailboxByIDLoaderFactory,
	documentloaders.NewCaptureRecordLabelLoaderFactory,
	documentloaders.NewShipmentSummaryByIDLoaderFactory,
	documentloaders.NewInboundAttachmentsByMessageIDLoaderFactory,
	documentloaders.NewInboundAttachmentByIDLoaderFactory,
	documentloaders.NewInboundReplyLoaderFactory,
	documentloaders.NewInboundDocumentReadingLoaderFactory,
	partiesloaders.NewCarrierMonitoringEnrollmentByCarrierIDLoaderFactory,
	agentloaders.NewAuditEntriesByAIAuditEventIDLoaderFactory,
))

func loaderSources(constructors ...any) fx.Option {
	annotated := make([]any, 0, len(constructors))
	for _, constructor := range constructors {
		annotated = append(annotated, fx.Annotate(
			constructor,
			fx.As(new(loadercore.Source)),
			fx.ResultTags(`group:"`+loadercore.SourceGroup+`"`),
		))
	}

	return fx.Provide(annotated...)
}

var graphQLResolverModule = fx.Module("api-graphql-resolvers", fx.Provide(
	resolver.New,
))

var graphQLExtensionModule = fx.Module("api-graphql-extensions", fx.Provide(
	gqlext.NewObservabilityExtension,
	gqlext.NewCostBudgetExtension,
	gqlext.NewReadOnlyExtension,
	gqlext.NewFieldGuardExtension,
	gqlext.NewQueryBudgetExtension,
))

var graphQLServerModule = fx.Module("api-graphql-server", fx.Provide(
	graphqlapi.NewPersistedOperationManifest,
	graphqlapi.NewServer,
))

// agentReadModule runs agent tools' reads through their persisted
// operations. Its executor holds only the shards and resolvers those
// operations reach (agentread/executor_gen.go), so the worker can take it
// without the full resolver root.
var agentReadModule = fx.Module("api-graphql-agent-read", fx.Provide(
	agentread.NewSchema,
	fx.Annotate(agentread.New, fx.As(new(graphqlreadports.GraphQLReader))),
))

var graphQLHandlerModule = fx.Module("api-graphql-handler", fx.Provide(
	graphqlapi.New,
))

var GraphQLModule = fx.Module(
	"api-graphql",
	graphQLLoaderModule,
	graphQLExtensionModule,
	graphQLResolverModule,
	graphQLServerModule,
	graphQLHandlerModule,
	agentReadModule,
)

// AgentReadWorkerModule is what the worker needs to run agent tools' reads:
// the loaders, the extensions every executor runs and the read executor,
// and not the API's resolver root, server or handler.
var AgentReadWorkerModule = fx.Module(
	"worker-graphql-agent-read",
	graphQLLoaderModule,
	graphQLExtensionModule,
	agentReadModule,
)
