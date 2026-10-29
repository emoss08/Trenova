package editransferservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AuditService     auditports.AuditService
	CustomFields     customfieldports.NamedValues `optional:"true"`
	DB               kernelports.Transactor
	Logger           *zap.Logger
	Outbound         *edioutbound.Service
	PartnerRepo      edipartnerports.EDIPartnerRepository
	Recipient        *editenderrecipient.Service
	Savepoints       kernelports.Savepointer
	Setup            *edisetup.Service
	ShipmentLinkRepo editenderports.EDIShipmentLinkRepository
	ShipmentSvc      shipmentserviceports.ShipmentWriter
	Shipments        *editendershipment.Service
	TenderMapping    *editendermapping.Service
	TransferRepo     editenderports.EDILoadTenderTransferRepository
	WorkflowStarter  kernelports.WorkflowStarter
}

type Service struct {
	auditService     auditports.AuditService
	customFields     customfieldports.NamedValues
	db               kernelports.Transactor
	l                *zap.Logger
	partnerRepo      edipartnerports.EDIPartnerRepository
	savepoints       kernelports.Savepointer
	shipmentLinkRepo editenderports.EDIShipmentLinkRepository
	shipmentSvc      shipmentserviceports.ShipmentWriter
	transferRepo     editenderports.EDILoadTenderTransferRepository
	workflowStarter  kernelports.WorkflowStarter
	outbound         *edioutbound.Service
	recipient        *editenderrecipient.Service
	setup            *edisetup.Service
	shipments        *editendershipment.Service
	tenderMapping    *editendermapping.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		auditService:     p.AuditService,
		customFields:     p.CustomFields,
		db:               p.DB,
		l:                p.Logger.Named("service.edi"),
		partnerRepo:      p.PartnerRepo,
		savepoints:       p.Savepoints,
		shipmentLinkRepo: p.ShipmentLinkRepo,
		shipmentSvc:      p.ShipmentSvc,
		transferRepo:     p.TransferRepo,
		workflowStarter:  p.WorkflowStarter,
		outbound:         p.Outbound,
		recipient:        p.Recipient,
		setup:            p.Setup,
		shipments:        p.Shipments,
		tenderMapping:    p.TenderMapping,
	}
}

func (s *Service) logUpdate(
	entity edilading.Audited,
	actor *permissionports.RequestActor,
	previous any,
	current any,
	comment string,
) {
	edilading.LogAction(s.auditService, s.l, &edilading.AuditEntry{
		Entity:    entity,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  previous,
		Current:   current,
		Comment:   comment,
	})
}
