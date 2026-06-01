package servicefailureservice

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type activeReasonCodeParams struct {
	reasonCodeID pulid.ID
	tenantInfo   pagination.TenantInfo
	stop         *shipment.Stop
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
