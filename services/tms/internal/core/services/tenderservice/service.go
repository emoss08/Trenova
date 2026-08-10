package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
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

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    ports.DBConnection
	Repo                  repositories.TenderRepository
	GuideRepo             repositories.RoutingGuideRepository
	CarrierRepo           repositories.CarrierRepository
	CarrierAssignmentRepo repositories.CarrierAssignmentRepository
	AssignmentRepo        repositories.AssignmentRepository
	ShipmentRepo          repositories.ShipmentRepository
	HoldRepo              repositories.ShipmentHoldRepository
	Workflows             portservices.WorkflowStarter
	EventService          portservices.ShipmentEventService
	Realtime              portservices.RealtimeService
}

type Service struct {
	l                     *zap.Logger
	db                    ports.DBConnection
	repo                  repositories.TenderRepository
	guideRepo             repositories.RoutingGuideRepository
	carrierRepo           repositories.CarrierRepository
	carrierAssignmentRepo repositories.CarrierAssignmentRepository
	assignmentRepo        repositories.AssignmentRepository
	shipmentRepo          repositories.ShipmentRepository
	holdRepo              repositories.ShipmentHoldRepository
	workflows             portservices.WorkflowStarter
	eventService          portservices.ShipmentEventService
	realtime              portservices.RealtimeService
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
		workflows:             p.Workflows,
		eventService:          p.EventService,
		realtime:              p.Realtime,
	}
}

var (
	_ portservices.TenderGuard            = (*Service)(nil)
	_ portservices.TenderResponseRecorder = (*Service)(nil)
)

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetTenderByIDRequest,
) (*tender.Tender, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetOffer(
	ctx context.Context,
	req repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return s.repo.GetOfferByID(ctx, req)
}

func (s *Service) ListByShipment(
	ctx context.Context,
	req repositories.ListTendersByShipmentRequest,
) ([]*tender.Tender, error) {
	return s.repo.ListByShipment(ctx, req)
}

func (s *Service) GetLiveByMoveID(
	ctx context.Context,
	req repositories.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return s.repo.GetLiveByMoveID(ctx, req)
}

func (s *Service) recordEvent(
	ctx context.Context,
	params *portservices.RecordShipmentEventParams,
) {
	if params == nil || params.ShipmentID.IsNil() {
		return
	}

	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record tender event", zap.Error(err))
	}
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	action string,
) {
	if s.realtime == nil || shipmentID.IsNil() {
		return
	}

	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		ActorType:      portservices.PrincipalTypeUser,
		ActorID:        tenantInfo.UserID,
		Resource:       "shipments",
		Action:         action,
		RecordID:       shipmentID,
	})
	if err != nil {
		s.l.Warn("failed to publish tender invalidation", zap.Error(err))
	}
}
