package shipmentinvalidation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const Resource = "shipments"

type Params struct {
	fx.In

	Realtime services.RealtimeService
	Epochs   repositories.ShipmentBoardEpochBumper
	Logger   *zap.Logger
}

type Service struct {
	realtime services.RealtimeService
	epochs   repositories.ShipmentBoardEpochBumper
	l        *zap.Logger
}

var _ services.ShipmentInvalidator = (*Service)(nil)

func New(p Params) services.ShipmentInvalidator {
	return NewWithDependencies(p.Realtime, p.Epochs, p.Logger)
}

func NewWithDependencies(
	realtime services.RealtimeService,
	epochs repositories.ShipmentBoardEpochBumper,
	logger *zap.Logger,
) *Service {
	return &Service{
		realtime: realtime,
		epochs:   epochs,
		l:        logger.Named("service.shipment-invalidation"),
	}
}

func (s *Service) InvalidateShipments(
	ctx context.Context,
	req *services.ShipmentInvalidation,
) {
	if req == nil || req.OrganizationID.IsNil() || req.BusinessUnitID.IsNil() {
		s.l.Warn("dropping shipment invalidation without a tenant")
		return
	}

	if s.epochs != nil {
		if err := s.epochs.BumpEpoch(ctx, req.OrganizationID, req.BusinessUnitID); err != nil {
			s.l.Warn(
				"failed to bump shipment board epoch",
				zap.String("organizationId", req.OrganizationID.String()),
				zap.Error(err),
			)
		}
	}

	if s.realtime == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		ActorUserID:    req.ActorUserID,
		ActorType:      req.ActorType,
		ActorID:        req.ActorID,
		ActorAPIKeyID:  req.ActorAPIKeyID,
		Resource:       Resource,
		Action:         req.Action,
		RecordID:       req.RecordID,
		Entity:         req.Entity,
	}); err != nil {
		s.l.Warn(
			"failed to publish shipment invalidation",
			zap.String("action", req.Action),
			zap.Error(err),
		)
	}
}
