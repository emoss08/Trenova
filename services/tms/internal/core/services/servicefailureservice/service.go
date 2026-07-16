package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.ServiceFailureRepository
	ReasonCodeRepo repositories.ServiceFailureReasonCodeRepository
	ShipmentRepo   repositories.ShipmentRepository
	DispatchRepo   repositories.DispatchControlRepository
	CommentService services.ShipmentCommentService
	AuditService   services.AuditService
	Realtime       services.RealtimeService
	OrderDerivation services.OrderDerivationService `optional:"true"`
}

type EDIServiceSetter interface {
	SetEDIService(service services.EDIService)
}

type service struct {
	l              *zap.Logger
	repo           repositories.ServiceFailureRepository
	reasonCodeRepo repositories.ServiceFailureReasonCodeRepository
	shipmentRepo   repositories.ShipmentRepository
	dispatchRepo   repositories.DispatchControlRepository
	commentService services.ShipmentCommentService
	auditService   services.AuditService
	realtime       services.RealtimeService
	ediService     services.EDIService
	delayedMarker  delayedShipmentMarker
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

func (s *service) SetEDIService(service services.EDIService) {
	s.ediService = service
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListServiceFailuresRequest,
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
	req *repositories.ListServiceFailureConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *service) GetByID(
	ctx context.Context,
	req *repositories.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.GetByID(ctx, req)
}

func (s *service) GetByShipment(
	ctx context.Context,
	req *repositories.GetServiceFailureByShipmentRequest,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.GetByShipment(ctx, req)
}
