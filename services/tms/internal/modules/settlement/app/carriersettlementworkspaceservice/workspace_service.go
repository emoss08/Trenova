package carriersettlementworkspaceservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementrealtime"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	SettlementRepo    carriersettlementports.CarrierSettlementRepository
	BatchRepo         carriersettlementports.CarrierSettlementBatchRepository
	CostEventRepo     carriersettlementports.CarrierCostEventRepository
	SettlementControl tenantports.CarrierSettlementControlRepository
	AuditService      auditports.AuditService
	Realtime          realtimeports.RealtimeService
}

type Service struct {
	l                 *zap.Logger
	publisher         carriersettlementrealtime.Publisher
	settlementRepo    carriersettlementports.CarrierSettlementRepository
	batchRepo         carriersettlementports.CarrierSettlementBatchRepository
	costEventRepo     carriersettlementports.CarrierCostEventRepository
	settlementControl tenantports.CarrierSettlementControlRepository
	auditService      auditports.AuditService
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.carrier-settlement")
	return &Service{
		l:                 l,
		publisher:         carriersettlementrealtime.NewPublisher(l, p.Realtime),
		settlementRepo:    p.SettlementRepo,
		batchRepo:         p.BatchRepo,
		costEventRepo:     p.CostEventRepo,
		settlementControl: p.SettlementControl,
		auditService:      p.AuditService,
	}
}
