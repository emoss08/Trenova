package modules

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountsreceivablerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/agentmemorysubjectrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/agentsubjectrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/assistantcaserepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/carrierinvoicematchrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/casechecklistrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/costingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dispatchconsolerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/organizationrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/orgstructurerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/roleagentgrantrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/schedulingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/tenantbootstraprepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/userrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tableinsight"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
)

var PostgresRepositoryModule = fx.Module("postgres-repositories", fx.Provide(
	m2msync.NewSyncer,
	organizationrepository.New,
	tenantbootstraprepository.New,
	userrepository.New,
	assistantcaserepository.New,
	casechecklistrepository.New,
	tableinsight.NewCustomerSource,
	tableinsight.NewInvoiceSource,
	tableinsight.NewLocationSource,
	tableinsight.NewCarrierSource,
	tableinsight.NewOrderSource,
	tableinsight.NewServiceFailureSource,
	orgstructurerepository.New,
	schedulingrepository.New,
	dispatchconsolerepository.New,
	roleagentgrantrepository.New,
	agentmemorysubjectrepository.New,
	agentsubjectrepository.New,
	seqgen.NewSequenceStore,
	seqgen.NewFormatProvider,
	seqgen.NewGenerator,
	carrierinvoicematchrepository.New,
	accountsreceivablerepository.New,
	accountsreceivablerepository.NewAnalytics,
	costingrepository.New,
	costingrepository.NewActuals,
))
