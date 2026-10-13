package trackingeventservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func validateObserved(
	params *trackingeventports.RecordObservedStopEventParams,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if params == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	addTarget(multiErr, params.TenantInfo, params.MoveID, params.StopID)
	if params.Source.Interactive() || !params.Source.IsValid() {
		multiErr.Add("source", errortypes.ErrInvalid, "Source must be Carrier, Telematics or EDI")
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

func validateReported(
	params *trackingeventports.RecordReportedStopEventParams,
) *errortypes.MultiError {
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
	if params.DepartedAt != nil {
		switch {
		case params.Action != shipmentports.StopActualActionArrive:
			multiErr.Add(
				"departedAt",
				errortypes.ErrInvalid,
				"A departure time goes with an arrival",
			)
		case params.OccurredAt == nil:
			multiErr.Add(
				"occurredAt",
				errortypes.ErrRequired,
				"An arrival time is required with a departure time",
			)
		default:
			addEventTime(multiErr, "departedAt", params.DepartedAt)
		}
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
	case *at > timeutils.NowUnix()+shipmentports.StopActualClockSkewSeconds:
		multiErr.Add(field, errortypes.ErrInvalid, "Occurred at cannot be in the future")
	}
}
