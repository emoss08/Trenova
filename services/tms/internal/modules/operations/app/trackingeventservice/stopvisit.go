package trackingeventservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const telematicsRawReferencePrefix = "telematics_event:"

func (s *Service) RecordStopVisit(
	ctx context.Context,
	report *telematicsports.StopVisitReport,
) (*telematicsports.StopVisitVerdict, error) {
	if report == nil || report.Event == nil {
		return nil, errortypes.NewValidationError(
			"event",
			errortypes.ErrRequired,
			"Telematics event is required",
		)
	}

	result, err := s.RecordObserved(ctx, stopVisitParams(report))
	if err != nil {
		return nil, err
	}

	return &telematicsports.StopVisitVerdict{
		Outcome: stopOutcomeFor(result.Outcome),
		Reason:  result.Reason,
	}, nil
}

func stopVisitParams(
	report *telematicsports.StopVisitReport,
) *trackingeventports.RecordObservedStopEventParams {
	event := report.Event
	kind := shipmentvocab.VisitArrival
	if report.Departure {
		kind = shipmentvocab.VisitDeparture
	}
	method := trackingevent.MatchLocation
	if report.ProviderStop {
		method = trackingevent.MatchStopReference
	}

	return &trackingeventports.RecordObservedStopEventParams{
		TenantInfo:   report.TenantInfo,
		MoveID:       report.MoveID,
		StopID:       report.StopID,
		Kind:         kind,
		Source:       trackingevent.SourceTelematics,
		SourceKey:    event.Provider + ":" + event.EventID,
		MatchMethod:  method,
		EventAt:      event.OccurredAt,
		SourceStatus: string(event.EventType),
		RawReference: telematicsRawReferencePrefix + event.ID.String(),
	}
}

func stopOutcomeFor(outcome trackingevent.Outcome) telematics.StopOutcome {
	switch outcome {
	case trackingevent.OutcomeApplied:
		return telematics.StopOutcomeRecorded
	case trackingevent.OutcomeDuplicate, trackingevent.OutcomeSuperseded:
		return telematics.StopOutcomeDuplicate
	case trackingevent.OutcomePending:
		return telematics.StopOutcomePending
	case trackingevent.OutcomeRefused:
		return telematics.StopOutcomeRefused
	default:
		return telematics.StopOutcomeRefused
	}
}
