package driversettlementworkspaceservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Settlements       *driversettlementservice.Service
	SettlementRepo    driversettlementports.DriverSettlementRepository
	BatchRepo         driversettlementports.SettlementBatchRepository
	PayEventRepo      driversettlementports.PayEventRepository
	SettlementControl tenantports.SettlementControlRepository
}

type Service struct {
	settlements       *driversettlementservice.Service
	settlementRepo    driversettlementports.DriverSettlementRepository
	batchRepo         driversettlementports.SettlementBatchRepository
	payEventRepo      driversettlementports.PayEventRepository
	settlementControl tenantports.SettlementControlRepository
}

func New(p Params) *Service {
	return &Service{
		settlements:       p.Settlements,
		settlementRepo:    p.SettlementRepo,
		batchRepo:         p.BatchRepo,
		payEventRepo:      p.PayEventRepo,
		settlementControl: p.SettlementControl,
	}
}
