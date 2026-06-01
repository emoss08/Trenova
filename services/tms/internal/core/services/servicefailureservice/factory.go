package servicefailureservice

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type shipmentStopContextParams struct {
	tenantInfo pagination.TenantInfo
	shipmentID pulid.ID
	moveID     pulid.ID
	stopID     pulid.ID
}

type activeReasonCodeParams struct {
	reasonCodeID pulid.ID
	tenantInfo   pagination.TenantInfo
	stop         *shipment.Stop
}

type manualEntityParams struct {
	req          *services.CreateManualServiceFailureRequest
	source       *shipment.Shipment
	move         *shipment.ShipmentMove
	stop         *shipment.Stop
	reason       *servicefailure.ReasonCode
	defaultGrace int
}

func (s *service) shipmentStopContext(
	ctx context.Context,
	params shipmentStopContextParams,
) (*shipment.Shipment, *shipment.ShipmentMove, *shipment.Stop, error) {
	source, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         params.shipmentID,
		TenantInfo: params.tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	for _, move := range source.Moves {
		if move == nil || (params.moveID.IsNotNil() && move.ID != params.moveID) {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil && stop.ID == params.stopID {
				return source, move, stop, nil
			}
		}
	}

	return nil, nil, nil, errortypes.NewNotFoundError("Stop not found within your organization")
}

func (s *service) activeReasonCode(
	ctx context.Context,
	params activeReasonCodeParams,
) (*servicefailure.ReasonCode, error) {
	reason, err := s.reasonCodeRepo.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         params.reasonCodeID,
		TenantInfo: params.tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !reason.Active {
		return nil, errortypes.NewValidationError(
			"reasonCodeId",
			errortypes.ErrInvalid,
			"Service failure reason code must be active",
		)
	}
	if params.stop != nil && !reason.AppliesTo.AllowsStopType(params.stop.Type) {
		return nil, errortypes.NewValidationError(
			"reasonCodeId",
			errortypes.ErrInvalid,
			"Service failure reason code does not apply to this stop type",
		)
	}
	return reason, nil
}

func (s *service) defaultReasonCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	stopType shipment.StopType,
) (*servicefailure.ReasonCode, error) {
	reason, err := s.reasonCodeRepo.FindDefault(ctx, tenantInfo, servicefailure.AppliesToForStop(&shipment.Stop{Type: stopType}))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return reason, nil
}

func (s *service) dispatchControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*dispatchcontrol.DispatchControl, error) {
	control, err := s.dispatchRepo.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return nil, err
	}
	return control, nil
}

func (s *service) manualEntity(params manualEntityParams) (*servicefailure.ServiceFailure, error) {
	cutoff := params.stop.EffectiveScheduledCutoff()
	if params.req.ScheduledCutoff != nil {
		cutoff = *params.req.ScheduledCutoff
	}
	if cutoff <= 0 {
		return nil, errortypes.NewValidationError(
			"scheduledCutoff",
			errortypes.ErrRequired,
			"Scheduled cutoff is required",
		)
	}

	actual := timeutils.NowUnix()
	if params.stop.ActualArrival != nil && *params.stop.ActualArrival > 0 {
		actual = *params.stop.ActualArrival
	}
	if params.req.ActualArrival != nil {
		actual = *params.req.ActualArrival
	}

	grace := params.defaultGrace
	if params.req.GracePeriodMinutes != nil {
		grace = *params.req.GracePeriodMinutes
	}
	lateMinutes := lateMinutesAfterGrace(actual, cutoff, grace)
	if params.req.LateMinutes != nil {
		lateMinutes = *params.req.LateMinutes
	}
	if lateMinutes < 1 {
		lateMinutes = 1
	}

	entity := &servicefailure.ServiceFailure{
		ShipmentID:            params.source.ID,
		ShipmentMoveID:        params.move.ID,
		StopID:                params.stop.ID,
		OrganizationID:        params.source.OrganizationID,
		BusinessUnitID:        params.source.BusinessUnitID,
		ReasonCodeID:          pulid.PtrOrNil(params.reason.ID),
		Type:                  params.req.Type,
		Source:                servicefailure.SourceManual,
		Status:                servicefailure.StatusOpen,
		StopType:              params.stop.Type,
		ScheduledCutoff:       cutoff,
		ActualArrival:         actual,
		GracePeriodMinutes:    grace,
		LateMinutes:           lateMinutes,
		Notes:                 strings.TrimSpace(params.req.Notes),
		InternalNotes:         strings.TrimSpace(params.req.InternalNotes),
		X12StatusCodeOverride: strings.TrimSpace(params.req.X12StatusCodeOverride),
		X12ReasonCodeOverride: strings.TrimSpace(params.req.X12ReasonCodeOverride),
		X12ExceptionCode:      strings.TrimSpace(params.req.X12ExceptionCode),
		DetectedAt:            timeutils.NowUnix(),
	}
	if entity.Notes == "" {
		entity.Notes = params.reason.DefaultNote
	}
	if entity.Notes == "" {
		entity.Notes = fmt.Sprintf("%s recorded manually", params.reason.Label)
	}
	return entity, nil
}

func validateServiceFailure(entity *servicefailure.ServiceFailure) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func activeStopRequest(entity *servicefailure.ServiceFailure) *repositories.ServiceFailureActiveStopRequest {
	return &repositories.ServiceFailureActiveStopRequest{
		TenantInfo:     serviceFailureTenantInfo(entity),
		ShipmentID:     entity.ShipmentID,
		ShipmentMoveID: entity.ShipmentMoveID,
		StopID:         entity.StopID,
		Type:           entity.Type,
	}
}

func normalizedGracePeriod(control *dispatchcontrol.DispatchControl) int {
	if control == nil || control.ServiceFailureGracePeriod == nil || *control.ServiceFailureGracePeriod <= 0 {
		return dispatchcontrol.DefaultServiceFailureGracePeriod
	}
	return *control.ServiceFailureGracePeriod
}

func lateMinutesAfterGrace(actualArrival int64, cutoff int64, graceMinutes int) int64 {
	lateSeconds := actualArrival - (cutoff + int64(graceMinutes)*60)
	if lateSeconds <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(lateSeconds) / 60))
}

func serviceFailureTenantInfo(entity *servicefailure.ServiceFailure) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

func detectedFailureNote(entity *servicefailure.ServiceFailure) string {
	label := "late delivery"
	if entity.Type == servicefailure.TypeLatePickup {
		label = "late pickup"
	}
	return fmt.Sprintf("Detected %s service failure %d minute(s) after grace.", label, entity.LateMinutes)
}

func optionalIDString(id *pulid.ID) string {
	if id == nil || id.IsNil() {
		return ""
	}
	return id.String()
}
