package telematicsservice

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) autoStopActualsEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) bool {
	if s.dispatchControlRepo == nil {
		return false
	}
	dc, err := s.dispatchControlRepo.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		s.l.Warn("failed to load dispatch control for auto stop actuals",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
		return false
	}
	return dc.EnableAutoStopActuals
}

type stopEvent struct {
	tenantInfo   pagination.TenantInfo
	record       *telematics.TelematicsEvent
	visit        shipment.VisitKind
	providerStop bool
}

func (s *Service) applyStopEvent(ctx context.Context, event *stopEvent) {
	// The visit reaches the agents' waits whether or not stop actuals are
	// recorded from it: a wait on a truck reaching a stop is about the truck,
	// not about the paperwork.
	autoActuals := s.autoStopActualsEnabled(ctx, event.tenantInfo)
	if !autoActuals && s.waits == nil {
		return
	}

	assignment, err := s.resolveActiveAssignment(
		ctx,
		event.tenantInfo,
		event.record.TractorID,
		event.record.WorkerID,
	)
	if err != nil {
		s.l.Warn("failed to resolve assignment for telematics stop event",
			zap.String("eventId", event.record.ID.String()),
			zap.Error(err))
		return
	}
	if assignment == nil {
		return
	}

	move, err := s.shipmentMoveRepo.GetByID(ctx, &shipmentmoveports.GetMoveByIDRequest{
		MoveID:            assignment.ShipmentMoveID,
		TenantInfo:        event.tenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		s.l.Warn("failed to load move for telematics stop event",
			zap.String("eventId", event.record.ID.String()),
			zap.String("moveId", assignment.ShipmentMoveID.String()),
			zap.Error(err))
		return
	}
	if move == nil {
		return
	}

	s.notifyVisitWaits(ctx, event, move)
	if !autoActuals {
		return
	}

	result, ok := s.settleStopVisit(ctx, event, move)
	if !ok {
		return
	}

	event.record.RecordStopOutcome(result)
	if err = s.repo.RecordStopOutcome(ctx, event.record); err != nil {
		s.l.Warn("failed to store telematics stop outcome",
			zap.String("eventId", event.record.ID.String()),
			zap.Error(err))
	}

	s.projectStopVisit(ctx, event.tenantInfo, event.record, move.ShipmentID)
}

// notifyVisitWaits tells the waits on the move that the truck reached or left
// one of its stops, when the visit matches one. A visit that matches no stop
// on the move cannot be what any wait on it is for.
func (s *Service) notifyVisitWaits(
	ctx context.Context,
	event *stopEvent,
	move *shipment.ShipmentMove,
) {
	if s.waits == nil {
		return
	}
	stop, match := move.MatchObservedVisit(event.record.LocationID, event.visit)
	if stop == nil || match == shipment.VisitMatchNoStop {
		return
	}

	kind, verb := agenttrigger.EventShipmentMoveArrived, "entered"
	if event.visit == shipment.VisitDeparture {
		kind, verb = agenttrigger.EventShipmentMoveDeparted, "left"
	}
	at := event.record.OccurredAt
	if at == 0 {
		at = timeutils.NowUnix()
	}

	s.waits.NotifyEvent(ctx, &agentports.AgentEvent{
		Kind:       kind,
		SubjectID:  move.ID,
		TenantInfo: event.tenantInfo,
		Related:    []pulid.ID{stop.ID},
		Detail: fmt.Sprintf("The truck %s the geofence of stop %s at %s.", verb, stop.ID,
			time.Unix(at, 0).UTC().Format("Mon Jan 2 15:04 MST")),
	})
}

func (s *Service) settleStopVisit(
	ctx context.Context,
	event *stopEvent,
	move *shipment.ShipmentMove,
) (*telematics.StopVisitResult, bool) {
	result := &telematics.StopVisitResult{Visit: event.visit, MoveID: move.ID}

	stop := move.StopForObservedVisit(event.record.LocationID, event.visit)
	if stop == nil {
		if !event.providerStop {
			return result, false
		}
		result.Outcome = telematics.StopOutcomeUnmatched
		result.Reason = unmatchedStopReason(event)
		return result, true
	}

	result.StopID = stop.ID
	verdict, err := s.stopVisits.RecordStopVisit(ctx, &telematicsports.StopVisitReport{
		TenantInfo:   event.tenantInfo,
		Event:        event.record,
		MoveID:       move.ID,
		StopID:       stop.ID,
		Departure:    event.visit == shipment.VisitDeparture,
		ProviderStop: event.providerStop,
	})
	if err != nil {
		result.Outcome = telematics.StopOutcomeRefused
		result.Reason = s.refusedStopReason(event, stop, err)
		return result, true
	}

	result.Outcome = verdict.Outcome
	switch verdict.Outcome {
	case telematics.StopOutcomeRefused:
		result.Reason = refusedVisitReason(event, stop, verdict.Reason)
	case telematics.StopOutcomeRecorded:
		s.l.Info("auto-recorded stop actual from telematics",
			zap.String("organizationId", event.tenantInfo.OrgID.String()),
			zap.String("stopId", stop.ID.String()),
			zap.String("visit", string(event.visit)),
			zap.Int64("occurredAt", event.record.OccurredAt))
	case telematics.StopOutcomeDuplicate,
		telematics.StopOutcomeUnmatched,
		telematics.StopOutcomePending:
		result.Reason = verdict.Reason
	}

	return result, true
}

func (s *Service) projectStopVisit(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *telematics.TelematicsEvent,
	shipmentID pulid.ID,
) {
	if s.watchtower == nil {
		return
	}

	switch {
	case record.StopOutcome.NeedsReview():
		s.watchtower.Upsert(ctx, watchtowersources.DescribeTelematicsStopVisit(
			&telematicsports.StopReview{Event: record, ShipmentID: shipmentID},
		))
	case record.StopOutcome == telematics.StopOutcomeRecorded:
		s.watchtower.Resolve(
			ctx,
			tenantInfo,
			watchtower.SourceTelematicsStopVisit,
			watchtowersources.TelematicsStopVisitSourceID(record),
		)
	}
}

func visitNoun(visit shipment.VisitKind) string {
	if visit == shipment.VisitDeparture {
		return "departure"
	}
	return "arrival"
}

func unmatchedStopReason(event *stopEvent) string {
	if event.record.LocationID.IsNil() {
		return "The provider reported this " + visitNoun(event.visit) +
			" at a stop that is not linked to a Trenova location, so it could not be matched to a stop on the load. Record it on the stop by hand."
	}
	return "The provider reported this " + visitNoun(event.visit) +
		" at a location that is not a stop on the driver's current load. Check the load's stops and record the visit by hand."
}

func refusedVisitReason(event *stopEvent, stop *shipment.Stop, reason string) string {
	return "Telematics reported the " + visitNoun(event.visit) + " at stop " +
		strconv.FormatInt(stop.Sequence, 10) + ", but it could not be recorded: " + reason
}

func (s *Service) refusedStopReason(event *stopEvent, stop *shipment.Stop, err error) string {
	if errortypes.IsBusinessError(err) || errortypes.IsError(err) ||
		errortypes.IsNotFoundError(err) {
		return refusedVisitReason(event, stop, errortypes.Summary(err))
	}

	s.l.Warn("failed to auto-record stop actual",
		zap.String("stopId", stop.ID.String()),
		zap.String("visit", string(event.visit)),
		zap.Error(err))
	return "Telematics reported the " + visitNoun(event.visit) + " at stop " +
		strconv.FormatInt(stop.Sequence, 10) +
		", but a system error stopped it being recorded. Record it on the stop by hand."
}

func (s *Service) resolveActiveAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tractorID pulid.ID,
	workerID pulid.ID,
) (*shipment.Assignment, error) {
	if s.assignmentRepo == nil {
		return nil, nil
	}
	if !tractorID.IsNil() {
		assignment, err := s.assignmentRepo.FindActiveByTractorID(ctx, tenantInfo, tractorID)
		if err != nil {
			return nil, err
		}
		if assignment != nil {
			return assignment, nil
		}
	}
	if !workerID.IsNil() {
		return s.assignmentRepo.FindActiveByWorkerID(ctx, tenantInfo, workerID)
	}
	return nil, nil
}

func stopExternalLocationID(externalIDs map[string]string) (pulid.ID, bool) {
	if locationID, ok := externalIDs["trenovaLocationId"]; ok {
		if parsed, parseErr := pulid.Parse(locationID); parseErr == nil {
			return parsed, true
		}
	}
	return pulid.Nil, false
}
