package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safetyDay = int64(86400)

func safetyEvent(kind worker.SafetyEventKind, occurredAgo int64, mutate func(*worker.WorkerSafetyEvent)) *worker.WorkerSafetyEvent {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	e := &worker.WorkerSafetyEvent{
		ID:          pulid.MustNew("wsev_"),
		WorkerID:    pulid.MustNew("wrk_"),
		Kind:        kind,
		Severity:    worker.SafetySeverityMinor,
		Status:      worker.SafetyEventStatusClosed,
		OccurredAt:  now - occurredAgo*safetyDay,
		Description: "test",
	}
	if mutate != nil {
		mutate(e)
	}
	return e
}

func TestDefaultSafetyPoints(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int32(2), worker.DefaultSafetyPoints(worker.SafetyEventAccident, worker.SafetySeverityMinor, false, ""))
	assert.Equal(t, int32(8), worker.DefaultSafetyPoints(worker.SafetyEventAccident, worker.SafetySeverityMajor, true, ""))
	assert.Equal(t, int32(4), worker.DefaultSafetyPoints(worker.SafetyEventIncident, worker.SafetySeverityCritical, true, ""))
	assert.Equal(t, int32(3), worker.DefaultSafetyPoints(worker.SafetyEventCitation, worker.SafetySeverityModerate, false, ""))
	assert.Equal(t, int32(0), worker.DefaultSafetyPoints(worker.SafetyEventNearMiss, worker.SafetySeverityCritical, true, ""))
	assert.Equal(t, int32(0), worker.DefaultSafetyPoints(worker.SafetyEventInspection, worker.SafetySeverityMinor, false, worker.InspectionResultPass))
	assert.Equal(t, int32(5), worker.DefaultSafetyPoints(worker.SafetyEventInspection, worker.SafetySeverityMinor, false, worker.InspectionResultOutOfService))
}

func TestSafetyEventValidate(t *testing.T) {
	t.Parallel()

	t.Run("inspections need a result and only inspections carry one", func(t *testing.T) {
		t.Parallel()
		e := safetyEvent(worker.SafetyEventInspection, 1, nil)
		e.Status = worker.SafetyEventStatusOpen
		multiErr := errortypes.NewMultiError()
		e.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "inspectionResult", multiErr.Errors[0].Field)

		c := safetyEvent(worker.SafetyEventCitation, 1, func(e *worker.WorkerSafetyEvent) {
			e.Status = worker.SafetyEventStatusOpen
			e.InspectionResult = worker.InspectionResultPass
		})
		multiErr = errortypes.NewMultiError()
		c.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "inspectionResult", multiErr.Errors[0].Field)
	})

	t.Run("closing needs a resolution and money cannot be negative", func(t *testing.T) {
		t.Parallel()
		e := safetyEvent(worker.SafetyEventAccident, 3, func(e *worker.WorkerSafetyEvent) {
			e.CostAmount = decimal.NewNullDecimal(decimal.NewFromInt(-5))
		})
		multiErr := errortypes.NewMultiError()
		e.Validate(multiErr)
		fields := make([]string, 0, len(multiErr.Errors))
		for _, err := range multiErr.Errors {
			fields = append(fields, err.Field)
		}
		assert.Contains(t, fields, "resolution")
		assert.Contains(t, fields, "costAmount")
	})
}

func TestSafetyEventPoints(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	e := safetyEvent(worker.SafetyEventAccident, 10, func(e *worker.WorkerSafetyEvent) { e.Points = 4 })
	e.DefaultPointsExpiry()
	require.NotNil(t, e.PointsExpireAt)
	assert.Equal(t, timeutils.AddMonthsUTC(e.OccurredAt, 24), *e.PointsExpireAt, "points roll off after two years")
	assert.Equal(t, int32(4), e.ActivePoints(now))
	assert.Equal(t, int32(0), e.ActivePoints(*e.PointsExpireAt), "expired on the day")

	zero := safetyEvent(worker.SafetyEventNearMiss, 1, nil)
	zero.DefaultPointsExpiry()
	assert.Nil(t, zero.PointsExpireAt)
}

func TestBuildSafetyScorecard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	workerID := pulid.MustNew("wrk_")

	t.Run("clean record is excellent", func(t *testing.T) {
		t.Parallel()
		card := worker.BuildSafetyScorecard(workerID, nil, nil, nil, now)
		assert.Equal(t, int32(100), card.Score)
		assert.Equal(t, worker.SafetyRatingExcellent, card.Rating)
		assert.Nil(t, card.CleanInspectionRate)
		assert.Nil(t, card.DaysSinceLastEvent)
	})

	t.Run("points, preventable accidents, out-of-service and discipline all cost", func(t *testing.T) {
		t.Parallel()
		expired := now - 30*safetyDay
		events := []*worker.WorkerSafetyEvent{
			safetyEvent(worker.SafetyEventAccident, 40, func(e *worker.WorkerSafetyEvent) {
				e.Points = 4
				e.Preventable = true
				e.Status = worker.SafetyEventStatusOpen
			}),
			safetyEvent(worker.SafetyEventInspection, 20, func(e *worker.WorkerSafetyEvent) {
				e.InspectionResult = worker.InspectionResultOutOfService
				e.Points = 5
			}),
			safetyEvent(worker.SafetyEventInspection, 100, func(e *worker.WorkerSafetyEvent) {
				e.InspectionResult = worker.InspectionResultPass
			}),
			safetyEvent(worker.SafetyEventCitation, 500, func(e *worker.WorkerSafetyEvent) {
				e.Points = 3
				e.PointsExpireAt = &expired
			}),
			safetyEvent(worker.SafetyEventNearMiss, 5, nil),
		}
		actions := []*worker.WorkerDisciplinaryAction{
			{Level: worker.DisciplinaryLevelWrittenWarning, Status: worker.DisciplinaryStatusActive, IssuedAt: now - 10*safetyDay},
			{Level: worker.DisciplinaryLevelSuspension, Status: worker.DisciplinaryStatusRescinded, IssuedAt: now - 5*safetyDay},
		}
		recognitions := []*worker.WorkerRecognition{{OccurredAt: now - 3*safetyDay}, {OccurredAt: now - 500*safetyDay}}

		card := worker.BuildSafetyScorecard(workerID, events, actions, recognitions, now)

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
		assert.Equal(t, worker.DisciplinaryLevelWrittenWarning, card.HighestDiscipline)
		assert.Equal(t, int32(1), card.Recognitions)
		require.NotNil(t, card.CleanInspectionRate)
		assert.InDelta(t, 0.5, *card.CleanInspectionRate, 0.001)
		require.NotNil(t, card.DaysSinceLastEvent)
		assert.Equal(t, int64(40), *card.DaysSinceLastEvent, "near misses and inspections do not reset the streak")

		// 100 - 9*5 - 1*10 - 1*15 - 1*5 = 25
		assert.Equal(t, int32(25), card.Score)
		assert.Equal(t, worker.SafetyRatingAtRisk, card.Rating)
	})

	t.Run("points alone can put a worker on watch", func(t *testing.T) {
		t.Parallel()
		events := []*worker.WorkerSafetyEvent{
			safetyEvent(worker.SafetyEventCitation, 400, func(e *worker.WorkerSafetyEvent) { e.Points = 6 }),
		}
		card := worker.BuildSafetyScorecard(workerID, events, nil, nil, now)
		assert.Equal(t, int32(70), card.Score)
		assert.Equal(t, worker.SafetyRatingWatch, card.Rating)
	})
}

func TestDisciplinaryLadder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()

	clean := worker.BuildDisciplinaryLadder(nil, now)
	assert.Equal(t, worker.DisciplinaryLevelCoaching, clean.SuggestedLevel)
	assert.False(t, clean.AtFinalStep)

	lapsed := now - safetyDay
	actions := []*worker.WorkerDisciplinaryAction{
		{Level: worker.DisciplinaryLevelCoaching, Status: worker.DisciplinaryStatusActive, IssuedAt: now - 300*safetyDay},
		{Level: worker.DisciplinaryLevelFinalWarning, Status: worker.DisciplinaryStatusActive, IssuedAt: now - 400*safetyDay, ExpiresAt: &lapsed},
		{Level: worker.DisciplinaryLevelWrittenWarning, Status: worker.DisciplinaryStatusActive, IssuedAt: now - 60*safetyDay},
	}
	ladder := worker.BuildDisciplinaryLadder(actions, now)
	require.Len(t, ladder.ActiveActions, 2)
	assert.Equal(t, worker.DisciplinaryLevelWrittenWarning, ladder.HighestLevel, "the lapsed final warning is off the ladder")
	assert.Equal(t, worker.DisciplinaryLevelFinalWarning, ladder.SuggestedLevel)

	top := worker.BuildDisciplinaryLadder([]*worker.WorkerDisciplinaryAction{
		{Level: worker.DisciplinaryLevelSuspension, Status: worker.DisciplinaryStatusActive, IssuedAt: now - safetyDay},
	}, now)
	assert.Equal(t, worker.DisciplinaryLevelTermination, top.SuggestedLevel)
	assert.True(t, top.AtFinalStep)
	assert.Equal(t, worker.DisciplinaryLevelTermination, worker.DisciplinaryLevelTermination.Next())

	action := &worker.WorkerDisciplinaryAction{
		WorkerID: pulid.MustNew("wrk_"), Level: worker.DisciplinaryLevelSuspension,
		Status: worker.DisciplinaryStatusActive, Reason: "Repeated late departures", IssuedAt: now,
	}
	multiErr := errortypes.NewMultiError()
	action.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "suspensionDays", multiErr.Errors[0].Field)
	action.DefaultExpiry()
	require.NotNil(t, action.ExpiresAt)
	assert.Equal(t, timeutils.AddMonthsUTC(now, 12), *action.ExpiresAt)

	termination := &worker.WorkerDisciplinaryAction{Level: worker.DisciplinaryLevelTermination, IssuedAt: now}
	termination.DefaultExpiry()
	assert.Nil(t, termination.ExpiresAt, "terminations never roll off")
}

func TestPerformanceReviewScoring(t *testing.T) {
	t.Parallel()
	template := &worker.PerformanceReviewTemplate{
		Items: []worker.ReviewItem{
			{Key: "safety", Label: "Safety", Weight: 3},
			{Key: "service", Label: "Customer service", Weight: 2},
			{Key: "paperwork", Label: "Paperwork", Weight: 1},
		},
	}
	ratings := worker.RatingsFromTemplate(template)
	require.Len(t, ratings, 3)
	assert.False(t, worker.ComputeOverallScore(ratings).Valid, "nothing rated yet")

	five, three, four := int32(5), int32(3), int32(4)
	ratings[0].Score = &five
	ratings[1].Score = &three
	ratings[2].Score = &four
	score := worker.ComputeOverallScore(ratings)
	require.True(t, score.Valid)
	// (5*3 + 3*2 + 4*1) / 6 = 25/6 = 4.17
	assert.Equal(t, "4.17", score.Decimal.StringFixed(2))

	review := &worker.PerformanceReview{
		WorkerID: pulid.MustNew("wrk_"), TemplateID: pulid.MustNew("prt_"), Status: worker.ReviewStatusDraft,
		Title: "H1 2026", PeriodStart: 1_700_000_000, PeriodEnd: 1_710_000_000,
		Ratings: worker.RatingsFromTemplate(template),
	}
	draft := errortypes.NewMultiError()
	review.Validate(draft)
	assert.False(t, draft.HasErrors(), "a draft may be unrated")

	submit := errortypes.NewMultiError()
	review.ValidateForSubmit(submit)
	require.True(t, submit.HasErrors())
	fields := make([]string, 0, len(submit.Errors))
	for _, err := range submit.Errors {
		fields = append(fields, err.Field)
	}
	assert.Contains(t, fields, "ratings[0].score")
	assert.Contains(t, fields, "summary")

	cadence := int32(6)
	template.CadenceMonths = &cadence
	next := review.NextReviewDate(template)
	require.NotNil(t, next)
	assert.Equal(t, timeutils.AddMonthsUTC(review.PeriodEnd, 6), *next)
}
