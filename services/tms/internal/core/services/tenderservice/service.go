package tenderservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrOfferNoLongerAvailable is returned to any channel whose response arrived
// after the offer left its live window. The copy is deliberately identical for
// expired, superseded, withdrawn, and already-answered offers.
var ErrOfferNoLongerAvailable = errortypes.NewBusinessError(
	"This offer is no longer available",
)

var ErrNoRoutingGuideMatch = errors.New("no routing guide matches the lane")

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    kernelports.DBConnection
	Repo                  tenderports.TenderRepository
	GuideRepo             tenderports.RoutingGuideRepository
	CarrierRepo           carrierports.CarrierRepository
	CarrierAssignmentRepo shipmentports.CarrierAssignmentRepository
	AssignmentRepo        shipmentports.AssignmentRepository
	ShipmentRepo          shipmentports.ShipmentRepository
	HoldRepo              shipmentports.ShipmentHoldRepository
	OrgRepo               tenantports.OrganizationRepository
	Workflows             kernelports.WorkflowStarter
	EventService          shipmenteventports.ShipmentEventService
	AuditService          auditports.AuditService
	Invalidator           shipmentports.ShipmentInvalidator
	Config                *config.Config
	Templates             documenttemplateports.DocumentTemplateResolver `optional:"true"`
	EmailService          emailports.EmailService                        `optional:"true"`
	Notifications         *notificationservice.Service                   `optional:"true"`
	EDIChannel            tenderports.EDITenderChannel                   `optional:"true"`
	RateConIssuer         rateconfirmationports.RateConfirmationIssuer   `optional:"true"`
	RateEngine            ratequoteports.RateEngine                      `optional:"true"`
}

type Service struct {
	l                     *zap.Logger
	db                    kernelports.DBConnection
	repo                  tenderports.TenderRepository
	guideRepo             tenderports.RoutingGuideRepository
	carrierRepo           carrierports.CarrierRepository
	carrierAssignmentRepo shipmentports.CarrierAssignmentRepository
	assignmentRepo        shipmentports.AssignmentRepository
	shipmentRepo          shipmentports.ShipmentRepository
	holdRepo              shipmentports.ShipmentHoldRepository
	orgRepo               tenantports.OrganizationRepository
	workflows             kernelports.WorkflowStarter
	eventService          shipmenteventports.ShipmentEventService
	auditService          auditports.AuditService
	invalidator           shipmentports.ShipmentInvalidator
	cfg                   *config.Config
	templates             documenttemplateports.DocumentTemplateResolver
	emailService          emailports.EmailService
	notifications         *notificationservice.Service
	assigner              tenderports.CarrierMoveAssigner
	ediChannel            tenderports.EDITenderChannel
	rateConIssuer         rateconfirmationports.RateConfirmationIssuer
	rateEngine            ratequoteports.RateEngine
	intelGate             carrierintelports.CarrierIntelGate
	lifecycle             carrierintelports.CarrierLifecycleObserver
}

func New(p Params) *Service {
	return &Service{
		l:                     p.Logger.Named("service.tender"),
		db:                    p.DB,
		repo:                  p.Repo,
		guideRepo:             p.GuideRepo,
		carrierRepo:           p.CarrierRepo,
		carrierAssignmentRepo: p.CarrierAssignmentRepo,
		assignmentRepo:        p.AssignmentRepo,
		shipmentRepo:          p.ShipmentRepo,
		holdRepo:              p.HoldRepo,
		orgRepo:               p.OrgRepo,
		workflows:             p.Workflows,
		eventService:          p.EventService,
		auditService:          p.AuditService,
		invalidator:           p.Invalidator,
		cfg:                   p.Config,
		templates:             p.Templates,
		emailService:          p.EmailService,
		notifications:         p.Notifications,
		ediChannel:            p.EDIChannel,
		rateConIssuer:         p.RateConIssuer,
		rateEngine:            p.RateEngine,
	}
}

// AssignerSetter breaks the construction cycle between the tender service
// (which needs the assigner to land accepted offers) and the carrier
// assignment service (which needs the tender guard to withdraw dead tenders):
// the assigner arrives after both are built.
type AssignerSetter interface {
	SetCarrierMoveAssigner(assigner tenderports.CarrierMoveAssigner)
}

func (s *Service) SetCarrierMoveAssigner(assigner tenderports.CarrierMoveAssigner) {
	s.assigner = assigner
}

func (s *Service) SetIntelGate(gate carrierintelports.CarrierIntelGate) {
	s.intelGate = gate
}

func (s *Service) SetLifecycleObserver(observer carrierintelports.CarrierLifecycleObserver) {
	s.lifecycle = observer
}

func (s *Service) intelGates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
	refresh bool,
) map[pulid.ID]*carrier.IntelGate {
	if s.intelGate == nil || len(carrierIDs) == 0 {
		return nil
	}
	if refresh {
		s.intelGate.EnsureFresh(ctx, &carrierintelports.CarrierIntelEnsureFreshRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: carrierIDs,
		})
	}
	gates, err := s.intelGate.GateFor(ctx, tenantInfo, carrierIDs)
	if err != nil {
		s.l.Warn("failed to evaluate carrier intelligence gates", zap.Error(err))
		return nil
	}
	return gates
}

// SetEDITenderChannel injects the EDI delivery seam after construction:
// providing it through FX would close the cycle ediservice -> ShipmentService
// -> TenderGuard -> tenderservice -> EDITenderChannel.
func (s *Service) SetEDITenderChannel(channel tenderports.EDITenderChannel) {
	s.ediChannel = channel
}

var (
	_ tenderports.TenderGuard            = (*Service)(nil)
	_ tenderports.TenderResponseRecorder = (*Service)(nil)
	_ tenderports.TenderLifecycle        = (*Service)(nil)
)

func (s *Service) Get(
	ctx context.Context,
	req tenderports.GetTenderByIDRequest,
) (*tender.Tender, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetOffer(
	ctx context.Context,
	req tenderports.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return s.repo.GetOfferByID(ctx, req)
}

func (s *Service) ListByShipment(
	ctx context.Context,
	req tenderports.ListTendersByShipmentRequest,
) ([]*tender.Tender, error) {
	return s.repo.ListByShipment(ctx, req)
}

func (s *Service) GetLiveByMoveID(
	ctx context.Context,
	req tenderports.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return s.repo.GetLiveByMoveID(ctx, req)
}

func (s *Service) recordEvent(
	ctx context.Context,
	params *shipmenteventports.RecordShipmentEventParams,
) {
	if params == nil || params.ShipmentID.IsNil() {
		return
	}

	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record tender event", zap.Error(err))
	}
}
