package ediservice

import (
	"context"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledilifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	PartnerRepo         ediports.EDIPartnerRepository
	MappingProfileRepo  ediports.EDIMappingProfileRepository
	ConnectionRepo      ediports.EDIConnectionRepository
	ProfileRepo         ediports.EDICommunicationProfileRepository
	TransferRepo        ediports.EDILoadTenderTransferRepository
	DocumentTypeRepo    ediports.EDIDocumentTypeRepository
	TransactionSetRepo  ediports.EDITransactionSetRepository
	SourceContextRepo   ediports.EDISourceContextRepository
	PartnerSettingRepo  ediports.EDIPartnerSettingRepository
	TemplateRepo        ediports.EDITemplateRepository
	DocumentProfileRepo ediports.EDIPartnerDocumentProfileRepository
	ControlNumberRepo   ediports.EDIControlNumberRepository
	MessageRepo         ediports.EDIMessageRepository
	TestCaseRepo        ediports.EDITestCaseRepository
	CarrierInvoiceRepo  ediports.EDICarrierInvoiceRepository
	InboundFileRepo     ediports.EDIInboundFileRepository
	InvoiceRepo         invoiceports.InvoiceRepository
	ShipmentEventRepo   shipmenteventports.ShipmentEventRepository
	ServiceFailureRepo  servicefailureports.ServiceFailureRepository
	ShipmentLinkRepo    ediports.EDIShipmentLinkRepository
	TransferChangeRepo  ediports.EDITransferChangeRepository
	TenderRecipientRepo ediports.EDITenderRecipientRepository
	TenderChangeRepo    ediports.EDITenderChangeRepository
	ShipmentCommentRepo shipmentports.ShipmentCommentRepository
	UserRepo            tenantports.UserRepository
	ShipmentRepo        shipmentports.ShipmentRepository
	TenderRepo          tenderports.TenderRepository         `optional:"true"`
	CarrierRepo         carrierports.CarrierRepository       `optional:"true"`
	ShipmentMoveRepo    shipmentports.ShipmentMoveRepository `optional:"true"`
	ShipmentMoves       shipmentports.ShipmentMoveService    `optional:"true"`
	ShipmentSvc         shipmentports.ShipmentService
	WorkflowStarter     kernelports.WorkflowStarter
	AuditService        auditports.AuditService
	Notifications       *notificationservice.Service
	Encryption          *encryptionservice.Service
	Validator           *Validator
	DB                  kernelports.DBConnection
	Coordinator         *shipmentstate.Coordinator
	Transport           ediports.EDITransportDispatcher
	OrderDerivation     orderports.OrderDerivationService
	Metrics             *metrics.Registry             `optional:"true"`
	Plans               platformplanports.PlanService `optional:"true"`
	Realtime            agentports.RealtimeService    `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	partnerRepo         ediports.EDIPartnerRepository
	mappingProfileRepo  ediports.EDIMappingProfileRepository
	connectionRepo      ediports.EDIConnectionRepository
	profileRepo         ediports.EDICommunicationProfileRepository
	transferRepo        ediports.EDILoadTenderTransferRepository
	documentTypeRepo    ediports.EDIDocumentTypeRepository
	transactionSetRepo  ediports.EDITransactionSetRepository
	sourceContextRepo   ediports.EDISourceContextRepository
	partnerSettingRepo  ediports.EDIPartnerSettingRepository
	templateRepo        ediports.EDITemplateRepository
	documentProfileRepo ediports.EDIPartnerDocumentProfileRepository
	controlNumberRepo   ediports.EDIControlNumberRepository
	messageRepo         ediports.EDIMessageRepository
	testCaseRepo        ediports.EDITestCaseRepository
	carrierInvoiceRepo  ediports.EDICarrierInvoiceRepository
	inboundFileRepo     ediports.EDIInboundFileRepository
	invoiceRepo         invoiceports.InvoiceRepository
	realtime            agentports.RealtimeService
	shipmentEventRepo   shipmenteventports.ShipmentEventRepository
	serviceFailureRepo  servicefailureports.ServiceFailureRepository
	shipmentLinkRepo    ediports.EDIShipmentLinkRepository
	transferChangeRepo  ediports.EDITransferChangeRepository
	tenderRecipientRepo ediports.EDITenderRecipientRepository
	tenderChangeRepo    ediports.EDITenderChangeRepository
	shipmentCommentRepo shipmentports.ShipmentCommentRepository
	userRepo            tenantports.UserRepository
	shipmentRepo        shipmentports.ShipmentRepository
	tenderRepo          tenderports.TenderRepository
	carrierRepo         carrierports.CarrierRepository
	shipmentMoveRepo    shipmentports.ShipmentMoveRepository
	shipmentMoves       shipmentports.ShipmentMoveService
	shipmentSvc         shipmentports.ShipmentService
	workflowStarter     kernelports.WorkflowStarter
	auditService        auditports.AuditService
	notifications       *notificationservice.Service
	encryption          *encryptionservice.Service
	validator           *Validator
	db                  kernelports.DBConnection
	coordinator         *shipmentstate.Coordinator
	transport           ediports.EDITransportDispatcher
	orderDerivation     orderports.OrderDerivationService
	lifecycleApplier    *internaledilifecycle.Applier
	metrics             *metrics.EDI
	plans               platformplanports.PlanService
}

func New(p Params) *Service {
	ediMetrics := metrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		l:                   p.Logger.Named("service.edi"),
		metrics:             ediMetrics,
		partnerRepo:         p.PartnerRepo,
		mappingProfileRepo:  p.MappingProfileRepo,
		connectionRepo:      p.ConnectionRepo,
		profileRepo:         p.ProfileRepo,
		transferRepo:        p.TransferRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		transactionSetRepo:  p.TransactionSetRepo,
		sourceContextRepo:   p.SourceContextRepo,
		partnerSettingRepo:  p.PartnerSettingRepo,
		templateRepo:        p.TemplateRepo,
		documentProfileRepo: p.DocumentProfileRepo,
		controlNumberRepo:   p.ControlNumberRepo,
		messageRepo:         p.MessageRepo,
		testCaseRepo:        p.TestCaseRepo,
		carrierInvoiceRepo:  p.CarrierInvoiceRepo,
		inboundFileRepo:     p.InboundFileRepo,
		invoiceRepo:         p.InvoiceRepo,
		realtime:            p.Realtime,
		shipmentEventRepo:   p.ShipmentEventRepo,
		serviceFailureRepo:  p.ServiceFailureRepo,
		shipmentLinkRepo:    p.ShipmentLinkRepo,
		transferChangeRepo:  p.TransferChangeRepo,
		tenderRecipientRepo: p.TenderRecipientRepo,
		tenderChangeRepo:    p.TenderChangeRepo,
		shipmentCommentRepo: p.ShipmentCommentRepo,
		userRepo:            p.UserRepo,
		shipmentRepo:        p.ShipmentRepo,
		tenderRepo:          p.TenderRepo,
		carrierRepo:         p.CarrierRepo,
		shipmentMoveRepo:    p.ShipmentMoveRepo,
		shipmentMoves:       p.ShipmentMoves,
		shipmentSvc:         p.ShipmentSvc,
		workflowStarter:     p.WorkflowStarter,
		auditService:        p.AuditService,
		notifications:       p.Notifications,
		encryption:          p.Encryption,
		validator:           p.Validator,
		db:                  p.DB,
		coordinator:         p.Coordinator,
		transport:           p.Transport,
		orderDerivation:     p.OrderDerivation,
		plans:               p.Plans,
		lifecycleApplier: internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: p.ShipmentRepo,
			Coordinator:  p.Coordinator,
		}),
	}
}

func (s *Service) requireIntegrations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityIntegrations,
	)
}

func mapEDIConnectionConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_connections_internal_open":
		multiErr.Add(
			"targetOrganizationId",
			errortypes.ErrDuplicate,
			"An open internal EDI connection already exists for these organizations",
		)
	default:
		return err
	}

	return multiErr
}

func mapEDICommunicationProfileConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_communication_profiles_name_org":
		multiErr.Add(
			"name",
			errortypes.ErrDuplicate,
			"EDI communication profile with this name already exists",
		)
	case "uq_edi_communication_profiles_active_as2_identifiers":
		multiErr.Add(
			"config.localAS2Id",
			errortypes.ErrDuplicate,
			"Another active AS2 profile already uses this local and partner AS2 ID pair",
		)
	default:
		return err
	}

	return multiErr
}
