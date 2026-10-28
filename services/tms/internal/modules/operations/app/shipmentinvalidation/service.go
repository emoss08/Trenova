package shipmentinvalidation

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const Resource = "shipments"

type Params struct {
	fx.In

	Realtime realtimeports.RealtimeService
	Epochs   shipmentboardports.ShipmentBoardEpochBumper
	Logger   *zap.Logger
}

type Service struct {
	realtime realtimeports.RealtimeService
	epochs   shipmentboardports.ShipmentBoardEpochBumper
	l        *zap.Logger
}

var _ shipmentinvalidationports.ShipmentInvalidator = (*Service)(nil)

func New(p Params) shipmentinvalidationports.ShipmentInvalidator {
	return NewWithDependencies(p.Realtime, p.Epochs, p.Logger)
}

func NewWithDependencies(
	realtime realtimeports.RealtimeService,
	epochs shipmentboardports.ShipmentBoardEpochBumper,
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
	req *shipmentinvalidationports.ShipmentInvalidation,
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
