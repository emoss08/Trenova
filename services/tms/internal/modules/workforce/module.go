package workforce

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/orgholidayrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/performancereviewrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/permitrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptoledgerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/ptopolicyrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/schedulingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/selfservicerepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/telematicsrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/timesheetrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/weekhoursrepository"
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
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalinvitationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalrecordsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/homevisitservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/orgholidayservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/permitservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/samsarasyncservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
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
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/samsarajobs/samsaracontract"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workertrainingports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		boardReaders,
		fx.Provide(
			schedulingrepository.New,
			drivernotificationservice.New,
			driverportalservice.New,
			driverportalinvitationservice.New,
			driverportalrecordsservice.New,
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
			func(s *workertrainingservice.Service) workertrainingports.TrainingAssigner { return s },
			workerchecklistservice.New,
			func(s *workercredentialservice.Service) workeroverviewservice.CredentialSummarizer { return s },
			func(s *workertrainingservice.Service) workeroverviewservice.TrainingSummarizer { return s },
			func(s *workersafetyservice.Service) workeroverviewservice.SafetyReader { return s },
			func(s *workercredentialservice.Service) workerdqfservice.CredentialSummarizer { return s },
			func(s *workerdrugalcoholservice.Service) workerdqfservice.DrugAlcoholStander { return s },
			func(s *workerchecklistservice.Service) workeroverviewservice.ChecklistLister { return s },
			func(s *performancereviewservice.Service) workeroverviewservice.ReviewLister { return s },
			func(s *workerleaveservice.Service) workeroverviewservice.LeaveCounter { return s },
			func(s *workerleaveservice.Service) workeroverviewservice.LeaveEntitler { return s },
			func(s *workerdrugalcoholservice.Service) workeroverviewservice.DrugAlcoholStander {
				return s
			},
			func(s *workerdqfservice.Service) workeroverviewservice.QualificationFiler { return s },
			func(
				r workeremploymentports.WorkerEmploymentEventRepository,
			) workeroverviewservice.EmploymentSummarizer {
				return r
			},
			func(s *schedulingservice.Service) workeroverviewservice.ShiftLister { return s },
			func(s *telematicsservice.Service) workeroverviewservice.HOSLinker { return s },
			func(r tractorports.TractorRepository) workeroverviewservice.TractorReader { return r },
			func(s workerptoports.WorkerPTOService) workeroverviewservice.TimeOffLister { return s },
			func(s *benefitsservice.Service) workeroverviewservice.BenefitReader { return s },
			func(
				r portalaccessports.PortalAccessRepository,
			) workeroverviewservice.PortalInvitationReader {
				return r
			},
			func(
				r selfserviceports.SelfServiceRepository,
			) workeroverviewservice.ProfileChangeLister {
				return r
			},
			func(s *workerleaveservice.Service) driverportalrecordsservice.LeaveReader { return s },
			orgholidayservice.New,
			func(s *workerchecklistservice.Service) workerchecklistports.ChecklistSpawner { return s },
			func(s *workerchecklistservice.Service) workerchecklistports.EvidenceRefresher { return s },
			fx.Annotate(
				func(s *workerchecklistservice.Service) documentports.ResourceDocumentsWatcher {
					return s
				},
				fx.ResultTags(`group:"resource_document_watchers"`),
			),
			workeremploymentservice.New,
			func(s *workeremploymentservice.Service) workeremploymentports.EmploymentEventRecorder { return s },
			samsaracontract.NewManager,
			telematicsservice.New,
			homevisitservice.New,
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
			weekhoursrepository.New,
			teamscope.NewScheduleNarrower,
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
			workerphotoservice.New,
			func(s *workerphotoservice.Service) selfserviceservice.WorkerPhotos { return s },
			workeroverviewservice.New,
			func(r workerreviewports.PerformanceReviewRepository) workeroverviewservice.ReviewCounter {
				return r
			},
			samsarasyncservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
