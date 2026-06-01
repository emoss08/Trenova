package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) CreateManual(
	ctx context.Context,
	req *services.CreateManualServiceFailureRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, move, stop, err := s.shipmentStopContext(ctx, shipmentStopContextParams{
		tenantInfo: req.TenantInfo,
		shipmentID: req.ShipmentID,
		moveID:     req.ShipmentMoveID,
		stopID:     req.StopID,
	})
	if err != nil {
		return nil, err
	}
	if stop.IsCanceled() || move.IsCanceled() || source.Status == shipment.StatusCanceled {
		return nil, errortypes.NewValidationError(
			"stopId",
			errortypes.ErrInvalidOperation,
			"Canceled shipments, moves, and stops cannot receive service failures",
		)
	}
	if req.Type != servicefailure.TypeForStop(stop) {
		return nil, errortypes.NewValidationError(
			"type",
			errortypes.ErrInvalid,
			"Service failure type must match the stop type",
		)
	}

	reason, err := s.activeReasonCode(ctx, activeReasonCodeParams{
		reasonCodeID: req.ReasonCodeID,
		tenantInfo:   req.TenantInfo,
		stop:         stop,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.dispatchControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	entity, err := s.manualEntity(manualEntityParams{
		req:          req,
		source:       source,
		move:         move,
		stop:         stop,
		reason:       reason,
		defaultGrace: normalizedGracePeriod(control),
	})
	if err != nil {
		return nil, err
	}
	entity.CreatedByID = pulid.PtrOrNil(actor.UserIDOrNil())

	if multiErr := validateServiceFailure(entity); multiErr != nil {
		return nil, multiErr
	}

	if _, err = s.repo.FindUnresolvedByStop(ctx, activeStopRequest(entity)); err == nil {
		return nil, errortypes.NewBusinessError("An unresolved service failure already exists for this stop").
			WithParam("shipmentId", entity.ShipmentID.String()).
			WithParam("stopId", entity.StopID.String())
	} else if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.afterServiceFailureCreate(ctx, created, actor, "Manual service failure created")
	s.comment(ctx, commentParams{
		entity:  created,
		comment: "Manual service failure recorded",
		metadata: map[string]any{
			"serviceFailureId": created.ID.String(),
			"source":           string(created.Source),
		},
	})
	return created, nil
}
