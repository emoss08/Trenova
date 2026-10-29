package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderdispatch"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderescalation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderlifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    kernelports.Transactor
	Repo                  tenderports.TenderRepository
	GuideRepo             tenderports.RoutingGuideRepository
	CarrierRepo           carrierports.CarrierRepository
	CarrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	AssignmentRepo        assignmentports.AssignmentRepository
	ShipmentRepo          shipmentports.ShipmentRepository
	HoldRepo              shipmentholdports.ShipmentHoldRepository
	OrgRepo               tenantports.OrganizationRepository
	Workflows             kernelports.WorkflowStarter
	EventService          shipmenteventports.ShipmentEventService
	AuditService          auditports.AuditService
	Invalidator           shipmentinvalidationports.ShipmentInvalidator
	AppConfig             *appconfig.Config
	TenderingConfig       *tenderingconfig.Config
	Templates             documenttemplateports.DocumentTemplateResolver `optional:"true"`
	EmailService          emailports.EmailService                        `optional:"true"`
	Notifications         *notificationservice.Service                   `optional:"true"`
	EDIChannel            tenderports.EDITenderChannel                   `optional:"true"`
	RateConIssuer         rateconfirmationports.RateConfirmationIssuer   `optional:"true"`
	RateEngine            ratequoteports.RateEngine                      `optional:"true"`
	DispatchControls      dispatchcontrolports.DispatchControlRepository `optional:"true"`
	Permissions           permissionports.PermissionEngine               `optional:"true"`
}

type Service struct {
	l                     *zap.Logger
	db                    kernelports.Transactor
	repo                  tenderports.TenderRepository
	guideRepo             tenderports.RoutingGuideRepository
	carrierRepo           carrierports.CarrierRepository
	carrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	assignmentRepo        assignmentports.AssignmentRepository
	shipmentRepo          shipmentports.ShipmentRepository
	holdRepo              shipmentholdports.ShipmentHoldRepository
	orgRepo               tenantports.OrganizationRepository
	workflows             kernelports.WorkflowStarter
	eventService          shipmenteventports.ShipmentEventService
	auditService          auditports.AuditService
	invalidator           shipmentinvalidationports.ShipmentInvalidator
	appConfig             *appconfig.Config
	tenderingConfig       *tenderingconfig.Config
	templates             documenttemplateports.DocumentTemplateResolver
	emailService          emailports.EmailService
	notifications         *notificationservice.Service
	assigner              tenderports.CarrierMoveAssigner
	ediChannel            tenderports.EDITenderChannel
	rateConIssuer         rateconfirmationports.RateConfirmationIssuer
	rateEngine            ratequoteports.RateEngine
	dispatchControls      dispatchcontrolports.DispatchControlRepository
	permissions           permissionports.PermissionEngine
	intelGate             carrierintelports.CarrierIntelGate
	lifecycle             carrierintelports.CarrierLifecycleObserver
	effects               *tendereffects.Effects
	creator               *tendercreate.Service
	dispatcher            *tenderdispatch.Service
	negotiator            *tendernegotiation.Service
	bouncer               *tenderbounce.Service
	canceler              *tendercancel.Service
	escalator             *tenderescalation.Service
	offerLifecycle        *tenderlifecycle.Service
	responder             *tenderresponse.Service
}

func New(p Params) *Service {
	s := &Service{
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
		appConfig:             p.AppConfig,
		tenderingConfig:       p.TenderingConfig,
		templates:             p.Templates,
		emailService:          p.EmailService,
		notifications:         p.Notifications,
		ediChannel:            p.EDIChannel,
		rateConIssuer:         p.RateConIssuer,
		rateEngine:            p.RateEngine,
		dispatchControls:      p.DispatchControls,
		permissions:           p.Permissions,
	}
	s.assemble()
	return s
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
	s.bouncer.SetCarrierMoveAssigner(assigner)
	s.offerLifecycle.SetCarrierMoveAssigner(assigner)
}

func (s *Service) SetIntelGate(gate carrierintelports.CarrierIntelGate) {
	s.intelGate = gate
	s.effects.SetIntelGate(gate)
	s.dispatcher.SetIntelGate(gate)
}

func (s *Service) SetLifecycleObserver(observer carrierintelports.CarrierLifecycleObserver) {
	s.lifecycle = observer
	s.effects.SetLifecycleObserver(observer)
}

// SetEDITenderChannel injects the EDI delivery seam after construction:
// providing it through FX would close the cycle editenderservice -> ShipmentService
// -> TenderGuard -> tenderservice -> EDITenderChannel.
func (s *Service) SetEDITenderChannel(channel tenderports.EDITenderChannel) {
	s.ediChannel = channel
	s.dispatcher.SetEDITenderChannel(channel)
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

func (s *Service) CreateWaterfall(
	ctx context.Context,
	req *tendercreate.CreateWaterfallTenderRequest,
) (*tendercreate.CreateWaterfallResult, error) {
	return s.creator.CreateWaterfall(ctx, req)
}

func (s *Service) CreateSpot(
	ctx context.Context,
	req *tendercreate.CreateSpotTenderRequest,
) (*tender.Tender, error) {
	return s.creator.CreateSpot(ctx, req)
}

func (s *Service) PreviewWaterfall(
	ctx context.Context,
	req *tendercreate.CreateWaterfallTenderRequest,
) (*tendercreate.TenderPreview, error) {
	return s.creator.PreviewWaterfall(ctx, req)
}

func (s *Service) PreviewSpot(
	ctx context.Context,
	req *tendercreate.CreateSpotTenderRequest,
) (*tendercreate.TenderPreview, error) {
	return s.creator.PreviewSpot(ctx, req)
}

func (s *Service) DispatchOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) (*tenderports.TenderDispatchResult, error) {
	return s.dispatcher.DispatchOffer(ctx, tenantInfo, offerID)
}

func (s *Service) RecordCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *tenderports.RecordCounterRequest,
) (*tenderports.TenderCounterResult, error) {
	return s.negotiator.RecordCounter(ctx, tenantInfo, req)
}

func (s *Service) DecideCounter(
	ctx context.Context,
	req *tendernegotiation.DecideCounterRequest,
) error {
	return s.negotiator.DecideCounter(ctx, req)
}

func (s *Service) PreviewDecideCounter(
	ctx context.Context,
	req *tendernegotiation.DecideCounterRequest,
) (*tendernegotiation.ResponsePreview, error) {
	return s.negotiator.PreviewDecideCounter(ctx, req)
}

func (s *Service) PreviewBounce(
	ctx context.Context,
	req *tenderbounce.BounceTenderRequest,
) (*tenderbounce.BouncePreview, error) {
	return s.bouncer.PreviewBounce(ctx, req)
}

func (s *Service) Bounce(
	ctx context.Context,
	req *tenderbounce.BounceTenderRequest,
) (*tender.Tender, error) {
	return s.bouncer.Bounce(ctx, req)
}

func (s *Service) Cancel(ctx context.Context, req *tendercancel.CancelTenderRequest) error {
	return s.canceler.Cancel(ctx, req)
}

func (s *Service) CancelLiveTenderForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	reason string,
) error {
	return s.canceler.CancelLiveTenderForMove(ctx, tenantInfo, moveID, reason)
}

func (s *Service) CancelLiveTendersForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	reason string,
) error {
	return s.canceler.CancelLiveTendersForShipment(ctx, tenantInfo, shipmentID, reason)
}

func (s *Service) CancelDirect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
	reason string,
) error {
	return s.canceler.CancelDirect(ctx, tenantInfo, entity, reason)
}

func (s *Service) PreviewCancel(
	ctx context.Context,
	req *tendercancel.CancelTenderRequest,
) (*tendercancel.CancelPreview, error) {
	return s.canceler.PreviewCancel(ctx, req)
}

func (s *Service) PreviewLiveTenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*tender.Tender, error) {
	return s.canceler.PreviewLiveTenders(ctx, tenantInfo, shipmentID)
}

func (s *Service) LoadPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
) (*tenderports.TenderPlan, error) {
	return s.offerLifecycle.LoadPlan(ctx, tenantInfo, tenderID)
}

func (s *Service) ExpireOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) error {
	return s.offerLifecycle.ExpireOffer(ctx, tenantInfo, offerID)
}

func (s *Service) DeclineOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	source tender.ResponseSource,
	reason string,
) error {
	return s.offerLifecycle.DeclineOffer(ctx, tenantInfo, offerID, source, reason)
}

func (s *Service) AcceptOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	source tender.ResponseSource,
) (*tenderports.TenderAcceptResult, error) {
	return s.offerLifecycle.AcceptOffer(ctx, tenantInfo, offerID, source)
}

func (s *Service) FinalizeAccepted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	acceptedOfferID pulid.ID,
) error {
	return s.offerLifecycle.FinalizeAccepted(ctx, tenantInfo, tenderID, acceptedOfferID)
}

func (s *Service) MarkNeedsReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	offerID pulid.ID,
	reason string,
) error {
	return s.offerLifecycle.MarkNeedsReview(ctx, tenantInfo, tenderID, offerID, reason)
}

func (s *Service) MarkExhausted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
) error {
	return s.offerLifecycle.MarkExhausted(ctx, tenantInfo, tenderID)
}

func (s *Service) IssueRateConfirmation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) error {
	return s.offerLifecycle.IssueRateConfirmation(ctx, tenantInfo, offerID)
}

func (s *Service) CancelFromWorkflow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	reason string,
	actorUserID pulid.ID,
) error {
	return s.offerLifecycle.CancelFromWorkflow(ctx, tenantInfo, tenderID, reason, actorUserID)
}

func (s *Service) RecordResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) error {
	return s.responder.RecordResponse(ctx, req)
}

func (s *Service) RecordLateResponse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	action tender.ResponseAction,
	source tender.ResponseSource,
) error {
	return s.responder.RecordLateResponse(ctx, tenantInfo, offerID, action, source)
}

func (s *Service) PreviewResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (*tendernegotiation.ResponsePreview, error) {
	return s.responder.PreviewResponse(ctx, req)
}

func (s *Service) PreviewByToken(
	ctx context.Context,
	rawToken string,
) (*tenderresponse.PublicOfferView, error) {
	return s.responder.PreviewByToken(ctx, rawToken)
}

func (s *Service) RespondByToken(
	ctx context.Context,
	rawToken string,
	response tenderresponse.PublicResponse,
) error {
	return s.responder.RespondByToken(ctx, rawToken, response)
}
