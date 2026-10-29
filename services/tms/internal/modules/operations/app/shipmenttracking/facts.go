package shipmenttracking

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Fact struct {
	Name   string
	Value  string
	SeenAt int64
	Stale  bool
}

func (s *Snapshot) Facts() []Fact {
	facts := make([]Fact, 0, 7)
	status := s.Status
	if s.Customer != "" {
		status += " for " + s.Customer
	}
	facts = append(facts, Fact{Name: "status", Value: status})
	if s.Canceled != nil {
		facts = append(
			facts,
			Fact{Name: "canceled", Value: s.Canceled.text(), SeenAt: s.Canceled.At},
		)
	}
	if s.Freight != "" {
		facts = append(facts, Fact{Name: "freight", Value: s.Freight})
	}
	if len(s.Flags) > 0 {
		facts = append(facts, Fact{Name: "needs attention", Value: strings.Join(s.Flags, "; ")})
	}

	if s.NextStop == nil {
		facts = append(facts, Fact{Name: "stops", Value: "every stop is complete"})
	} else {
		facts = append(facts, Fact{Name: "next stop", Value: nextStopText(s.NextStop)})
	}
	if coverage := s.coverageText(); coverage != "" {
		facts = append(facts, Fact{Name: "covered by", Value: coverage})
	}

	if s.Position != nil {
		value := positionText(s.Position)
		if s.Position.Tractor != "" {
			value = s.Position.Tractor + " " + value
		}
		if s.Position.SpeedMph > 0 {
			value += fmt.Sprintf(", moving at %.0f mph", s.Position.SpeedMph)
		}
		facts = append(facts, Fact{
			Name: "last position", Value: value,
			SeenAt: s.Position.RecordedAt, Stale: s.Position.Stale,
		})
	}
	if s.Driver != nil {
		facts = append(facts, Fact{
			Name:   "driver hours",
			Value:  driverHoursText(s.Driver),
			SeenAt: s.Driver.RecordedAt,
			Stale:  s.Driver.Stale,
		})
	}
	if s.Estimate != nil && s.Estimate.Verdict != VerdictUnknown &&
		s.Estimate.EstimatedArrival > 0 {
		facts = append(facts, Fact{
			Name: "estimated arrival",
			Value: fmt.Sprintf("%s at %s, %s",
				s.Estimate.EstimatedArrivalText, s.Estimate.StopLabel, verdictText(s.Estimate)),
			SeenAt: estimateSeenAt(s.Estimate),
			Stale:  s.Estimate.InputStale,
		})
	}

	return facts
}

func estimateSeenAt(e *ArrivalEstimate) int64 {
	if e.InputAt > 0 {
		return e.InputAt
	}
	return e.ComputedAt
}

func nextStopText(stop *StopSnapshot) string {
	text := fmt.Sprintf(
		"%s at %s, due by %s",
		strings.ToLower(stop.Type),
		stopPlace(stop),
		dueText(stop),
	)
	switch {
	case stop.ActualArrival > 0 && stop.ActualDeparture == 0:
		text += ", arrived " + stop.ActualArrivalText + " and not yet departed"
	case stop.Overdue:
		text += fmt.Sprintf(", overdue by %s with no arrival",
			timeutils.FormatLongDurationMs(stop.LateMinutes*msPerMinute))
	}

	return text
}

func (s *Snapshot) coverageText() string {
	moveID := ""
	if s.NextStop != nil {
		moveID = s.NextStop.MoveID
	}
	for idx := range s.Moves {
		move := &s.Moves[idx]
		if moveID != "" && move.ID != moveID {
			continue
		}
		switch {
		case move.Carrier != "":
			return "carrier " + move.Carrier
		case move.Driver != "" && move.Tractor != "":
			return move.Driver + " on " + move.Tractor
		case move.Driver != "":
			return move.Driver
		case move.Status == string(shipmentvocab.MoveStatusCompleted),
			move.Status == string(shipmentvocab.MoveStatusCanceled):
			return "nobody on record"
		default:
			return "nobody: the move needs a driver or carrier"
		}
	}

	return ""
}

func driverHoursText(driver *DriverSnapshot) string {
	name := driver.Name
	if name == "" {
		name = "the driver"
	}
	duty := driver.DutyStatus
	if duty == "" {
		duty = "duty status unknown"
	}

	return fmt.Sprintf("%s is %s, %s drive and %s on duty left, %s in the cycle",
		name, duty,
		timeutils.FormatLongDurationMs(driver.DriveRemainingMinutes*msPerMinute),
		timeutils.FormatLongDurationMs(driver.ShiftRemainingMinutes*msPerMinute),
		timeutils.FormatLongDurationMs(driver.CycleRemainingMinutes*msPerMinute))
}

func PositionStale(recordedAt, now int64) bool {
	return now-recordedAt > positionStaleAfterSeconds
}

func HOSStale(recordedAt, now int64) bool {
	return now-recordedAt > hosStaleAfterSeconds
}

// freightText is what the load weighs and how many pieces, so a follow-up such
// as "make it 39,250" reads against the weight the conversation was about
// rather than a rate it never mentioned.
func freightText(weight, pieces *int64) string {
	parts := make([]string, 0, 2)
	if weight != nil && *weight > 0 {
		parts = append(parts, intutils.FormatWithCommas(*weight)+" lb")
	}
	if pieces != nil && *pieces > 0 {
		parts = append(parts, intutils.FormatWithCommas(*pieces)+" pieces")
	}

	return strings.Join(parts, ", ")
}

func (c *CanceledSnapshot) text() string {
	switch {
	case c.By != "" && c.Reason != "":
		return "by " + c.By + ": " + c.Reason
	case c.By != "":
		return "by " + c.By + ", no reason given"
	case c.Reason != "":
		return c.Reason
	default:
		return "no reason given"
	}
}
