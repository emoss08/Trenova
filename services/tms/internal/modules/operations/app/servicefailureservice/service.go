package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	Repo            servicefailureports.ServiceFailureRepository
	ReasonCodeRepo  servicefailureports.ServiceFailureReasonCodeRepository
	ShipmentRepo    shipmentports.ShipmentRepository
	DispatchRepo    dispatchcontrolports.DispatchControlRepository
	CommentService  shipmentports.ShipmentCommentService
	AuditService    auditports.AuditService
	Realtime        agentports.RealtimeService
	OrderDerivation orderports.OrderDerivationService `optional:"true"`
	Publisher       agentports.AgentEventPublisher    `optional:"true"`
	// Watchtower puts an open failure on the feed a person reads.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type EDIServiceSetter interface {
	SetEDIService(service ediports.EDIService)
}

type service struct {
	l              *zap.Logger
	repo           servicefailureports.ServiceFailureRepository
	reasonCodeRepo servicefailureports.ServiceFailureReasonCodeRepository
	shipmentRepo   shipmentports.ShipmentRepository
	dispatchRepo   dispatchcontrolports.DispatchControlRepository
	commentService shipmentports.ShipmentCommentService
	auditService   auditports.AuditService
	realtime       agentports.RealtimeService
	ediService     ediports.EDIService
	delayedMarker  delayedShipmentMarker
	publisher      agentports.AgentEventPublisher
	watchtower     watchtowerports.WatchtowerProjector
}

func New(p Params) *service {
	s := &service{
		l:              p.Logger.Named("service.service-failure"),
		repo:           p.Repo,
		reasonCodeRepo: p.ReasonCodeRepo,
		shipmentRepo:   p.ShipmentRepo,
		dispatchRepo:   p.DispatchRepo,
		commentService: p.CommentService,
		auditService:   p.AuditService,
		realtime:       p.Realtime,
		publisher:      p.Publisher,
		watchtower:     p.Watchtower,
	}
	s.delayedMarker = newDelayedShipmentMarker(delayedShipmentMarkerParams{
		logger:          s.l,
		shipmentRepo:    s.shipmentRepo,
		auditService:    s.auditService,
		realtime:        s.realtime,
		orderDerivation: p.OrderDerivation,
	})
	return s
}

func (s *service) SetEDIService(service ediports.EDIService) {
	s.ediService = service
}

func (s *service) List(
	ctx context.Context,
	req *servicefailureports.ListServiceFailuresRequest,
) (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Service failure list request is required",
		)
	}
	return s.repo.List(ctx, req)
}

func (s *service) ListConnection(
	ctx context.Context,
	req *servicefailureports.ListServiceFailureConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *service) GetByID(
	ctx context.Context,
	req *servicefailureports.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.GetByID(ctx, req)
}

func (s *service) GetByShipment(
	ctx context.Context,
	req *servicefailureports.GetServiceFailureByShipmentRequest,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.GetByShipment(ctx, req)
}

func (s *service) CountUnresolved(
	ctx context.Context,
	req *servicefailureports.CountUnresolvedServiceFailuresRequest,
) (int, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return 0, multiErr
	}
	return s.repo.CountUnresolved(ctx, req)
}
