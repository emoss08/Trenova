package workforce

import (
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/orgholidayrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/performancereviewrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/permitrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptoledgerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptopolicyrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/schedulingrepository"
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
	"github.com/emoss08/trenova/internal/modules/workforce/app/benefitsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/orgholidayservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/permitservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/samsarasyncservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/timesheetservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/samsarajobs/samsaracontract"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			schedulingrepository.New,
			drivernotificationservice.New,
			driverportalservice.New,
			ptopolicyservice.New,
			workercredentialservice.New,
			workertrainingservice.New,
			workerdqfservice.New,
			workerdrugalcoholservice.New,
			workerinjuryservice.New,
			workerleaveservice.New,
			benefitsservice.New,
			schedulingservice.New,
			selfserviceservice.New,
			selfserviceservice.NewDocumentChecksums,
			timesheetservice.New,
			workersafetyservice.New,
			performancereviewservice.New,
			func(s *workertrainingservice.Service) workerports.TrainingAssigner { return s },
			workerchecklistservice.New,
			func(s *workercredentialservice.Service) workeroverviewservice.CredentialSummarizer { return s },
			func(s *workertrainingservice.Service) workeroverviewservice.TrainingSummarizer { return s },
			func(s *workersafetyservice.Service) workeroverviewservice.SafetyScorer { return s },
			func(s *workercredentialservice.Service) workerdqfservice.CredentialSummarizer { return s },
			func(s *workerdrugalcoholservice.Service) workerdqfservice.DrugAlcoholStander { return s },
			func(s *workerchecklistservice.Service) workeroverviewservice.ChecklistLister { return s },
			func(s *performancereviewservice.Service) workeroverviewservice.ReviewLister { return s },
			func(s *workerleaveservice.Service) workeroverviewservice.LeaveCounter { return s },
			func(s *workerleaveservice.Service) driverportalservice.LeaveReader { return s },
			orgholidayservice.New,
			func(s *workerchecklistservice.Service) workerports.ChecklistSpawner { return s },
			workeremploymentservice.New,
			func(s *workeremploymentservice.Service) workerports.EmploymentEventRecorder { return s },
			samsaracontract.NewManager,
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
