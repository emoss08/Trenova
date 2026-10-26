package workspace

import (
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditdlqrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/bulkeditrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldvaluerepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/dataentrycontrolrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/emailrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/homelayoutrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/integrationrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/notificationpreferencerepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/notificationrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/pagefavoriterepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/pushsubscriptionrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/reportrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/sidebarpreferencerepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/tableconfigurationrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/tablelayoutrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/tcaallowlistrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/tcasubscriptionrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/analyticsservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/bulkeditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/dataentrycontrolservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/googlemapsservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationpreferenceservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/pagefavoriteservice"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/sidebarpreferenceservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableexportservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/webpushservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		analyticsservice.Module,
		fx.Provide(
			tableexportservice.New,
			tableconfigurationrepository.New,
			pagefavoriterepository.New,
			reportrepository.NewDefinitionRepository,
			reportrepository.NewCatalogVersionRepository,
			reportrepository.NewDashboardRepository,
			reportrepository.NewRunRepository,
			reportrepository.NewScheduleRepository,
			reportrepository.NewViewRepository,
			sidebarpreferencerepository.New,
			tablelayoutrepository.New,
			bulkeditrepository.New,
			homelayoutrepository.New,
			auditdlqrepository.New,
			auditrepository.New,
			dataentrycontrolrepository.New,
			customfieldrepository.New,
			customfieldvaluerepository.New,
			pushsubscriptionrepository.New,
			integrationrepository.New,
			emailrepository.New,
			tcasubscriptionrepository.New,
			tcaallowlistrepository.New,
			notificationrepository.New,
			notificationpreferencerepository.New,
			customfieldservice.NewValidator,
			customfieldservice.NewValuesValidator,
			emailservice.NewValidator,
			dataentrycontrolservice.NewValidator,
			tableconfigurationservice.New,
			pagefavoriteservice.New,
			sidebarpreferenceservice.New,
			tablelayoutservice.New,
			bulkeditservice.New,
			bulkeditservice.NewCustomerEditor,
			bulkeditservice.NewLocationEditor,
			bulkeditservice.NewCarrierEditor,
			bulkeditservice.NewShipmentEditor,
			homelayoutservice.New,
			reportingservice.New,
			auditservice.New,
			auditservice.NewSecurityAuditor,
			customfieldservice.New,
			customfieldservice.NewValuesService,
			emailservice.New,
			func(s *emailservice.Service) emailports.EmailService { return s },
			func(s *emailservice.Service) emailports.EmailSenderResolver { return s },
			webpushservice.New,
			googlemapsservice.NewAutoCompleteService,
			dataentrycontrolservice.New,
			notificationservice.New,
			notificationpreferenceservice.New,
			tablechangealertservice.New,
			tablechangealertservice.NewConsumer,
			integrationservice.New,
			fx.Annotate(
				func(svc *integrationservice.Service) fuelpurchaseports.FuelCardFeedResolver { return svc },
			),
		),
		fx.Invoke(emailservice.RegisterObservers),
		formulaassistantservice.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
