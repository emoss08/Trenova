package driversettlementaccrualservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Settlements       *driversettlementservice.Service
	PayEventRepo      driversettlementports.PayEventRepository
	AssignmentRepo    driverpayports.WorkerPayAssignmentRepository
	SettlementControl tenantports.SettlementControlRepository
	ShipmentRepo      shipmentports.ShipmentRepository
}

type Service struct {
	l                 *zap.Logger
	settlements       *driversettlementservice.Service
	payEventRepo      driversettlementports.PayEventRepository
	assignmentRepo    driverpayports.WorkerPayAssignmentRepository
	settlementControl tenantports.SettlementControlRepository
	shipmentRepo      shipmentports.ShipmentRepository
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.driver-settlement"),
		settlements:       p.Settlements,
		payEventRepo:      p.PayEventRepo,
		assignmentRepo:    p.AssignmentRepo,
		settlementControl: p.SettlementControl,
		shipmentRepo:      p.ShipmentRepo,
	}
}
