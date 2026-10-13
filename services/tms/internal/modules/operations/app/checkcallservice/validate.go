package checkcallservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxClientEventIDLength  = 128
	maxEstimatedArrivalDays = int64(30)
	secondsPerHour          = int64(3600)
	secondsPerDay           = int64(24 * 3600)
)

type timePolicy int

const (
	refuseFutureTimes timePolicy = iota
	clampFutureTimes
)

type normalizeInput struct {
	details *checkcallports.CheckCallDetails
	now     int64
	policy  timePolicy
}

func normalizeDetails(in normalizeInput) (int64, *errortypes.MultiError) {
	multiErr := errortypes.NewMultiError()
	d := in.details

	if !d.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind must be Update, Arrival or Departure")
	}
	if d.Kind.IsStopVisit() && d.StopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Choose the stop that was reached or left")
	}

	reportedAt := in.now
	if d.ReportedAt != nil {
		reportedAt = *d.ReportedAt
	}
	switch {
	case reportedAt <= 0:
		multiErr.Add("reportedAt", errortypes.ErrInvalid, "Reported time must be a valid timestamp")
	case reportedAt > in.now+shipmentports.StopActualClockSkewSeconds:
		if in.policy == clampFutureTimes {
			reportedAt = in.now
		} else {
			multiErr.Add("reportedAt", errortypes.ErrInvalid, "Reported time cannot be in the future")
		}
	case reportedAt < in.now-checkcall.MaxReportBackdateHours*secondsPerHour:
		multiErr.Add(
			"reportedAt",
			errortypes.ErrInvalid,
			"Reported time cannot be more than 7 days ago",
		)
	}

	if d.EstimatedArrival != nil {
		eta := *d.EstimatedArrival
		if eta <= 0 || eta > in.now+maxEstimatedArrivalDays*secondsPerDay {
			multiErr.Add(
				"estimatedArrival",
				errortypes.ErrInvalid,
				"Estimated arrival must be within the next 30 days",
			)
		}
	}

	d.City = strings.TrimSpace(d.City)
	d.StateCode = strings.ToUpper(strings.TrimSpace(d.StateCode))
	d.DelayReason = strings.TrimSpace(d.DelayReason)
	d.Note = strings.TrimSpace(d.Note)
	d.ContactName = strings.TrimSpace(d.ContactName)
	if !d.Delayed {
		d.DelayReason = ""
	}

	if multiErr.HasErrors() {
		return 0, multiErr
	}
	return reportedAt, nil
}

func validateLogRequest(req *checkcallports.LogCheckCallRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	addTenant(multiErr, req.TenantInfo)
	if req.MoveID.IsNil() {
		multiErr.Add("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	if req.Source != checkcall.SourceDispatcher && req.Source != checkcall.SourceAgent {
		multiErr.Add("source", errortypes.ErrInvalid, "Source must be Dispatcher or Agent")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func validateClientEventID(id string) error {
	if id == "" || stringutils.IsCompactToken(id, maxClientEventIDLength) {
		return nil
	}
	return errortypes.NewValidationError(
		"clientEventId",
		errortypes.ErrInvalid,
		"Client event ID must be 128 characters or fewer with no spaces",
	)
}

func addTenant(multiErr *errortypes.MultiError, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() {
		multiErr.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if tenantInfo.BuID.IsNil() {
		multiErr.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
}

func findStop(move *shipment.ShipmentMove, stopID pulid.ID) *shipment.Stop {
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID && stop.Status != shipment.StopStatusCanceled {
			return stop
		}
	}
	return nil
}

func moveIsClosed(move *shipment.ShipmentMove) bool {
	return move.Status == shipment.MoveStatusCompleted || move.Status == shipment.MoveStatusCanceled
}

func visitKind(kind checkcall.Kind) shipment.VisitKind {
	if kind == checkcall.KindDeparture {
		return shipment.VisitDeparture
	}
	return shipment.VisitArrival
}
