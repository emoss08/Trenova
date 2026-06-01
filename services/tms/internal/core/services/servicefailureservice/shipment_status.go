package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

type delayedShipmentMarker interface {
	MarkDelayedForServiceFailure(
		ctx context.Context,
		params delayedShipmentMarkParams,
	) (*shipment.Shipment, error)
}

type delayedShipmentMarkerParams struct {
	logger       *zap.Logger
	shipmentRepo repositories.ShipmentRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

type delayedShipmentMarkParams struct {
	shipment *shipment.Shipment
	actor    *services.RequestActor
	failure  *servicefailure.ServiceFailure
}

type shipmentStatusMarker struct {
	l            *zap.Logger
	shipmentRepo repositories.ShipmentRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func newDelayedShipmentMarker(params delayedShipmentMarkerParams) delayedShipmentMarker {
	return &shipmentStatusMarker{
		l:            params.logger,
		shipmentRepo: params.shipmentRepo,
		auditService: params.auditService,
		realtime:     params.realtime,
	}
}

func (s *service) transitionShipmentToDelayed(
	ctx context.Context,
	source *shipment.Shipment,
	actor *services.RequestActor,
	failure *servicefailure.ServiceFailure,
) {
	if s.delayedMarker == nil {
		return
	}
	updated, err := s.delayedMarker.MarkDelayedForServiceFailure(ctx, delayedShipmentMarkParams{
		shipment: source,
		actor:    actor,
		failure:  failure,
	})
	if err != nil {
		fields := []zap.Field{zap.Error(err)}
		if source != nil {
			fields = append(fields, zap.String("shipmentID", source.ID.String()))
		}
		if failure != nil {
			fields = append(fields, zap.String("serviceFailureID", failure.ID.String()))
		}
		s.l.Warn("failed to transition shipment to delayed for service failure", fields...)
		return
	}
	if updated != nil {
		*source = *updated
	}
}

func (m *shipmentStatusMarker) MarkDelayedForServiceFailure(
	ctx context.Context,
	params delayedShipmentMarkParams,
) (*shipment.Shipment, error) {
	if params.shipment == nil ||
		!shipmentstate.IsDelayedEligibleShipmentStatus(params.shipment.Status) ||
		!shipmentstate.CanTransitionShipmentStatus(params.shipment.Status, shipment.StatusDelayed) {
		return nil, nil
	}

	previous := *params.shipment
	candidate := *params.shipment
	candidate.Status = shipment.StatusDelayed
	updated, err := m.shipmentRepo.UpdateDerivedState(ctx, &candidate)
	if err != nil {
		return nil, err
	}

	auditActor := params.actor.AuditActorOrSystem()
	if err = m.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		PreviousState:  jsonutils.MustToJSON(&previous),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Shipment marked delayed due to service failure"), auditservice.WithDiff(&previous, updated)); err != nil {
		m.l.Warn("failed to log delayed shipment audit for service failure", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, m.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       permission.ResourceShipment.String(),
		Action:         "delayed",
		RecordID:       updated.ID,
		Entity:         updated,
	}); err != nil {
		m.l.Warn("failed to publish delayed shipment invalidation for service failure", zap.Error(err))
	}

	return updated, nil
}
