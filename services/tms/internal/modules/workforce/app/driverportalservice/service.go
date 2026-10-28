package driverportalservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/driverportalports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	DriverRoleName        = "Driver"
	driverRoleDescription = "Portal access for drivers; grants no back-office permissions"
	invitationTTLSeconds  = int64(7 * 24 * 60 * 60)
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Config              *config.Config
	PortalRepo          portalaccessports.PortalAccessRepository
	OrgRepo             tenantports.OrganizationRepository
	SettlementRepo      driversettlementports.DriverSettlementRepository
	PayEventRepo        driversettlementports.PayEventRepository
	AdvanceRepo         driverpayports.PayAdvanceRepository
	EscrowRepo          driverpayports.EscrowAccountRepository
	SettlementControl   tenantports.SettlementControlRepository
	DashControlRepo     tenantports.DashControlRepository
	DocumentTypeRepo    documenttypeports.DocumentTypeRepository
	PTORepo             workerptoports.WorkerPTORepository
	EmailService        emailports.EmailService
	Templates           documenttemplateports.DocumentTemplateResolver
	CommentService      shipmentcommentports.ShipmentCommentService
	StopReporter        driverportalports.StopReporter
	WorkerService       *workerservice.Service
	DocumentService     *documentservice.Service
	PTOService          workerptoports.WorkerPTOService
	PTOLedger           *ptoledgerservice.Service
	Credentials         *workercredentialservice.Service
	Training            *workertrainingservice.Service
	Safety              *workersafetyservice.Service
	Reviews             *performancereviewservice.Service
	Leave               LeaveReader
	Scheduling          *schedulingservice.Service
	SelfService         *selfserviceservice.Service
	SettlementService   *driversettlementservice.Service
	NotificationService *notificationservice.Service
	AuditService        auditports.AuditService
	Sessions            sessionports.SessionRepository
}

type Service struct {
	l                   *zap.Logger
	cfg                 *config.Config
	portalRepo          portalaccessports.PortalAccessRepository
	orgRepo             tenantports.OrganizationRepository
	settlementRepo      driversettlementports.DriverSettlementRepository
	payEventRepo        driversettlementports.PayEventRepository
	advanceRepo         driverpayports.PayAdvanceRepository
	escrowRepo          driverpayports.EscrowAccountRepository
	settlementControl   tenantports.SettlementControlRepository
	dashControlRepo     tenantports.DashControlRepository
	documentTypeRepo    documenttypeports.DocumentTypeRepository
	ptoRepo             workerptoports.WorkerPTORepository
	emailService        emailports.EmailService
	templates           documenttemplateports.DocumentTemplateResolver
	commentService      shipmentcommentports.ShipmentCommentService
	stopReporter        driverportalports.StopReporter
	workerService       *workerservice.Service
	documentService     *documentservice.Service
	ptoService          workerptoports.WorkerPTOService
	ptoLedger           *ptoledgerservice.Service
	credentials         *workercredentialservice.Service
	training            *workertrainingservice.Service
	safety              *workersafetyservice.Service
	reviews             *performancereviewservice.Service
	leave               LeaveReader
	scheduling          *schedulingservice.Service
	selfService         *selfserviceservice.Service
	settlementService   *driversettlementservice.Service
	notificationService *notificationservice.Service
	auditService        auditports.AuditService
	sessions            sessionports.SessionRepository
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                   p.Logger.Named("service.driver-portal"),
		cfg:                 p.Config,
		portalRepo:          p.PortalRepo,
		orgRepo:             p.OrgRepo,
		settlementRepo:      p.SettlementRepo,
		payEventRepo:        p.PayEventRepo,
		advanceRepo:         p.AdvanceRepo,
		escrowRepo:          p.EscrowRepo,
		settlementControl:   p.SettlementControl,
		dashControlRepo:     p.DashControlRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		ptoRepo:             p.PTORepo,
		emailService:        p.EmailService,
		templates:           p.Templates,
		commentService:      p.CommentService,
		stopReporter:        p.StopReporter,
		workerService:       p.WorkerService,
		documentService:     p.DocumentService,
		ptoService:          p.PTOService,
		ptoLedger:           p.PTOLedger,
		credentials:         p.Credentials,
		training:            p.Training,
		safety:              p.Safety,
		reviews:             p.Reviews,
		leave:               p.Leave,
		scheduling:          p.Scheduling,
		selfService:         p.SelfService,
		settlementService:   p.SettlementService,
		notificationService: p.NotificationService,
		auditService:        p.AuditService,
		sessions:            p.Sessions,
	}
}
