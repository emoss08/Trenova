package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *service) PreviewEvaluateShipment(
	ctx context.Context,
	req *services.EvaluateShipmentServiceFailuresRequest,
) (*services.ServiceFailureDetectionPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, control, err := s.evaluationSource(ctx, req.TenantInfo, req.ShipmentID)
	if err != nil {
		return nil, err
	}

	return s.planDetection(ctx, evaluateShipmentParams{
		source:  source,
		control: control,
		force:   req.Force,
	})
}

func (s *service) evaluationSource(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, *dispatchcontrol.DispatchControl, error) {
	source, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	control, err := s.dispatchControl(ctx, tenantInfo)
	if err != nil {
		return nil, nil, err
	}

	return source, control, nil
}

func (s *service) planDetection(
	ctx context.Context,
	params evaluateShipmentParams,
) (*services.ServiceFailureDetectionPlan, error) {
	plan := &services.ServiceFailureDetectionPlan{
		Shipment:     params.source,
		Detected:     make([]services.DetectedServiceFailure, 0),
		SkippedStops: make([]services.ServiceFailureSkippedStop, 0),
	}
	if params.source == nil {
		return plan, nil
	}
	if params.source.Status == shipment.StatusCanceled {
		plan.SkippedStops = append(plan.SkippedStops, skippedStopDetail(addSkippedEvaluationParams{
			shipmentID: params.source.ID,
			reason:     "shipment canceled",
		}))
		return plan, nil
	}

	shipperStop := params.source.ShipperStop()
	gracePeriod := normalizedGracePeriod(params.control)
	for _, move := range params.source.Moves {
		if move == nil || move.IsCanceled() {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if params.onlyStopID != nil && stop.ID != *params.onlyStopID {
				continue
			}
			action, reason := s.qualifyingFailure(qualifyingFailureParams{
				source:      params.source,
				move:        move,
				stop:        stop,
				shipperStop: shipperStop,
				control:     params.control,
				gracePeriod: gracePeriod,
				force:       params.force,
			})
			if action == nil {
				plan.SkippedStops = append(
					plan.SkippedStops,
					skippedStopDetail(addSkippedEvaluationParams{
						shipmentID:         params.source.ID,
						shipmentMoveID:     move.ID,
						stop:               stop,
						gracePeriodMinutes: gracePeriod,
						reason:             reason,
					}),
				)
				s.l.Debug(
					"service failure stop skipped",
					zap.String("stopID", stop.ID.String()),
					zap.String("reason", reason),
				)
				continue
			}
			detected, err := s.planDetected(ctx, action.entity)
			if err != nil {
				return nil, err
			}
			if detected.Existing == nil && !plan.MarksDelayed {
				plan.MarksDelayed = s.delayedMarker != nil && canMarkDelayed(params.source)
			}
			plan.Detected = append(plan.Detected, detected)
		}
	}

	return plan, nil
}

func (s *service) planDetected(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (services.DetectedServiceFailure, error) {
	tenantInfo := serviceFailureTenantInfo(entity)
	defaultReason, err := s.defaultReasonCode(ctx, tenantInfo, entity.StopType)
	if err != nil {
		return services.DetectedServiceFailure{}, err
	}
	if defaultReason != nil {
		entity.ReasonCodeID = pulid.PtrOrNil(defaultReason.ID)
		entity.Notes = defaultReason.DefaultNote
	}
	if entity.Notes == "" {
		entity.Notes = detectedFailureNote(entity)
	}

	existing, err := s.repo.FindUnresolvedByStop(ctx, activeStopRequest(entity))
	switch {
	case err == nil:
		updated := *existing
		updated.ScheduledCutoff = entity.ScheduledCutoff
		updated.ActualArrival = entity.ActualArrival
		updated.GracePeriodMinutes = entity.GracePeriodMinutes
		updated.LateMinutes = entity.LateMinutes
		if updated.ReasonCodeID == nil {
			updated.ReasonCodeID = entity.ReasonCodeID
			updated.Notes = entity.Notes
		}
		return services.DetectedServiceFailure{Failure: &updated, Existing: existing}, nil
	case errortypes.IsNotFoundError(err):
	default:
		return services.DetectedServiceFailure{}, err
	}

	if multiErr := validateServiceFailure(entity); multiErr != nil {
		return services.DetectedServiceFailure{}, multiErr
	}

	return services.DetectedServiceFailure{Failure: entity}, nil
}

func (s *service) persistDetected(
	ctx context.Context,
	detected services.DetectedServiceFailure,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if detected.Existing != nil {
		return s.repo.UpdateDetectionSnapshot(ctx, detected.Failure)
	}

	created, err := s.repo.Create(ctx, detected.Failure)
	if err != nil {
		return nil, err
	}
	s.afterServiceFailureCreate(ctx, created, actor, "Service failure detected")
	s.comment(ctx, commentParams{
		entity:   created,
		comment:  "Service failure detected",
		metadata: serviceFailureMetadata(created, actor),
	})

	return created, nil
}

func canMarkDelayed(entity *shipment.Shipment) bool {
	return entity != nil &&
		shipmentstate.IsDelayedEligibleShipmentStatus(entity.Status) &&
		shipmentstate.CanTransitionShipmentStatus(entity.Status, shipment.StatusDelayed)
}
