package safetyscorecard

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	scorecardWindowMonths       = 12
	scorePenaltyPerPoint        = 5
	scorePenaltyPreventable     = 10
	scorePenaltyOutOfService    = 15
	scorePenaltyOpenDiscipline  = 5
	scoreExcellentFloor         = 90
	scoreGoodFloor              = 75
	scoreWatchFloor             = 50
	safetyPointsWatchThreshold  = 6
	safetyPointsAtRiskThreshold = 10
)

// SafetyScorecard is the twelve-month picture of a worker's record plus the
// points still active from the two-year look-back.
type SafetyScorecard struct {
	WorkerID              pulid.ID
	AsOf                  int64
	Score                 int32
	Rating                worker.SafetyRating
	ActivePoints          int32
	PointsWatchThreshold  int32
	PointsAtRiskThreshold int32
	Accidents             int32
	PreventableAccidents  int32
	Incidents             int32
	NearMisses            int32
	Citations             int32
	Inspections           int32
	InspectionsPassed     int32
	InspectionsFailed     int32
	OutOfServiceOrders    int32
	CleanInspectionRate   *float64
	OpenEvents            int32
	ActiveDiscipline      int32
	HighestDiscipline     workerdiscipline.DisciplinaryLevel
	DaysSinceLastEvent    *int64
	LastEventAt           *int64
	Recognitions          int32
}

// BuildSafetyScorecard is the pure roll-up. Counts cover the trailing twelve
// months; points count until they expire; the score starts at 100 and loses
// ground for active points, preventable accidents, out-of-service orders and
// open discipline.
func BuildSafetyScorecard(
	workerID pulid.ID,
	events []*workersafety.WorkerSafetyEvent,
	actions []*workerdiscipline.WorkerDisciplinaryAction,
	recognitions []*workerdiscipline.WorkerRecognition,
	now int64,
) *SafetyScorecard {
	card := &SafetyScorecard{
		WorkerID:              workerID,
		AsOf:                  now,
		PointsWatchThreshold:  safetyPointsWatchThreshold,
		PointsAtRiskThreshold: safetyPointsAtRiskThreshold,
	}
	windowStart := timeutils.AddMonthsUTC(now, -scorecardWindowMonths)

	card.tallyEvents(events, now, windowStart)
	card.tallyDiscipline(actions, now)
	for _, recognition := range recognitions {
		if recognition != nil && recognition.OccurredAt >= windowStart {
			card.Recognitions++
		}
	}

	score := int32(100) -
		card.ActivePoints*scorePenaltyPerPoint -
		card.PreventableAccidents*scorePenaltyPreventable -
		card.OutOfServiceOrders*scorePenaltyOutOfService -
		card.ActiveDiscipline*scorePenaltyOpenDiscipline
	if score < 0 {
		score = 0
	}
	card.Score = score
	card.Rating = ratingFor(score, card.ActivePoints)
	return card
}

func (card *SafetyScorecard) tallyEvents(
	events []*workersafety.WorkerSafetyEvent,
	now, windowStart int64,
) {
	var lastEvent int64
	for _, event := range events {
		if event == nil {
			continue
		}
		card.ActivePoints += event.ActivePoints(now)
		if !event.IsClosed() {
			card.OpenEvents++
		}
		if event.Kind.CountsAgainstRecord() && event.OccurredAt > lastEvent {
			lastEvent = event.OccurredAt
		}
		if event.OccurredAt >= windowStart {
			card.countWindowEvent(event)
		}
	}
	if card.Inspections > 0 {
		rate := float64(card.InspectionsPassed) / float64(card.Inspections)
		card.CleanInspectionRate = &rate
	}
	if lastEvent > 0 {
		days := workercredential.DaysUntil(now, lastEvent)
		card.LastEventAt = &lastEvent
		card.DaysSinceLastEvent = &days
	}
}

func (card *SafetyScorecard) countWindowEvent(event *workersafety.WorkerSafetyEvent) {
	switch event.Kind {
	case workersafety.SafetyEventAccident:
		card.Accidents++
		if event.Preventable {
			card.PreventableAccidents++
		}
	case workersafety.SafetyEventIncident:
		card.Incidents++
	case workersafety.SafetyEventNearMiss:
		card.NearMisses++
	case workersafety.SafetyEventCitation:
		card.Citations++
	case workersafety.SafetyEventInspection:
		card.countInspection(event)
	}
}

func (card *SafetyScorecard) countInspection(event *workersafety.WorkerSafetyEvent) {
	card.Inspections++
	switch event.InspectionResult {
	case workersafety.InspectionResultPass:
		card.InspectionsPassed++
	case workersafety.InspectionResultFail:
		card.InspectionsFailed++
	case workersafety.InspectionResultOutOfService:
		card.InspectionsFailed++
		card.OutOfServiceOrders++
	case workersafety.InspectionResultNone:
	}
	if event.OutOfService &&
		event.InspectionResult != workersafety.InspectionResultOutOfService {
		card.OutOfServiceOrders++
	}
}

func (card *SafetyScorecard) tallyDiscipline(
	actions []*workerdiscipline.WorkerDisciplinaryAction,
	now int64,
) {
	for _, action := range actions {
		if action == nil || !action.IsActive(now) {
			continue
		}
		card.ActiveDiscipline++
		if action.Level.Rank() > card.HighestDiscipline.Rank() {
			card.HighestDiscipline = action.Level
		}
	}
}

func ratingFor(score, points int32) worker.SafetyRating {
	switch {
	case points >= safetyPointsAtRiskThreshold || score < scoreWatchFloor:
		return worker.SafetyRatingAtRisk
	case points >= safetyPointsWatchThreshold || score < scoreGoodFloor:
		return worker.SafetyRatingWatch
	case score < scoreExcellentFloor:
		return worker.SafetyRatingGood
	default:
		return worker.SafetyRatingExcellent
	}
}
