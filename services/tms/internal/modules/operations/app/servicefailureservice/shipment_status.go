package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
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
	logger          *zap.Logger
	shipmentRepo    shipmentports.ShipmentRepository
	auditService    auditports.AuditService
	realtime        realtimeports.RealtimeService
	orderDerivation orderports.OrderDerivationService
}

type delayedShipmentMarkParams struct {
	shipment *shipment.Shipment
	actor    *permissionports.RequestActor
	failure  *servicefailure.ServiceFailure
}

type shipmentStatusMarker struct {
	l               *zap.Logger
	shipmentRepo    shipmentports.ShipmentRepository
	auditService    auditports.AuditService
	realtime        realtimeports.RealtimeService
	orderDerivation orderports.OrderDerivationService
}

func newDelayedShipmentMarker(params delayedShipmentMarkerParams) delayedShipmentMarker {
	return &shipmentStatusMarker{
		l:               params.logger,
		shipmentRepo:    params.shipmentRepo,
		auditService:    params.auditService,
		realtime:        params.realtime,
		orderDerivation: params.orderDerivation,
	}
}

func (s *service) transitionShipmentToDelayed(
	ctx context.Context,
	source *shipment.Shipment,
	actor *permissionports.RequestActor,
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
	if !canMarkDelayed(params.shipment) {
		return nil, nil
	}

	previous := *params.shipment
	candidate := *params.shipment
	candidate.Status = shipmentvocab.StatusDelayed
	updated, err := m.shipmentRepo.UpdateDerivedState(ctx, &candidate)
	if err != nil {
		return nil, err
	}

	if m.orderDerivation != nil && !updated.OrderID.IsNil() {
		if derivErr := m.orderDerivation.RecomputeOrder(
			ctx,
			pagination.TenantInfo{
				OrgID: updated.OrganizationID,
				BuID:  updated.BusinessUnitID,
			},
			updated.OrderID,
		); derivErr != nil {
			m.l.Warn(
				"failed to recompute order after service-failure delay",
				zap.Error(derivErr),
				zap.String("orderId", updated.OrderID.String()),
			)
		}
	}

	auditActor := params.actor.AuditActorOrSystem()
	if err = m.auditService.LogAction(&auditports.LogActionParams{
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
		m.l.Warn(
			"failed to publish delayed shipment invalidation for service failure",
			zap.Error(err),
		)
	}

	return updated, nil
}
