package telematics

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type StopOutcome string

const (
	StopOutcomeRecorded  = StopOutcome("Recorded")
	StopOutcomeDuplicate = StopOutcome("Duplicate")
	StopOutcomeUnmatched = StopOutcome("Unmatched")
	StopOutcomeRefused   = StopOutcome("Refused")
)

const MaxStopOutcomeReasonLength = 500

func (o StopOutcome) IsValid() bool {
	switch o {
	case StopOutcomeRecorded, StopOutcomeDuplicate, StopOutcomeUnmatched, StopOutcomeRefused:
		return true
	}
	return false
}

func (o StopOutcome) NeedsReview() bool {
	return o == StopOutcomeUnmatched || o == StopOutcomeRefused
}

func (e *TelematicsEvent) RecordStopOutcome(result *StopVisitResult) {
	e.StopOutcome = result.Outcome
	e.StopVisit = result.Visit
	e.ShipmentMoveID = result.MoveID
	e.StopID = result.StopID
	e.StopOutcomeReason = stringutils.TruncateRunes(
		strings.TrimSpace(result.Reason),
		MaxStopOutcomeReasonLength,
	)
}

type StopVisitResult struct {
	Outcome StopOutcome
	Visit   shipment.VisitKind
	MoveID  pulid.ID
	StopID  pulid.ID
	Reason  string
}
