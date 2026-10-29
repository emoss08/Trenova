package editenderservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AuditService   auditports.AuditService
	CarrierRepo    carrierports.CarrierRepository `optional:"true"`
	ConnectionRepo edipartnerports.EDIConnectionRepository
	DB             kernelports.Transactor
	Logger         *zap.Logger
	Outbound       *edioutbound.Service
	PartnerRepo    edipartnerports.EDIPartnerRepository
	ProfileRepo    edipartnerports.EDICommunicationProfileRepository
	Recipient      *editenderrecipient.Service
	ShipmentSvc    shipmentserviceports.ShipmentReader
	Shipments      *editendershipment.Service
	TenderRepo     tenderports.TenderRepository `optional:"true"`
	TenderMapping  *editendermapping.Service
	TransferRepo   editenderports.EDILoadTenderTransferRepository
}

type Service struct {
	auditService   auditports.AuditService
	carrierRepo    carrierports.CarrierRepository
	connectionRepo edipartnerports.EDIConnectionRepository
	db             kernelports.Transactor
	l              *zap.Logger
	partnerRepo    edipartnerports.EDIPartnerRepository
	profileRepo    edipartnerports.EDICommunicationProfileRepository
	shipmentSvc    shipmentserviceports.ShipmentReader
	tenderRepo     tenderports.TenderRepository
	transferRepo   editenderports.EDILoadTenderTransferRepository
	outbound       *edioutbound.Service
	recipient      *editenderrecipient.Service
	shipments      *editendershipment.Service
	tenderMapping  *editendermapping.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		auditService:   p.AuditService,
		carrierRepo:    p.CarrierRepo,
		connectionRepo: p.ConnectionRepo,
		db:             p.DB,
		l:              p.Logger.Named("service.edi"),
		partnerRepo:    p.PartnerRepo,
		profileRepo:    p.ProfileRepo,
		shipmentSvc:    p.ShipmentSvc,
		tenderRepo:     p.TenderRepo,
		transferRepo:   p.TransferRepo,
		outbound:       p.Outbound,
		recipient:      p.Recipient,
		shipments:      p.Shipments,
		tenderMapping:  p.TenderMapping,
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
