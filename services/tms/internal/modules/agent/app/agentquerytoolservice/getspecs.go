package agentquerytoolservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/accountingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/billingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ediquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/equipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/factoringquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/formulaquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/homequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/inboxquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/oversightquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ratequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/settlementquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// graphQLGetSpecs are every get tool that reads through GraphQL by a spec
// alone.
func graphQLGetSpecs() []readspec.Get {
	return append([]readspec.Get{
		partyquerytools.CustomerGetSpec(),
		homequerytools.CustomerUpdatePreferencesGetSpec(),
		partyquerytools.CarrierGetSpec(),
		partyquerytools.LocationGetSpec(),
		partyquerytools.CarrierIntelEventGetSpec(),
		equipmentquerytools.TractorGetSpec(),
		equipmentquerytools.TrailerGetSpec(),
		operationsquerytools.OrderGetSpec(),
		operationsquerytools.ServiceFailureGetSpec(),
		operationsquerytools.DetentionOccurrenceGetSpec(),
		accountingquerytools.JournalEntryGetSpec(),
		accountingquerytools.ManualJournalGetSpec(),
		ratequerytools.RateAgreementGetSpec(),
		workerquerytools.GetSpec(),
		workerquerytools.CredentialGetSpec(),
		workerquerytools.DOTRandomDrawGetSpec(),
		workerquerytools.HOSGetSpec(),
		formulaquerytools.FormulaTemplateGetSpec(),
		formulaquerytools.FormulaLibraryGetSpec(),
		billingquerytools.InvoiceGetSpec(),
		billingquerytools.InvoiceAdjustmentGetSpec(),
		homequerytools.HomeLayoutGetSpec(),
		ratequerytools.RateMatrixGetSpec(),
		oversightquerytools.DailyBriefingGetSpec(),
	}, slices.Concat(
		accountingquerytools.GetSpecs(),
		ediquerytools.GetSpecs(),
		settlementquerytools.GetSpecs(),
		factoringquerytools.GetSpecs(),
		inboxquerytools.GetSpecs(),
	)...)
}

// graphQLGetProviders registers each spec as its own get tool.
func graphQLGetProviders() []any {
	specs := graphQLGetSpecs()
	providers := make([]any, 0, len(specs))
	for idx := range specs {
		spec := &specs[idx]
		providers = append(providers, func(
			reader graphqlreadports.GraphQLReader,
			permissions permissionports.PermissionEngine,
		) (agenttoolports.AgentQueryTool, error) {
			return gettool.NewGraphQL(spec, reader, permissions)
		})
	}

	return providers
}

func getCatalogSpecs() []readspec.Get {
	return []readspec.Get{
		partyquerytools.CustomerGetSpec(),
		partyquerytools.CarrierGetSpec(),
		equipmentquerytools.TractorGetSpec(),
		equipmentquerytools.TrailerGetSpec(),
		billingquerytools.InvoiceGetSpec(),
		billingquerytools.InvoiceAdjustmentGetSpec(),
		partyquerytools.LocationGetSpec(),
		operationsquerytools.DetentionOccurrenceGetSpec(),
		operationsquerytools.ServiceFailureGetSpec(),
		formulaquerytools.FormulaTemplateGetSpec(),
	}
}
