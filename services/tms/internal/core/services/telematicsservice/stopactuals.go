package telematicsservice

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	if !s.autoStopActualsEnabled(ctx, event.tenantInfo) {
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

	move, err := s.shipmentMoveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
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

func (s *Service) settleStopVisit(
	ctx context.Context,
	event *stopEvent,
	move *shipment.ShipmentMove,
) (*telematics.StopVisitResult, bool) {
	result := &telematics.StopVisitResult{Visit: event.visit, MoveID: move.ID}

	stop, match := move.MatchObservedVisit(event.record.LocationID, event.visit)
	switch match {
	case shipment.VisitMatchDuplicate:
		result.Outcome = telematics.StopOutcomeDuplicate
		return result, true
	case shipment.VisitMatchNoStop:
		if !event.providerStop {
			return result, false
		}
		result.Outcome = telematics.StopOutcomeUnmatched
		result.Reason = unmatchedStopReason(event)
		return result, true
	case shipment.VisitMatchStop:
	}

	result.StopID = stop.ID
	occurredAt := event.record.OccurredAt
	_, err := s.shipmentMoveService.RecordStopActual(ctx, &repositories.RecordStopActualRequest{
		TenantInfo: event.tenantInfo,
		MoveID:     move.ID,
		StopID:     stop.ID,
		Action:     stopActualAction(event.visit),
		OccurredAt: &occurredAt,
	})
	if err != nil {
		result.Outcome = telematics.StopOutcomeRefused
		result.Reason = s.refusedStopReason(event, stop, err)
		return result, true
	}

	result.Outcome = telematics.StopOutcomeRecorded
	s.l.Info("auto-recorded stop actual from telematics",
		zap.String("organizationId", event.tenantInfo.OrgID.String()),
		zap.String("stopId", stop.ID.String()),
		zap.String("visit", string(event.visit)),
		zap.Int64("occurredAt", occurredAt))

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
			&repositories.StopReview{Event: record, ShipmentID: shipmentID},
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

func stopActualAction(visit shipment.VisitKind) repositories.StopActualAction {
	if visit == shipment.VisitDeparture {
		return repositories.StopActualActionDepart
	}
	return repositories.StopActualActionArrive
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

func (s *Service) refusedStopReason(event *stopEvent, stop *shipment.Stop, err error) string {
	if errortypes.IsBusinessError(err) || errortypes.IsError(err) ||
		errortypes.IsNotFoundError(err) {
		return "Telematics reported the " + visitNoun(event.visit) + " at stop " +
			strconv.FormatInt(stop.Sequence, 10) + ", but it could not be recorded: " +
			refusalMessage(err)
	}

	s.l.Warn("failed to auto-record stop actual",
		zap.String("stopId", stop.ID.String()),
		zap.String("visit", string(event.visit)),
		zap.Error(err))
	return "Telematics reported the " + visitNoun(event.visit) + " at stop " +
		strconv.FormatInt(stop.Sequence, 10) +
		", but a system error stopped it being recorded. Record it on the stop by hand."
}

func refusalMessage(err error) string {
	if multiErr, ok := errors.AsType[*errortypes.MultiError](err); ok && len(multiErr.Errors) > 0 {
		messages := make([]string, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			messages = append(messages, fieldErr.Error())
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
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
