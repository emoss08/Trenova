package driverportalservice

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementaccrualservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementreviewservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementworkspaceservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/driverportalports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	PortalRepo          portalaccessports.PortalAccessRepository
	SettlementRepo      driversettlementports.DriverSettlementRepository
	PayEventRepo        driversettlementports.PayEventRepository
	AdvanceRepo         driverpayports.PayAdvanceRepository
	EscrowRepo          driverpayports.EscrowAccountRepository
	SettlementControl   tenantports.SettlementControlRepository
	DashControlRepo     tenantports.DashControlRepository
	DocumentTypeRepo    documenttypeports.DocumentTypeRepository
	PTORepo             workerptoports.WorkerPTORepository
	CommentService      shipmentcommentports.ShipmentCommentService
	StopReporter        driverportalports.StopReporter
	WorkerService       *workerservice.Service
	DocumentService     *documentservice.Service
	PTOService          workerptoports.WorkerPTOService
	PTOLedger           *ptoledgerservice.Service
	SelfService         *selfserviceservice.Service
	ExpenseService      *driversettlementreviewservice.Service
	PayEstimator        *driversettlementaccrualservice.Service
	PayHistory          *driversettlementworkspaceservice.Service
	NotificationService *notificationservice.Service
}

type Service struct {
	l                   *zap.Logger
	portalRepo          portalaccessports.PortalAccessRepository
	settlementRepo      driversettlementports.DriverSettlementRepository
	payEventRepo        driversettlementports.PayEventRepository
	advanceRepo         driverpayports.PayAdvanceRepository
	escrowRepo          driverpayports.EscrowAccountRepository
	settlementControl   tenantports.SettlementControlRepository
	dashControlRepo     tenantports.DashControlRepository
	documentTypeRepo    documenttypeports.DocumentTypeRepository
	ptoRepo             workerptoports.WorkerPTORepository
	commentService      shipmentcommentports.ShipmentCommentService
	stopReporter        driverportalports.StopReporter
	workerService       *workerservice.Service
	documentService     *documentservice.Service
	ptoService          workerptoports.WorkerPTOService
	ptoLedger           *ptoledgerservice.Service
	selfService         *selfserviceservice.Service
	expenseService      *driversettlementreviewservice.Service
	payEstimator        *driversettlementaccrualservice.Service
	payHistory          *driversettlementworkspaceservice.Service
	notificationService *notificationservice.Service
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                   p.Logger.Named("service.driver-portal"),
		portalRepo:          p.PortalRepo,
		settlementRepo:      p.SettlementRepo,
		payEventRepo:        p.PayEventRepo,
		advanceRepo:         p.AdvanceRepo,
		escrowRepo:          p.EscrowRepo,
		settlementControl:   p.SettlementControl,
		dashControlRepo:     p.DashControlRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		ptoRepo:             p.PTORepo,
		commentService:      p.CommentService,
		stopReporter:        p.StopReporter,
		workerService:       p.WorkerService,
		documentService:     p.DocumentService,
		ptoService:          p.PTOService,
		ptoLedger:           p.PTOLedger,
		selfService:         p.SelfService,
		expenseService:      p.ExpenseService,
		payEstimator:        p.PayEstimator,
		payHistory:          p.PayHistory,
		notificationService: p.NotificationService,
	}
}
