package modules

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tableinsight"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
)

var PostgresRepositoryModule = fx.Module("postgres-repositories", fx.Provide(
	tableinsight.NewCustomerSource,
	tableinsight.NewInvoiceSource,
	tableinsight.NewLocationSource,
	tableinsight.NewCarrierSource,
	tableinsight.NewOrderSource,
	tableinsight.NewServiceFailureSource,
	seqgen.NewSequenceStore,
	seqgen.NewFormatProvider,
	seqgen.NewGenerator,
))
