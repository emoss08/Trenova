package ediservicetest

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundapply"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edimessageview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservicefailure"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"go.uber.org/zap"
)

type Deps struct {
	Logger              *zap.Logger
	PartnerRepo         edipartnerports.EDIPartnerRepository
	MappingProfileRepo  edipartnerports.EDIMappingProfileRepository
	ConnectionRepo      edipartnerports.EDIConnectionRepository
	ProfileRepo         edipartnerports.EDICommunicationProfileRepository
	TransferRepo        editenderports.EDILoadTenderTransferRepository
	DocumentTypeRepo    edipartnerports.EDIDocumentTypeRepository
	TransactionSetRepo  edipartnerports.EDITransactionSetRepository
	ControlNumberRepo   edipartnerports.EDIControlNumberRepository
	MessageRepo         edimessageports.EDIMessageRepository
	FlowRepo            ladingports.FlowRepository
	Lading              *edilading.Runtime
	Flows               *edilading.Flows
	Replayer            *edilading.Replayer
	LocationRepo        locationports.LocationRepository
	CarrierInvoiceRepo  ediports.EDICarrierInvoiceRepository
	InboundFileRepo     edimessageports.EDIInboundFileRepository
	InvoiceRepo         invoiceports.InvoiceRepository
	ShipmentEventRepo   shipmenteventports.ShipmentEventRepository
	ServiceFailureRepo  servicefailureports.ServiceFailureRepository
	ShipmentLinkRepo    editenderports.EDIShipmentLinkRepository
	TransferChangeRepo  editenderports.EDITransferChangeRepository
	TenderRecipientRepo editenderports.EDITenderRecipientRepository
	TenderChangeRepo    editenderports.EDITenderChangeRepository
	ShipmentCommentRepo shipmentcommentports.ShipmentCommentRepository
	UserRepo            tenantports.UserRepository
	ShipmentRepo        shipmentports.ShipmentRepository
	TenderRepo          tenderports.TenderRepository
	CarrierRepo         carrierports.CarrierRepository
	ShipmentMoveRepo    shipmentmoveports.ShipmentMoveRepository
	TrackingEvents      trackingeventports.TrackingEventService
	ShipmentSvc         shipmentserviceports.ShipmentService
	WorkflowStarter     kernelports.WorkflowStarter
	AuditService        auditports.AuditService
	Notifications       *notificationservice.Service
	Encryption          *encryptionservice.Service
	Validator           *edivalidation.Validator
	DB                  kernelports.Transactor
	Savepoints          kernelports.Savepointer
	ShipmentTenderRepo  editenderports.EDIShipmentTenderRepository
	Coordinator         *shipmentstate.Coordinator
	Transport           editransportports.EDITransportDispatcher
	OrderDerivation     orderports.OrderDerivationService
	Metrics             *metrics.Registry
	Plans               platformplanports.PlanService
	Realtime            realtimeports.RealtimeService
	CustomFields        customfieldports.NamedValues
}

type Services struct {
	Alerts          *edialert.Service
	Secrets         *ediprofilesecret.Service
	Recipients      *editenderrecipient.Service
	Shipments       *editendershipment.Service
	Payloads        *edipayloadbuild.Service
	TenderMapping   *editendermapping.Service
	Messages        *edimessageview.Service
	Delivery        *edidelivery.Service
	Setup           *edisetup.Service
	Outbound        *edioutbound.Service
	ServiceFailure  *ediservicefailure.Service
	Overview        *edioverview.Service
	InboundApply    *ediinboundapply.Service
	TransferChanges *editransferchangeservice.Service
	Tenders         *editenderservice.Service
	TenderChanges   *editenderchangeservice.Service
	Transfers       *editransferservice.Service
}

func New(d Deps) *Services {
	s := &Services{}
	s.Alerts = edialert.New(edialert.Params{
		Logger:        d.Logger,
		Notifications: d.Notifications,
	})
	s.Secrets = ediprofilesecret.New(ediprofilesecret.Params{
		Encryption: d.Encryption,
	})
	s.Recipients = editenderrecipient.New(editenderrecipient.Params{
		ProfileRepo:         d.ProfileRepo,
		TenderChangeRepo:    d.TenderChangeRepo,
		TenderRecipientRepo: d.TenderRecipientRepo,
	})
	s.Shipments = editendershipment.New(editendershipment.Params{
		ShipmentCommentRepo: d.ShipmentCommentRepo,
		ShipmentTenderRepo:  d.ShipmentTenderRepo,
		UserRepo:            d.UserRepo,
	})
	s.Payloads = edipayloadbuild.New(edipayloadbuild.Params{
		ServiceFailureRepo: d.ServiceFailureRepo,
		ShipmentSvc:        d.ShipmentSvc,
	})
	s.TenderMapping = editendermapping.New(editendermapping.Params{
		MappingProfileRepo: d.MappingProfileRepo,
	})
	s.Messages = edimessageview.New(edimessageview.Params{
		MessageRepo: d.MessageRepo,
	})
	s.Delivery = edidelivery.New(edidelivery.Params{
		Alert:           s.Alerts,
		InvoiceRepo:     d.InvoiceRepo,
		Logger:          d.Logger,
		MessageRepo:     d.MessageRepo,
		Metrics:         d.Metrics,
		PartnerRepo:     d.PartnerRepo,
		ProfileRepo:     d.ProfileRepo,
		Realtime:        d.Realtime,
		Recipient:       s.Recipients,
		Secret:          s.Secrets,
		Transport:       d.Transport,
		WorkflowStarter: d.WorkflowStarter,
	})
	s.Setup = edisetup.New(edisetup.Params{
		AuditService:       d.AuditService,
		ConnectionRepo:     d.ConnectionRepo,
		ControlNumberRepo:  d.ControlNumberRepo,
		DocumentTypeRepo:   d.DocumentTypeRepo,
		Encryption:         d.Encryption,
		Logger:             d.Logger,
		MappingProfileRepo: d.MappingProfileRepo,
		PartnerRepo:        d.PartnerRepo,
		Plans:              d.Plans,
		ProfileRepo:        d.ProfileRepo,
		Secret:             s.Secrets,
		TransactionSetRepo: d.TransactionSetRepo,
		Transport:          d.Transport,
		Validator:          d.Validator,
	})
	s.Outbound = edioutbound.New(edioutbound.Params{
		Alert:             s.Alerts,
		AuditService:      d.AuditService,
		ControlNumberRepo: d.ControlNumberRepo,
		CustomFields:      d.CustomFields,
		Delivery:          s.Delivery,
		DocumentTypeRepo:  d.DocumentTypeRepo,
		FlowRepo:          d.FlowRepo,
		Flows:             d.Flows,
		InvoiceRepo:       d.InvoiceRepo,
		Lading:            d.Lading,
		LocationRepo:      d.LocationRepo,
		Logger:            d.Logger,
		MessageRepo:       d.MessageRepo,
		PartnerRepo:       d.PartnerRepo,
		Payload:           s.Payloads,
		Recipient:         s.Recipients,
		Replayer:          d.Replayer,
		ShipmentEventRepo: d.ShipmentEventRepo,
		ShipmentSvc:       d.ShipmentSvc,
		TransferRepo:      d.TransferRepo,
	})
	s.ServiceFailure = ediservicefailure.New(ediservicefailure.Params{
		ConnectionRepo:     d.ConnectionRepo,
		Lading:             d.Lading,
		MappingProfileRepo: d.MappingProfileRepo,
		MessageRepo:        d.MessageRepo,
		Outbound:           s.Outbound,
		PartnerRepo:        d.PartnerRepo,
		ServiceFailureRepo: d.ServiceFailureRepo,
		ShipmentSvc:        d.ShipmentSvc,
	})
	s.Overview = edioverview.New(edioverview.Params{
		CarrierRepo:        d.CarrierRepo,
		InboundFileRepo:    d.InboundFileRepo,
		Lading:             d.Lading,
		MappingProfileRepo: d.MappingProfileRepo,
		MessageRepo:        d.MessageRepo,
		PartnerRepo:        d.PartnerRepo,
		ProfileRepo:        d.ProfileRepo,
		TransferRepo:       d.TransferRepo,
	})
	s.InboundApply = ediinboundapply.New(ediinboundapply.Params{
		CarrierInvoiceRepo:  d.CarrierInvoiceRepo,
		Logger:              d.Logger,
		MappingProfileRepo:  d.MappingProfileRepo,
		ShipmentEventRepo:   d.ShipmentEventRepo,
		ShipmentMoveRepo:    d.ShipmentMoveRepo,
		ShipmentSvc:         d.ShipmentSvc,
		Shipments:           s.Shipments,
		TenderRecipientRepo: d.TenderRecipientRepo,
		TenderMapping:       s.TenderMapping,
		TrackingEvents:      d.TrackingEvents,
		TransferRepo:        d.TransferRepo,
	})
	s.TransferChanges = editransferchangeservice.New(editransferchangeservice.Params{
		AuditService:       d.AuditService,
		Coordinator:        d.Coordinator,
		DB:                 d.DB,
		Logger:             d.Logger,
		OrderDerivation:    d.OrderDerivation,
		ShipmentEventRepo:  d.ShipmentEventRepo,
		ShipmentLinkRepo:   d.ShipmentLinkRepo,
		ShipmentRepo:       d.ShipmentRepo,
		Shipments:          s.Shipments,
		TransferChangeRepo: d.TransferChangeRepo,
	})
	s.Tenders = editenderservice.New(editenderservice.Params{
		AuditService:   d.AuditService,
		CarrierRepo:    d.CarrierRepo,
		ConnectionRepo: d.ConnectionRepo,
		DB:             d.DB,
		Logger:         d.Logger,
		Outbound:       s.Outbound,
		PartnerRepo:    d.PartnerRepo,
		ProfileRepo:    d.ProfileRepo,
		Recipient:      s.Recipients,
		ShipmentSvc:    d.ShipmentSvc,
		Shipments:      s.Shipments,
		TenderRepo:     d.TenderRepo,
		TenderMapping:  s.TenderMapping,
		TransferRepo:   d.TransferRepo,
	})
	s.TenderChanges = editenderchangeservice.New(editenderchangeservice.Params{
		DB:                  d.DB,
		Delivery:            s.Delivery,
		Logger:              d.Logger,
		Outbound:            s.Outbound,
		PartnerRepo:         d.PartnerRepo,
		Recipient:           s.Recipients,
		ShipmentLinkRepo:    d.ShipmentLinkRepo,
		ShipmentSvc:         d.ShipmentSvc,
		TenderChangeRepo:    d.TenderChangeRepo,
		TenderRecipientRepo: d.TenderRecipientRepo,
		TenderMapping:       s.TenderMapping,
		TransferRepo:        d.TransferRepo,
	})
	s.Transfers = editransferservice.New(editransferservice.Params{
		AuditService:     d.AuditService,
		CustomFields:     d.CustomFields,
		DB:               d.DB,
		Logger:           d.Logger,
		Outbound:         s.Outbound,
		PartnerRepo:      d.PartnerRepo,
		Recipient:        s.Recipients,
		Savepoints:       d.Savepoints,
		Setup:            s.Setup,
		ShipmentLinkRepo: d.ShipmentLinkRepo,
		ShipmentSvc:      d.ShipmentSvc,
		Shipments:        s.Shipments,
		TenderMapping:    s.TenderMapping,
		TransferRepo:     d.TransferRepo,
		WorkflowStarter:  d.WorkflowStarter,
	})
	return s
}
