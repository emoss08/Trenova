package aicli

import (
	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	modulesinfra "github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingjobs"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingrepository"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingservice"
	"github.com/emoss08/trenova/internal/cloud/aitraining/retrainingalert"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/core/services/aidocumentservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionrouter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/aicorrectionrepository"
	"go.uber.org/fx"
)

func CommandOptions() fx.Option {
	return fx.Options(
		fx.NopLogger,
		config.SectionsOption(cloudconfig.Section()),
		config.Module,
		infrastructure.ObservabilityModule,
		infrastructure.DatabaseModule,
		modulesinfra.StorageModule,
		fx.Provide(
			temporaljobs.NewTemporalClient,
			aitrainingrepository.NewExports,
			aitrainingrepository.NewRecords,
			aitrainingrepository.NewCycles,
			aicorrectionrepository.New,
			aitrainingjobs.NewExportStarter,
			aitrainingjobs.AsExportStarter,
			aitrainingservice.NewOperator,
			aitrainingservice.AsOperator,
			aitrainingservice.NewRenderer,
			aitrainingservice.AsRenderer,
			aitrainingservice.NewRetrainer,
			aitrainingservice.AsRetrainer,
			retrainingalert.New,
			retrainingalert.AsAlerter,
			aidocumentservice.NewContract,
			completionrouter.NewPromptRenderer,
		),
	)
}
