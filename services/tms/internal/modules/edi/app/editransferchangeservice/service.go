package editransferchangeservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledilifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AuditService       auditports.AuditService
	Coordinator        *shipmentstate.Coordinator
	DB                 kernelports.Transactor
	Logger             *zap.Logger
	OrderDerivation    orderports.OrderDerivationService
	ShipmentEventRepo  shipmenteventports.ShipmentEventRepository
	ShipmentLinkRepo   editenderports.EDIShipmentLinkRepository
	ShipmentRepo       shipmentports.ShipmentRepository
	Shipments          *editendershipment.Service
	TransferChangeRepo editenderports.EDITransferChangeRepository
}

type Service struct {
	auditService       auditports.AuditService
	coordinator        *shipmentstate.Coordinator
	db                 kernelports.Transactor
	l                  *zap.Logger
	lifecycleApplier   *internaledilifecycle.Applier
	orderDerivation    orderports.OrderDerivationService
	shipmentEventRepo  shipmenteventports.ShipmentEventRepository
	shipmentLinkRepo   editenderports.EDIShipmentLinkRepository
	shipmentRepo       shipmentports.ShipmentRepository
	transferChangeRepo editenderports.EDITransferChangeRepository
	shipments          *editendershipment.Service
}

func New(p Params) *Service {
	return &Service{
		auditService: p.AuditService,
		coordinator:  p.Coordinator,
		db:           p.DB,
		l:            p.Logger.Named("service.edi"),
		lifecycleApplier: internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: p.ShipmentRepo,
			Coordinator:  p.Coordinator,
		}),
		orderDerivation:    p.OrderDerivation,
		shipmentEventRepo:  p.ShipmentEventRepo,
		shipmentLinkRepo:   p.ShipmentLinkRepo,
		shipmentRepo:       p.ShipmentRepo,
		transferChangeRepo: p.TransferChangeRepo,
		shipments:          p.Shipments,
	}
}

func (s *Service) logAction(
	entity edilading.Audited,
	actor *permissionports.RequestActor,
	operation permission.Operation,
	previous any,
	current any,
	comment string,
) {
	edilading.LogAction(s.auditService, s.l, &edilading.AuditEntry{
		Entity:    entity,
		Actor:     actor,
		Operation: operation,
		Previous:  previous,
		Current:   current,
		Comment:   comment,
	})
}
