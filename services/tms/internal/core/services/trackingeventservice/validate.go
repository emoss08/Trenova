package trackingeventservice

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func validateObserved(params *services.RecordObservedStopEventParams) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if params == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	addTarget(multiErr, params.TenantInfo, params.MoveID, params.StopID)
	if params.Source.Interactive() || !params.Source.IsValid() {
		multiErr.Add("source", errortypes.ErrInvalid, "Source must be Telematics or EDI")
	}
	if params.Kind != shipment.VisitArrival && params.Kind != shipment.VisitDeparture {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind must be Arrival or Departure")
	}
	addEventTime(multiErr, "eventAt", &params.EventAt)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func validateReported(params *services.RecordReportedStopEventParams) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if params == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	addTarget(multiErr, params.TenantInfo, params.MoveID, params.StopID)
	if !params.Source.Interactive() {
		multiErr.Add("source", errortypes.ErrInvalid, "Source must be Dispatcher, Agent or Driver")
	}
	if !params.Action.IsValid() {
		multiErr.Add("action", errortypes.ErrInvalid, "Action must be Arrive or Depart")
	}
	if params.OccurredAt != nil {
		addEventTime(multiErr, "occurredAt", params.OccurredAt)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func addTarget(
	multiErr *errortypes.MultiError,
	tenantInfo pagination.TenantInfo,
	moveID, stopID pulid.ID,
) {
	if tenantInfo.OrgID.IsNil() {
		multiErr.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if tenantInfo.BuID.IsNil() {
		multiErr.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if moveID.IsNil() {
		multiErr.Add("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	if stopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")
	}
}

func addEventTime(multiErr *errortypes.MultiError, field string, at *int64) {
	switch {
	case *at <= 0:
		multiErr.Add(field, errortypes.ErrInvalid, "Occurred at must be a valid timestamp")
	case *at > timeutils.NowUnix()+repositories.StopActualClockSkewSeconds:
		multiErr.Add(field, errortypes.ErrInvalid, "Occurred at cannot be in the future")
	}
}
