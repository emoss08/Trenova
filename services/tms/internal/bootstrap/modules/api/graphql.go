//revive:disable-next-line:var-naming
package api

import (
	graphqlapi "github.com/emoss08/trenova/internal/api/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver"
	"go.uber.org/fx"
)

var graphQLLoaderModule = fx.Module("api-graphql-loaders", fx.Provide(
	loaders.NewFactory,
	loaders.NewOrganizationByIDLoaderFactory,
	loaders.NewLocationByIDLoaderFactory,
	loaders.NewOrderByIDLoaderFactory,
	loaders.NewShipmentProfitabilityLoaderFactory,
	loaders.NewEDIPartnerByCustomerIDLoaderFactory,
	loaders.NewFormulaTemplateStatsLoaderFactory,
	loaders.NewPayProfileActiveAssignmentCountLoaderFactory,
	loaders.NewPTOPolicyOpenAssignmentCountLoaderFactory,
	loaders.NewShiftTemplateActiveAssignmentCountLoaderFactory,
	loaders.NewWorkerChecklistTemplateOpenChecklistCountLoaderFactory,
	loaders.NewWorkerCredentialTypeActiveCredentialCountLoaderFactory,
	loaders.NewTrainingCourseOpenRecordCountLoaderFactory,
	loaders.NewPerformanceReviewTemplateOpenReviewCountLoaderFactory,
	loaders.NewCustomerByIDLoaderFactory,
	loaders.NewInvoiceByIDLoaderFactory,
	loaders.NewGLAccountByIDLoaderFactory,
	loaders.NewRoutingGuideWithEntriesByIDLoaderFactory,
	loaders.NewWorkerDQFVerificationsByWorkerIDLoaderFactory,
	loaders.NewWorkerLeaveEntriesByCaseIDLoaderFactory,
	loaders.NewFiscalPeriodsByFiscalYearIDLoaderFactory,
	loaders.NewDocumentTemplateKindByTemplateIDLoaderFactory,
))

var graphQLResolverModule = fx.Module("api-graphql-resolvers", fx.Provide(
	resolver.New,
))

var graphQLServerModule = fx.Module("api-graphql-server", fx.Provide(
	graphqlapi.NewPersistedOperationManifest,
	graphqlapi.NewObservabilityExtension,
	graphqlapi.NewCostBudgetExtension,
	graphqlapi.NewServer,
))

var graphQLHandlerModule = fx.Module("api-graphql-handler", fx.Provide(
	graphqlapi.New,
))

var GraphQLModule = fx.Module(
	"api-graphql",
	graphQLLoaderModule,
	graphQLResolverModule,
	graphQLServerModule,
	graphQLHandlerModule,
)
