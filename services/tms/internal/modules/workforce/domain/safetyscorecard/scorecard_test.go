package safetyscorecard_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safetyDay = int64(86400)

func safetyEvent(
	kind workersafety.SafetyEventKind,
	occurredAgo int64,
	mutate func(*workersafety.WorkerSafetyEvent),
) *workersafety.WorkerSafetyEvent {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	e := &workersafety.WorkerSafetyEvent{
		ID:          pulid.MustNew("wsev_"),
		WorkerID:    pulid.MustNew("wrk_"),
		Kind:        kind,
		Severity:    workersafety.SafetySeverityMinor,
		Status:      workersafety.SafetyEventStatusClosed,
		OccurredAt:  now - occurredAgo*safetyDay,
		Description: "test",
	}
	if mutate != nil {
		mutate(e)
	}
	return e
}

func TestBuildSafetyScorecard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	workerID := pulid.MustNew("wrk_")

	t.Run("clean record is excellent", func(t *testing.T) {
		t.Parallel()
		card := safetyscorecard.BuildSafetyScorecard(workerID, nil, nil, nil, now)
		assert.Equal(t, int32(100), card.Score)
		assert.Equal(t, worker.SafetyRatingExcellent, card.Rating)
		assert.Nil(t, card.CleanInspectionRate)
		assert.Nil(t, card.DaysSinceLastEvent)
	})

	t.Run(
		"points, preventable accidents, out-of-service and discipline all cost",
		func(t *testing.T) {
			t.Parallel()
			expired := now - 30*safetyDay
			events := []*workersafety.WorkerSafetyEvent{
				safetyEvent(
					workersafety.SafetyEventAccident,
					40,
					func(e *workersafety.WorkerSafetyEvent) {
						e.Points = 4
						e.Preventable = true
						e.Status = workersafety.SafetyEventStatusOpen
					},
				),
				safetyEvent(
					workersafety.SafetyEventInspection,
					20,
					func(e *workersafety.WorkerSafetyEvent) {
						e.InspectionResult = workersafety.InspectionResultOutOfService
						e.Points = 5
					},
				),
				safetyEvent(
					workersafety.SafetyEventInspection,
					100,
					func(e *workersafety.WorkerSafetyEvent) {
						e.InspectionResult = workersafety.InspectionResultPass
					},
				),
				safetyEvent(
					workersafety.SafetyEventCitation,
					500,
					func(e *workersafety.WorkerSafetyEvent) {
						e.Points = 3
						e.PointsExpireAt = &expired
					},
				),
				safetyEvent(workersafety.SafetyEventNearMiss, 5, nil),
			}
			actions := []*workerdiscipline.WorkerDisciplinaryAction{
				{
					Level:    workerdiscipline.DisciplinaryLevelWrittenWarning,
					Status:   workerdiscipline.DisciplinaryStatusActive,
					IssuedAt: now - 10*safetyDay,
				},
				{
					Level:    workerdiscipline.DisciplinaryLevelSuspension,
					Status:   workerdiscipline.DisciplinaryStatusRescinded,
					IssuedAt: now - 5*safetyDay,
				},
			}
			recognitions := []*workerdiscipline.WorkerRecognition{
				{OccurredAt: now - 3*safetyDay},
				{OccurredAt: now - 500*safetyDay},
			}

			card := safetyscorecard.BuildSafetyScorecard(
				workerID,
				events,
				actions,
				recognitions,
				now,
			)

			assert.Equal(t, int32(9), card.ActivePoints, "the expired citation no longer counts")
			assert.Equal(t, int32(1), card.Accidents)
			assert.Equal(t, int32(1), card.PreventableAccidents)
			assert.Equal(t, int32(0), card.Citations, "the citation is older than twelve months")
			assert.Equal(t, int32(2), card.Inspections)
			assert.Equal(t, int32(1), card.InspectionsPassed)
			assert.Equal(t, int32(1), card.OutOfServiceOrders)
			assert.Equal(t, int32(1), card.NearMisses)
			assert.Equal(t, int32(1), card.OpenEvents)
			assert.Equal(t, int32(1), card.ActiveDiscipline, "rescinded actions are off the ladder")
			assert.Equal(
				t,
				workerdiscipline.DisciplinaryLevelWrittenWarning,
				card.HighestDiscipline,
			)
			assert.Equal(t, int32(1), card.Recognitions)
			require.NotNil(t, card.CleanInspectionRate)
			assert.InDelta(t, 0.5, *card.CleanInspectionRate, 0.001)
			require.NotNil(t, card.DaysSinceLastEvent)
			assert.Equal(
				t,
				int64(40),
				*card.DaysSinceLastEvent,
				"near misses and inspections do not reset the streak",
			)

			// 100 - 9*5 - 1*10 - 1*15 - 1*5 = 25
			assert.Equal(t, int32(25), card.Score)
			assert.Equal(t, worker.SafetyRatingAtRisk, card.Rating)
		},
	)

	t.Run("points alone can put a worker on watch", func(t *testing.T) {
		t.Parallel()
		events := []*workersafety.WorkerSafetyEvent{
			safetyEvent(
				workersafety.SafetyEventCitation,
				400,
				func(e *workersafety.WorkerSafetyEvent) { e.Points = 6 },
			),
		}
		card := safetyscorecard.BuildSafetyScorecard(workerID, events, nil, nil, now)
		assert.Equal(t, int32(70), card.Score)
		assert.Equal(t, worker.SafetyRatingWatch, card.Rating)
	})
}
