package agentquerytoolservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/accountingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/billingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ediquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/equipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/factoringquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/formulaquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fuelquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/inboxquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/ratequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/reportquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/settlementquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workforcequerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// graphQLListSpecs are every list tool, module by module, that reads
// through GraphQL by a spec alone.
func graphQLListSpecs() []readspec.List {
	return slices.Concat(
		masterdataquerytools.ReferenceReadSpecs(),
		partyquerytools.ReadSpecs(),
		equipmentquerytools.ReadSpecs(),
		workerquerytools.ReadSpecs(),
		workforcequerytools.TrainingReadSpecs(),
		workforcequerytools.TimeOffReadSpecs(),
		workerquerytools.RecordReadSpecs(),
		operationsquerytools.ReadSpecs(),
		accountingquerytools.ReadSpecs(),
		ediquerytools.ReadSpecs(),
		accountingquerytools.LedgerReadSpecs(),
		ratequerytools.ReadSpecs(),
		workforcequerytools.ProgrammeReadSpecs(),
		formulaquerytools.ReadSpecs(),
		fuelquerytools.ReadSpecs(),
		settlementquerytools.ReadSpecs(),
		factoringquerytools.ReadSpecs(),
		billingquerytools.ReadSpecs(),
		partyquerytools.CarrierReadSpecs(),
		reportquerytools.ReadSpecs(),
		inboxquerytools.ReadSpecs(),
	)
}

// graphQLListProviders registers each spec as its own list tool.
func graphQLListProviders() []any {
	specs := graphQLListSpecs()
	providers := make([]any, 0, len(specs))
	for idx := range specs {
		spec := &specs[idx]
		providers = append(providers, func(
			reader graphqlreadports.GraphQLReader,
			permissions permissionports.PermissionEngine,
		) (agenttoolports.AgentQueryTool, error) {
			return graphqllisttool.NewRead(spec, reader, permissions)
		})
	}

	return providers
}
