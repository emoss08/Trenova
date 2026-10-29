package carriersettlementcostservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementrealtime"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	CostEventRepo     carriersettlementports.CarrierCostEventRepository
	InvoiceMatchRepo  carriersettlementports.CarrierInvoiceMatchRepository
	SettlementControl tenantports.CarrierSettlementControlRepository
	CarrierRepo       carrierports.CarrierRepository
	AssignmentRepo    carrierassignmentports.CarrierAssignmentRepository
	ShipmentRepo      shipmentports.ShipmentRepository
	EDIInvoiceRepo    ediports.EDICarrierInvoiceRepository
	AuditService      auditports.AuditService
	Realtime          realtimeports.RealtimeService
}

type Service struct {
	l                 *zap.Logger
	publisher         carriersettlementrealtime.Publisher
	costEventRepo     carriersettlementports.CarrierCostEventRepository
	invoiceMatchRepo  carriersettlementports.CarrierInvoiceMatchRepository
	settlementControl tenantports.CarrierSettlementControlRepository
	carrierRepo       carrierports.CarrierRepository
	assignmentRepo    carrierassignmentports.CarrierAssignmentRepository
	shipmentRepo      shipmentports.ShipmentRepository
	ediInvoiceRepo    ediports.EDICarrierInvoiceRepository
	auditService      auditports.AuditService
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.carrier-settlement")
	return &Service{
		l:                 l,
		publisher:         carriersettlementrealtime.NewPublisher(l, p.Realtime),
		costEventRepo:     p.CostEventRepo,
		invoiceMatchRepo:  p.InvoiceMatchRepo,
		settlementControl: p.SettlementControl,
		carrierRepo:       p.CarrierRepo,
		assignmentRepo:    p.AssignmentRepo,
		shipmentRepo:      p.ShipmentRepo,
		ediInvoiceRepo:    p.EDIInvoiceRepo,
		auditService:      p.AuditService,
	}
}
