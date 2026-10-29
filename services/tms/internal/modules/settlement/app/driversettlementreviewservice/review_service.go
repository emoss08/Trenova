package driversettlementreviewservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Settlements         *driversettlementservice.Service
	SettlementRepo      driversettlementports.DriverSettlementRepository
	DisputeRepo         driversettlementports.SettlementDisputeRepository
	ExpenseRepo         driverpayports.DriverExpenseRepository
	PayCodeRepo         driverpayports.PayCodeRepository
	SettlementControl   tenantports.SettlementControlRepository
	DashControlRepo     tenantports.DashControlRepository
	AuditService        auditports.AuditService
	DriverNotify        *drivernotificationservice.Service
	NotificationService *notificationservice.Service
}

type Service struct {
	l                   *zap.Logger
	settlements         *driversettlementservice.Service
	settlementRepo      driversettlementports.DriverSettlementRepository
	disputeRepo         driversettlementports.SettlementDisputeRepository
	expenseRepo         driverpayports.DriverExpenseRepository
	payCodeRepo         driverpayports.PayCodeRepository
	settlementControl   tenantports.SettlementControlRepository
	dashControlRepo     tenantports.DashControlRepository
	auditService        auditports.AuditService
	driverNotify        *drivernotificationservice.Service
	notificationService *notificationservice.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.driver-settlement"),
		settlements:         p.Settlements,
		settlementRepo:      p.SettlementRepo,
		disputeRepo:         p.DisputeRepo,
		expenseRepo:         p.ExpenseRepo,
		payCodeRepo:         p.PayCodeRepo,
		settlementControl:   p.SettlementControl,
		dashControlRepo:     p.DashControlRepo,
		auditService:        p.AuditService,
		driverNotify:        p.DriverNotify,
		notificationService: p.NotificationService,
	}
}
