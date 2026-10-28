package ediservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type stopActualTarget string

const (
	stopTargetOrigin      = stopActualTarget("origin")
	stopTargetDestination = stopActualTarget("destination")
)

const inferredArrivalKeySuffix = ":arrival"

type stopActuals214Plan struct {
	target          stopActualTarget
	action          shipmentmoveports.StopActualAction
	arriveIfMissing bool
}

func (p stopActuals214Plan) visit() shipment.VisitKind {
	if p.action == shipmentmoveports.StopActualActionDepart {
		return shipment.VisitDeparture
	}
	return shipment.VisitArrival
}

func (t stopActualTarget) matches(stop *shipment.Stop) bool {
	switch t {
	case stopTargetOrigin:
		return stop.IsOriginStop()
	case stopTargetDestination:
		return stop.IsDestinationStop()
	default:
		return false
	}
}

// stopActualsPlanForStatusCode maps the carrier 214 status codes that carry a
// concrete arrival/departure meaning onto the stop-actual pipeline; every
// other code stays comment-and-event only.
func stopActualsPlanForStatusCode(statusCode string) (stopActuals214Plan, bool) {
	switch statusCode {
	case "X3":
		return stopActuals214Plan{
			target: stopTargetOrigin,
			action: shipmentmoveports.StopActualActionArrive,
		}, true
	case "AF":
		return stopActuals214Plan{
			target: stopTargetOrigin,
			action: shipmentmoveports.StopActualActionDepart,
		}, true
	case "X1":
		return stopActuals214Plan{
			target: stopTargetDestination,
			action: shipmentmoveports.StopActualActionArrive,
		}, true
	case "D1":
		return stopActuals214Plan{
			target:          stopTargetDestination,
			action:          shipmentmoveports.StopActualActionDepart,
			arriveIfMissing: true,
		}, true
	case "CD":
		return stopActuals214Plan{
			target: stopTargetDestination,
			action: shipmentmoveports.StopActualActionDepart,
		}, true
	default:
		return stopActuals214Plan{}, false
	}
}

// resolve214TargetStop names the stop a status is about: the stop whose
// sequence the carrier echoed in an L11*QN, else a stop of the status's role
// in the MS1 or N4 city, else the stop of that role the truck is at or is
// due at next.
func resolve214TargetStop(
	move *shipment.ShipmentMove,
	plan stopActuals214Plan,
	event *edi.ShipmentStatusEvent,
) (*shipment.Stop, trackingevent.MatchMethod) {
	if event != nil && event.StopSequence != nil {
		for _, stop := range move.ActiveStopsBySequence() {
			if stop.Sequence == *event.StopSequence {
				return stop, trackingevent.MatchStopReference
			}
		}
	}

	kind := plan.visit()
	if event != nil && event.HasPlace() {
		stop := move.StopForReportedVisit(func(stop *shipment.Stop) bool {
			return plan.target.matches(stop) && stopInPlace(stop, event.City, event.StateCode)
		}, kind)
		if stop != nil {
			return stop, trackingevent.MatchLocation
		}
	}

	if stop := move.StopForReportedVisit(plan.target.matches, kind); stop != nil {
		return stop, trackingevent.MatchStopRole
	}
	return nil, ""
}

func stopInPlace(stop *shipment.Stop, city, stateCode string) bool {
	if stop.Location == nil || stop.Location.State == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(stop.Location.City), strings.TrimSpace(city)) &&
		strings.EqualFold(
			strings.TrimSpace(stop.Location.State.Abbreviation),
			strings.TrimSpace(stateCode),
		)
}

type tendered214Target struct {
	plan    stopActuals214Plan
	move    *shipment.ShipmentMove
	stop    *shipment.Stop
	method  trackingevent.MatchMethod
	eventAt int64
}

// resolveTendered214Target finds the move and stop a mapped carrier status is
// about and reads the status's time in that stop's time zone when the carrier
// sent local time. Business-rule rejections degrade to warnings; only
// infrastructure failures propagate.
func (s *Service) resolveTendered214Target(
	ctx context.Context,
	req *RecordTenderedShipmentStatusRequest,
) (*tendered214Target, []string, error) {
	plan, ok := stopActualsPlanForStatusCode(req.StatusCode)
	if !ok || s.trackingEvents == nil || s.shipmentMoveRepo == nil {
		return nil, nil, nil
	}
	moveID := req.Offer.Tender.ShipmentMoveID
	if moveID.IsNil() {
		return nil, []string{
			fmt.Sprintf(
				"shipment status %s could not update stop actuals: the accepted tender offer has no shipment move",
				req.StatusCode,
			),
		}, nil
	}
	move, err := s.shipmentMoveRepo.GetByID(ctx, &shipmentmoveports.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		if errortypes.IsRefusal(err) {
			return nil, []string{fmt.Sprintf(
				"shipment status %s could not resolve the tendered move: %s",
				req.StatusCode,
				err.Error(),
			)}, nil
		}
		return nil, nil, err
	}

	stop, method := resolve214TargetStop(move, plan, req.Event)
	if stop == nil {
		return nil, []string{fmt.Sprintf(
			"shipment status %s has no active %s stop on the tendered move",
			req.StatusCode,
			plan.target,
		)}, nil
	}

	return &tendered214Target{
		plan:    plan,
		move:    move,
		stop:    stop,
		method:  method,
		eventAt: tendered214EventAt(req, stop),
	}, nil, nil
}

func tendered214EventAt(req *RecordTenderedShipmentStatusRequest, stop *shipment.Stop) int64 {
	if req.Event == nil || req.Event.DateTime.IsZero() || !req.Event.DateTime.IsLocal() {
		return req.EventAt
	}
	if stop.Location == nil || strings.TrimSpace(stop.Location.Timezone) == "" {
		return req.EventAt
	}
	zone, _ := timeutils.ResolveZone(stop.Location.Timezone)
	at, ok := req.Event.DateTime.Unix(zone)
	if !ok {
		return req.EventAt
	}
	return at
}

// applyTendered214StopActuals records a mapped carrier 214 status as tracking
// events on its stop. The events settle the stop's actuals by event time with
// every other source, so a late, repeated or out-of-order status never
// overwrites a better one. Refused and waiting statuses surface as warnings so
// the inbound file keeps its comment-and-event record; only infrastructure
// failures propagate.
func (s *Service) applyTendered214StopActuals(
	ctx context.Context,
	req *RecordTenderedShipmentStatusRequest,
	target *tendered214Target,
) ([]string, error) {
	if target == nil {
		return nil, nil
	}
	eventAt := target.eventAt
	if eventAt <= 0 {
		eventAt = timeutils.NowUnix()
	}

	sourceKey := tendered214SourceKey(req, target.stop, eventAt)
	reports := make([]*trackingeventports.RecordObservedStopEventParams, 0, 2)
	if target.plan.arriveIfMissing && target.stop.ActualArrival == nil {
		reports = append(reports, tendered214Report(req, target, tendered214ReportInput{
			kind:      shipment.VisitArrival,
			method:    trackingevent.MatchInferred,
			sourceKey: sourceKey + inferredArrivalKeySuffix,
			eventAt:   eventAt,
		}))
	}
	reports = append(reports, tendered214Report(req, target, tendered214ReportInput{
		kind:      target.plan.visit(),
		method:    target.method,
		sourceKey: sourceKey,
		eventAt:   eventAt,
	}))

	warnings := make([]string, 0, len(reports))
	for _, report := range reports {
		result, err := s.trackingEvents.RecordObserved(ctx, report)
		if err != nil {
			if errortypes.IsRefusal(err) {
				warnings = append(warnings, tendered214Warning(
					req, report, target.stop, "was not applied", errortypes.Summary(err),
				))
				continue
			}
			return nil, err
		}
		switch result.Outcome {
		case trackingevent.OutcomeRefused:
			warnings = append(warnings, tendered214Warning(
				req, report, target.stop, "was not applied", result.Reason,
			))
		case trackingevent.OutcomePending:
			warnings = append(warnings, tendered214Warning(
				req, report, target.stop, "is waiting", result.Reason,
			))
		case trackingevent.OutcomeApplied,
			trackingevent.OutcomeDuplicate,
			trackingevent.OutcomeSuperseded:
		}
	}
	if len(warnings) == 0 {
		return nil, nil
	}
	return warnings, nil
}

type tendered214ReportInput struct {
	kind      shipment.VisitKind
	method    trackingevent.MatchMethod
	sourceKey string
	eventAt   int64
}

func tendered214Report(
	req *RecordTenderedShipmentStatusRequest,
	target *tendered214Target,
	in tendered214ReportInput,
) *trackingeventports.RecordObservedStopEventParams {
	params := &trackingeventports.RecordObservedStopEventParams{
		TenantInfo:   req.TenantInfo,
		MoveID:       target.move.ID,
		StopID:       target.stop.ID,
		Kind:         in.kind,
		Source:       trackingevent.SourceEDI,
		SourceKey:    in.sourceKey,
		MatchMethod:  in.method,
		EventAt:      in.eventAt,
		SourceStatus: req.StatusCode,
		RawReference: req.RawReference,
	}
	if req.Event != nil && req.Event.Latitude != nil && req.Event.Longitude != nil {
		params.Latitude = req.Event.Latitude
		params.Longitude = req.Event.Longitude
	}
	return params
}

func tendered214SourceKey(
	req *RecordTenderedShipmentStatusRequest,
	stop *shipment.Stop,
	eventAt int64,
) string {
	return "edi:" + req.Offer.ID.String() + ":" + stop.ID.String() + ":" +
		req.StatusCode + ":" + strconv.FormatInt(eventAt, 10)
}

func tendered214Warning(
	req *RecordTenderedShipmentStatusRequest,
	report *trackingeventports.RecordObservedStopEventParams,
	stop *shipment.Stop,
	verb string,
	reason string,
) string {
	action := "arrival"
	if report.Kind == shipment.VisitDeparture {
		action = "departure"
	}
	return fmt.Sprintf(
		"shipment status %s %s at stop %d %s: %s",
		req.StatusCode,
		action,
		stop.Sequence,
		verb,
		reason,
	)
}
