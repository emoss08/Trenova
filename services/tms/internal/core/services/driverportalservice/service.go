package driverportalservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/workerptoservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	PortalRepo          repositories.PortalAccessRepository
	SettlementRepo      repositories.DriverSettlementRepository
	PayEventRepo        repositories.PayEventRepository
	AdvanceRepo         repositories.PayAdvanceRepository
	EscrowRepo          repositories.EscrowAccountRepository
	SettlementControl   repositories.SettlementControlRepository
	DashControlRepo     repositories.DashControlRepository
	DocumentTypeRepo    repositories.DocumentTypeRepository
	PTORepo             repositories.WorkerPTORepository
	EmailService        serviceports.EmailService
	CommentService      serviceports.ShipmentCommentService
	MoveService         serviceports.ShipmentMoveService
	WorkerService       *workerservice.Service
	DocumentService     *documentservice.Service
	PTOService          *workerptoservice.Service
	SettlementService   *driversettlementservice.Service
	NotificationService *notificationservice.Service
	AuditService        serviceports.AuditService
}

type Service struct {
	l                   *zap.Logger
	cfg                 *config.Config
	portalRepo          repositories.PortalAccessRepository
	settlementRepo      repositories.DriverSettlementRepository
	payEventRepo        repositories.PayEventRepository
	advanceRepo         repositories.PayAdvanceRepository
	escrowRepo          repositories.EscrowAccountRepository
	settlementControl   repositories.SettlementControlRepository
	dashControlRepo     repositories.DashControlRepository
	documentTypeRepo    repositories.DocumentTypeRepository
	ptoRepo             repositories.WorkerPTORepository
	emailService        serviceports.EmailService
	commentService      serviceports.ShipmentCommentService
	moveService         serviceports.ShipmentMoveService
	workerService       *workerservice.Service
	documentService     *documentservice.Service
	ptoService          *workerptoservice.Service
	settlementService   *driversettlementservice.Service
	notificationService *notificationservice.Service
	auditService        serviceports.AuditService
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                   p.Logger.Named("service.driver-portal"),
		cfg:                 p.Config,
		portalRepo:          p.PortalRepo,
		settlementRepo:      p.SettlementRepo,
		payEventRepo:        p.PayEventRepo,
		advanceRepo:         p.AdvanceRepo,
		escrowRepo:          p.EscrowRepo,
		settlementControl:   p.SettlementControl,
		dashControlRepo:     p.DashControlRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		ptoRepo:             p.PTORepo,
		emailService:        p.EmailService,
		commentService:      p.CommentService,
		moveService:         p.MoveService,
		workerService:       p.WorkerService,
		documentService:     p.DocumentService,
		ptoService:          p.PTOService,
		settlementService:   p.SettlementService,
		notificationService: p.NotificationService,
		auditService:        p.AuditService,
	}
}
