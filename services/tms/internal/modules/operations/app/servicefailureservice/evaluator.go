package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type evaluateShipmentParams struct {
	source     *shipment.Shipment
	control    *dispatchcontrol.DispatchControl
	force      bool
	actor      *permissionports.RequestActor
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

func newServiceFailureEvaluationResult() *servicefailureports.ServiceFailureEvaluationResult {
	return &servicefailureports.ServiceFailureEvaluationResult{
		CreatedIDs:   make([]pulid.ID, 0),
		UpdatedIDs:   make([]pulid.ID, 0),
		CreatedStops: make([]servicefailureports.ServiceFailureEvaluatedStopSummary, 0),
		UpdatedStops: make([]servicefailureports.ServiceFailureEvaluatedStopSummary, 0),
		SkippedStops: make([]servicefailureports.ServiceFailureSkippedStop, 0),
	}
}

func (s *service) EvaluateShipment(
	ctx context.Context,
	req *servicefailureports.EvaluateShipmentServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, control, err := s.evaluationSource(ctx, req.TenantInfo, req.ShipmentID)
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
	req *servicefailureports.EvaluateStopServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	source, control, err := s.evaluationSource(ctx, req.TenantInfo, req.ShipmentID)
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
	req *servicefailureports.BulkEvaluateServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	result := newServiceFailureEvaluationResult()
	for _, shipmentID := range req.ShipmentIDs {
		if shipmentID.IsNil() {
			addSkippedEvaluation(addSkippedEvaluationParams{
				result: result,
				reason: "missing shipment ID",
			})
			continue
		}
		current, err := s.EvaluateShipment(
			ctx,
			&servicefailureports.EvaluateShipmentServiceFailuresRequest{
				TenantInfo: req.TenantInfo,
				ShipmentID: shipmentID,
				Force:      req.Force,
			},
			actor,
		)
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
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	plan, err := s.planDetection(ctx, params)
	if err != nil {
		return nil, err
	}

	result := newServiceFailureEvaluationResult()
	result.SkippedStops = append(result.SkippedStops, plan.SkippedStops...)
	result.Skipped = len(plan.SkippedStops)
	for _, detected := range plan.Detected {
		failure, pErr := s.persistDetected(ctx, detected, params.actor)
		if pErr != nil {
			return nil, pErr
		}
		if detected.Existing != nil {
			result.UpdatedIDs = append(result.UpdatedIDs, failure.ID)
			result.UpdatedStops = append(result.UpdatedStops, serviceFailureStopSummary(failure))
			continue
		}
		result.CreatedIDs = append(result.CreatedIDs, failure.ID)
		result.CreatedStops = append(result.CreatedStops, serviceFailureStopSummary(failure))
		s.transitionShipmentToDelayed(ctx, params.source, params.actor, failure)
	}

	return result, nil
}

func (s *service) qualifyingFailure(params qualifyingFailureParams) (*detectedAction, string) {
	if params.stop.IsCanceled() {
		return nil, "stop canceled"
	}
	if params.stop.ActualArrival == nil || *params.stop.ActualArrival <= 0 {
		return nil, "missing actual arrival"
	}
	cutoff := params.stop.EffectiveScheduledCutoff()
	if cutoff <= 0 {
		return nil, "missing scheduled cutoff"
	}
	if params.stop.CountLateOverride != nil && !*params.stop.CountLateOverride {
		return nil, "count late override disabled"
	}
	if !params.force {
		if !shouldEvaluateStop(shouldEvaluateStopParams{
			source:      params.source,
			stop:        params.stop,
			shipperStop: params.shipperStop,
			policy:      params.control.RecordServiceFailures,
		}) {
			return nil, "policy skipped"
		}
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
	entity.Normalize()
	return &detectedAction{entity: entity}, ""
}

func (s *service) createOrUpdateDetected(
	ctx context.Context,
	action *detectedAction,
	actor *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	detected, err := s.planDetected(ctx, action.entity)
	if err != nil {
		return nil, err
	}
	action.existing = detected.Existing != nil

	return s.persistDetected(ctx, detected, actor)
}

func mergeEvaluationResult(target, source *servicefailureports.ServiceFailureEvaluationResult) {
	if target == nil || source == nil {
		return
	}
	target.CreatedIDs = append(target.CreatedIDs, source.CreatedIDs...)
	target.UpdatedIDs = append(target.UpdatedIDs, source.UpdatedIDs...)
	target.CreatedStops = append(target.CreatedStops, source.CreatedStops...)
	target.UpdatedStops = append(target.UpdatedStops, source.UpdatedStops...)
	target.SkippedStops = append(target.SkippedStops, source.SkippedStops...)
	target.Skipped += source.Skipped
}

type addSkippedEvaluationParams struct {
	result             *servicefailureports.ServiceFailureEvaluationResult
	shipmentID         pulid.ID
	shipmentMoveID     pulid.ID
	stop               *shipment.Stop
	gracePeriodMinutes int
	reason             string
}

func addSkippedEvaluation(params addSkippedEvaluationParams) {
	if params.result == nil {
		return
	}
	params.result.Skipped++
	params.result.SkippedStops = append(params.result.SkippedStops, skippedStopDetail(params))
}

func skippedStopDetail(
	params addSkippedEvaluationParams,
) servicefailureports.ServiceFailureSkippedStop {
	detail := stopSummaryFromStop(
		params.shipmentID,
		params.shipmentMoveID,
		params.stop,
		params.gracePeriodMinutes,
	)
	detail.Reason = params.reason

	return detail
}

func serviceFailureStopSummary(
	failure *servicefailure.ServiceFailure,
) servicefailureports.ServiceFailureEvaluatedStopSummary {
	if failure == nil {
		return servicefailureports.ServiceFailureEvaluatedStopSummary{}
	}

	detail := servicefailureports.ServiceFailureEvaluatedStopSummary{
		ShipmentID:         failure.ShipmentID,
		ShipmentMoveID:     failure.ShipmentMoveID,
		StopID:             failure.StopID,
		StopType:           failure.StopType,
		ScheduledCutoff:    failure.ScheduledCutoff,
		ActualArrival:      failure.ActualArrival,
		GracePeriodMinutes: failure.GracePeriodMinutes,
		LateMinutes:        failure.LateMinutes,
		ServiceFailureID:   failure.ID,
	}
	applyStopContext(&detail, failure.Stop)
	return detail
}

func stopSummaryFromStop(
	shipmentID pulid.ID,
	shipmentMoveID pulid.ID,
	stop *shipment.Stop,
	gracePeriodMinutes int,
) servicefailureports.ServiceFailureEvaluatedStopSummary {
	detail := servicefailureports.ServiceFailureEvaluatedStopSummary{
		ShipmentID:         shipmentID,
		ShipmentMoveID:     shipmentMoveID,
		GracePeriodMinutes: gracePeriodMinutes,
	}
	if stop == nil {
		return detail
	}

	detail.StopID = stop.ID
	detail.StopSequence = stop.Sequence
	detail.StopType = stop.Type
	detail.ScheduledCutoff = stop.EffectiveScheduledCutoff()
	if stop.ShipmentMoveID.IsNotNil() {
		detail.ShipmentMoveID = stop.ShipmentMoveID
	}
	if stop.ActualArrival != nil {
		detail.ActualArrival = *stop.ActualArrival
		if detail.ScheduledCutoff > 0 {
			detail.LateMinutes = lateMinutesAfterGrace(
				*stop.ActualArrival,
				detail.ScheduledCutoff,
				gracePeriodMinutes,
			)
		}
	}

	applyStopContext(&detail, stop)
	return detail
}

func applyStopContext(
	detail *servicefailureports.ServiceFailureEvaluatedStopSummary,
	stop *shipment.Stop,
) {
	if detail == nil || stop == nil {
		return
	}
	detail.StopSequence = stop.Sequence
	if stop.Type != "" {
		detail.StopType = stop.Type
	}
	if stop.LocationID.IsNotNil() {
		detail.LocationID = stop.LocationID
	}
	if stop.Location == nil {
		return
	}

	detail.LocationID = stop.Location.ID
	detail.LocationName = stop.Location.Name
	detail.LocationCode = stop.Location.Code
	detail.City = stop.Location.City
	if stop.Location.State != nil {
		detail.StateCode = stop.Location.State.Abbreviation
	}
}
