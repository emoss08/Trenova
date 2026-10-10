package workforce

import (
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/orgholidayrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/performancereviewrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/permitrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptoledgerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptopolicyrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/selfservicerepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/telematicsrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/timesheetrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerchecklistrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workercredentialrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdqfrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdrugalcoholrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workeremploymenteventrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerinjuryrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerleaverepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerptorepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workersafetyrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workertrainingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/permitservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/samsarasyncservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			telematicsservice.New,
			workerrepository.New,
			workerptorepository.New,
			workercredentialrepository.New,
			workertrainingrepository.New,
			workerdqfrepository.New,
			workerdrugalcoholrepository.New,
			workerinjuryrepository.New,
			workerleaverepository.New,
			selfservicerepository.New,
			timesheetrepository.New,
			workersafetyrepository.New,
			performancereviewrepository.New,
			workerchecklistrepository.New,
			orgholidayrepository.New,
			workeremploymenteventrepository.New,
			ptopolicyrepository.New,
			ptoledgerrepository.New,
			telematicsrepository.New,
			permitrepository.New,
			permitrepository.NewJurisdictionRuleRepository,
			workerservice.NewValidator,
			workerservice.New,
			permitservice.NewService,
			func(ws *workerservice.Service) selfserviceservice.WorkerUpdater { return ws },
			workeroverviewservice.New,
			func(r workerports.PerformanceReviewRepository) workeroverviewservice.ReviewCounter {
				return r
			},
			samsarasyncservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
