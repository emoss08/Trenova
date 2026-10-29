package edioutbound

import (
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Alert             *edialert.Service
	AuditService      auditports.AuditService
	ControlNumberRepo edipartnerports.EDIControlNumberRepository
	CustomFields      customfieldports.NamedValues `optional:"true"`
	Delivery          *edidelivery.Service
	DocumentTypeRepo  edipartnerports.EDIDocumentTypeRepository
	FlowRepo          ladingports.FlowRepository
	Flows             *edilading.Flows
	InvoiceRepo       invoiceports.InvoiceRepository
	Lading            *edilading.Runtime
	LocationRepo      locationports.LocationRepository `optional:"true"`
	Logger            *zap.Logger
	MessageRepo       edimessageports.EDIMessageRepository
	PartnerRepo       edipartnerports.EDIPartnerRepository
	Payload           *edipayloadbuild.Service
	Recipient         *editenderrecipient.Service
	Replayer          *edilading.Replayer
	ShipmentEventRepo shipmenteventports.ShipmentEventRepository
	ShipmentSvc       shipmentserviceports.ShipmentService
	TransferRepo      editenderports.EDILoadTenderTransferRepository
}

type Service struct {
	auditService      auditports.AuditService
	controlNumberRepo edipartnerports.EDIControlNumberRepository
	customFields      customfieldports.NamedValues
	documentTypeRepo  edipartnerports.EDIDocumentTypeRepository
	flowRepo          ladingports.FlowRepository
	flows             *edilading.Flows
	invoiceRepo       invoiceports.InvoiceRepository
	l                 *zap.Logger
	lading            *edilading.Runtime
	locationRepo      locationports.LocationRepository
	messageRepo       edimessageports.EDIMessageRepository
	partnerRepo       edipartnerports.EDIPartnerRepository
	replayer          *edilading.Replayer
	shipmentEventRepo shipmenteventports.ShipmentEventRepository
	shipmentSvc       shipmentserviceports.ShipmentService
	transferRepo      editenderports.EDILoadTenderTransferRepository
	alert             *edialert.Service
	delivery          *edidelivery.Service
	payload           *edipayloadbuild.Service
	recipient         *editenderrecipient.Service
}

func New(p Params) *Service {
	return &Service{
		auditService:      p.AuditService,
		controlNumberRepo: p.ControlNumberRepo,
		customFields:      p.CustomFields,
		documentTypeRepo:  p.DocumentTypeRepo,
		flowRepo:          p.FlowRepo,
		flows:             p.Flows,
		invoiceRepo:       p.InvoiceRepo,
		l:                 p.Logger.Named("service.edi"),
		lading:            p.Lading,
		locationRepo:      p.LocationRepo,
		messageRepo:       p.MessageRepo,
		partnerRepo:       p.PartnerRepo,
		replayer:          p.Replayer,
		shipmentEventRepo: p.ShipmentEventRepo,
		shipmentSvc:       p.ShipmentSvc,
		transferRepo:      p.TransferRepo,
		alert:             p.Alert,
		delivery:          p.Delivery,
		payload:           p.Payload,
		recipient:         p.Recipient,
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
