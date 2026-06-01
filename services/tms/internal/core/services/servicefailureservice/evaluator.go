package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type evaluateShipmentParams struct {
	source     *shipment.Shipment
	control    *dispatchcontrol.DispatchControl
	force      bool
	actor      *services.RequestActor
	onlyStopID *pulid.ID
}

type detectedAction struct {
	entity   *servicefailure.ServiceFailure
	existing bool
}

type qualifyingFailureParams struct {
	source      *shipment.Shipment
	move        *shipment.ShipmentMove
	stop        *shipment.Stop
	shipperStop *shipment.Stop
	control     *dispatchcontrol.DispatchControl
	gracePeriod int
	force       bool
}

func (s *service) EvaluateShipment(
	ctx context.Context,
	req *services.EvaluateShipmentServiceFailuresRequest,
	actor *services.RequestActor,
) (*services.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	control, err := s.dispatchControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return s.evaluateShipment(ctx, evaluateShipmentParams{
		source:  source,
		control: control,
		force:   req.Force,
		actor:   actor,
	})
}

func (s *service) EvaluateStop(
	ctx context.Context,
	req *services.EvaluateStopServiceFailuresRequest,
	actor *services.RequestActor,
) (*services.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	control, err := s.dispatchControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	stopID := req.StopID
	return s.evaluateShipment(ctx, evaluateShipmentParams{
		source:     source,
		control:    control,
		force:      req.Force,
		actor:      actor,
		onlyStopID: &stopID,
	})
}

func (s *service) BulkEvaluate(
	ctx context.Context,
	req *services.BulkEvaluateServiceFailuresRequest,
	actor *services.RequestActor,
) (*services.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	result := &services.ServiceFailureEvaluationResult{}
	for _, shipmentID := range req.ShipmentIDs {
		if shipmentID.IsNil() {
			result.Skipped++
			continue
		}
		current, err := s.EvaluateShipment(ctx, &services.EvaluateShipmentServiceFailuresRequest{
			TenantInfo: req.TenantInfo,
			ShipmentID: shipmentID,
			Force:      req.Force,
		}, actor)
		if err != nil {
			return nil, err
		}
		mergeEvaluationResult(result, current)
	}

	return result, nil
}

func (s *service) evaluateShipment(
	ctx context.Context,
	params evaluateShipmentParams,
) (*services.ServiceFailureEvaluationResult, error) {
	result := &services.ServiceFailureEvaluationResult{}
	if params.source == nil {
		return result, nil
	}
	if params.source.Status == shipment.StatusCanceled {
		result.Skipped++
		return result, nil
	}
	if !params.force && params.control.RecordServiceFailures == dispatchcontrol.ServiceIncidentTypeNever {
		result.Skipped++
		return result, nil
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
				result.Skipped++
				s.l.Debug("service failure stop skipped", zap.String("stopID", stop.ID.String()), zap.String("reason", reason))
				continue
			}
			failure, err := s.createOrUpdateDetected(ctx, action, params.actor)
			if err != nil {
				return nil, err
			}
			if action.existing {
				result.UpdatedIDs = append(result.UpdatedIDs, failure.ID)
				continue
			}
			result.CreatedIDs = append(result.CreatedIDs, failure.ID)
			s.transitionShipmentToDelayed(ctx, params.source, params.actor, failure)
		}
	}

	return result, nil
}

func (s *service) qualifyingFailure(params qualifyingFailureParams) (*detectedAction, string) {
	if params.stop.IsCanceled() {
		return nil, "stop canceled"
	}
	if !params.stop.IsCompleted() {
		return nil, "stop not completed"
	}
	if params.stop.ActualArrival == nil || *params.stop.ActualArrival <= 0 {
		return nil, "missing actual arrival"
	}
	cutoff := params.stop.EffectiveScheduledCutoff()
	if cutoff <= 0 {
		return nil, "missing scheduled cutoff"
	}
	if !params.force && !shouldEvaluateStop(shouldEvaluateStopParams{
		source:      params.source,
		stop:        params.stop,
		shipperStop: params.shipperStop,
		policy:      params.control.RecordServiceFailures,
	}) {
		return nil, "policy skipped"
	}

	graceSeconds := int64(params.gracePeriod) * 60
	if *params.stop.ActualArrival <= cutoff+graceSeconds {
		return nil, "not late after grace"
	}

	lateMinutes := lateMinutesAfterGrace(*params.stop.ActualArrival, cutoff, params.gracePeriod)
	entity := &servicefailure.ServiceFailure{
		ShipmentID:         params.source.ID,
		ShipmentMoveID:     params.move.ID,
		StopID:             params.stop.ID,
		OrganizationID:     params.source.OrganizationID,
		BusinessUnitID:     params.source.BusinessUnitID,
		Type:               servicefailure.TypeForStop(params.stop),
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusOpen,
		StopType:           params.stop.Type,
		ScheduledCutoff:    cutoff,
		ActualArrival:      *params.stop.ActualArrival,
		GracePeriodMinutes: params.gracePeriod,
		LateMinutes:        lateMinutes,
		DetectedAt:         timeutils.NowUnix(),
	}
	return &detectedAction{entity: entity}, ""
}

func (s *service) createOrUpdateDetected(
	ctx context.Context,
	action *detectedAction,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	entity := action.entity
	tenantInfo := serviceFailureTenantInfo(entity)
	defaultReason, err := s.defaultReasonCode(ctx, tenantInfo, entity.StopType)
	if err != nil {
		return nil, err
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
		action.existing = true
		updated := *existing
		updated.ScheduledCutoff = entity.ScheduledCutoff
		updated.ActualArrival = entity.ActualArrival
		updated.GracePeriodMinutes = entity.GracePeriodMinutes
		updated.LateMinutes = entity.LateMinutes
		if updated.ReasonCodeID == nil {
			updated.ReasonCodeID = entity.ReasonCodeID
			updated.Notes = entity.Notes
		}
		return s.repo.UpdateDetectionSnapshot(ctx, &updated)
	case errortypes.IsNotFoundError(err):
	default:
		return nil, err
	}

	if multiErr := validateServiceFailure(entity); multiErr != nil {
		return nil, multiErr
	}
	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.afterServiceFailureCreate(ctx, created, actor, "Service failure detected")
	s.comment(ctx, commentParams{
		entity:  created,
		comment: "Service failure detected",
		metadata: map[string]any{
			"serviceFailureId": created.ID.String(),
			"source":           string(created.Source),
			"lateMinutes":      created.LateMinutes,
		},
	})
	return created, nil
}

func mergeEvaluationResult(target, source *services.ServiceFailureEvaluationResult) {
	if target == nil || source == nil {
		return
	}
	target.CreatedIDs = append(target.CreatedIDs, source.CreatedIDs...)
	target.UpdatedIDs = append(target.UpdatedIDs, source.UpdatedIDs...)
	target.Skipped += source.Skipped
}
