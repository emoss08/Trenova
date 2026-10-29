package driversettlementtransferservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.Transactor
	Settlements       *driversettlementservice.Service
	SettlementRepo    driversettlementports.DriverSettlementRepository
	PayEventRepo      driversettlementports.PayEventRepository
	SettlementControl tenantports.SettlementControlRepository
	DriverNotify      *drivernotificationservice.Service
}

type Service struct {
	l                 *zap.Logger
	db                kernelports.Transactor
	settlements       *driversettlementservice.Service
	settlementRepo    driversettlementports.DriverSettlementRepository
	payEventRepo      driversettlementports.PayEventRepository
	settlementControl tenantports.SettlementControlRepository
	driverNotify      *drivernotificationservice.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.driver-settlement"),
		db:                p.DB,
		settlements:       p.Settlements,
		settlementRepo:    p.SettlementRepo,
		payEventRepo:      p.PayEventRepo,
		settlementControl: p.SettlementControl,
		driverNotify:      p.DriverNotify,
	}
}
