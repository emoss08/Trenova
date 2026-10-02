package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/internaledilifecycle"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	PartnerRepo         repositories.EDIPartnerRepository
	MappingProfileRepo  repositories.EDIMappingProfileRepository
	ConnectionRepo      repositories.EDIConnectionRepository
	ProfileRepo         repositories.EDICommunicationProfileRepository
	TransferRepo        repositories.EDILoadTenderTransferRepository
	DocumentTypeRepo    repositories.EDIDocumentTypeRepository
	TransactionSetRepo  repositories.EDITransactionSetRepository
	SourceContextRepo   repositories.EDISourceContextRepository
	PartnerSettingRepo  repositories.EDIPartnerSettingRepository
	TemplateRepo        repositories.EDITemplateRepository
	DocumentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	ControlNumberRepo   repositories.EDIControlNumberRepository
	MessageRepo         repositories.EDIMessageRepository
	TestCaseRepo        repositories.EDITestCaseRepository
	CarrierInvoiceRepo  repositories.EDICarrierInvoiceRepository
	InboundFileRepo     repositories.EDIInboundFileRepository
	InvoiceRepo         repositories.InvoiceRepository
	ShipmentEventRepo   repositories.ShipmentEventRepository
	ServiceFailureRepo  repositories.ServiceFailureRepository
	ShipmentLinkRepo    repositories.EDIShipmentLinkRepository
	TransferChangeRepo  repositories.EDITransferChangeRepository
	TenderRecipientRepo repositories.EDITenderRecipientRepository
	TenderChangeRepo    repositories.EDITenderChangeRepository
	ShipmentCommentRepo repositories.ShipmentCommentRepository
	UserRepo            repositories.UserRepository
	ShipmentRepo        repositories.ShipmentRepository
	TenderRepo          repositories.TenderRepository       `optional:"true"`
	CarrierRepo         repositories.CarrierRepository      `optional:"true"`
	ShipmentMoveRepo    repositories.ShipmentMoveRepository `optional:"true"`
	ShipmentMoves       services.ShipmentMoveService        `optional:"true"`
	Realtime            services.RealtimeService            `optional:"true"`
	ShipmentSvc         services.ShipmentService
	WorkflowStarter     services.WorkflowStarter
	AuditService        services.AuditService
	Notifications       *notificationservice.Service
	Encryption          *encryptionservice.Service
	Validator           *Validator
	DB                  coreports.DBConnection
	Coordinator         *shipmentstate.Coordinator
	Transport           services.EDITransportDispatcher
	OrderDerivation     services.OrderDerivationService
	Metrics             *metrics.Registry `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	partnerRepo         repositories.EDIPartnerRepository
	mappingProfileRepo  repositories.EDIMappingProfileRepository
	connectionRepo      repositories.EDIConnectionRepository
	profileRepo         repositories.EDICommunicationProfileRepository
	transferRepo        repositories.EDILoadTenderTransferRepository
	documentTypeRepo    repositories.EDIDocumentTypeRepository
	transactionSetRepo  repositories.EDITransactionSetRepository
	sourceContextRepo   repositories.EDISourceContextRepository
	partnerSettingRepo  repositories.EDIPartnerSettingRepository
	templateRepo        repositories.EDITemplateRepository
	documentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	controlNumberRepo   repositories.EDIControlNumberRepository
	messageRepo         repositories.EDIMessageRepository
	testCaseRepo        repositories.EDITestCaseRepository
	carrierInvoiceRepo  repositories.EDICarrierInvoiceRepository
	inboundFileRepo     repositories.EDIInboundFileRepository
	invoiceRepo         repositories.InvoiceRepository
	realtime            services.RealtimeService
	shipmentEventRepo   repositories.ShipmentEventRepository
	serviceFailureRepo  repositories.ServiceFailureRepository
	shipmentLinkRepo    repositories.EDIShipmentLinkRepository
	transferChangeRepo  repositories.EDITransferChangeRepository
	tenderRecipientRepo repositories.EDITenderRecipientRepository
	tenderChangeRepo    repositories.EDITenderChangeRepository
	shipmentCommentRepo repositories.ShipmentCommentRepository
	userRepo            repositories.UserRepository
	shipmentRepo        repositories.ShipmentRepository
	tenderRepo          repositories.TenderRepository
	carrierRepo         repositories.CarrierRepository
	shipmentMoveRepo    repositories.ShipmentMoveRepository
	shipmentMoves       services.ShipmentMoveService
	shipmentSvc         services.ShipmentService
	workflowStarter     services.WorkflowStarter
	auditService        services.AuditService
	notifications       *notificationservice.Service
	encryption          *encryptionservice.Service
	validator           *Validator
	db                  coreports.DBConnection
	coordinator         *shipmentstate.Coordinator
	transport           services.EDITransportDispatcher
	orderDerivation     services.OrderDerivationService
	lifecycleApplier    *internaledilifecycle.Applier
	metrics             *metrics.EDI
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
		lifecycleApplier: internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: p.ShipmentRepo,
			Coordinator:  p.Coordinator,
		}),
	}
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
