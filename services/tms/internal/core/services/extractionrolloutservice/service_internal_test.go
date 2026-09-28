package extractionrolloutservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (w *world) update(
	t *testing.T,
	mutate func(*services.UpdateExtractionRolloutRequest),
) (*extractionrollout.ExtractionRollout, error) {
	t.Helper()

	current, err := w.svc.Get(t.Context(), testTenant())
	require.NoError(t, err)
	req := &services.UpdateExtractionRolloutRequest{
		TenantInfo:                 testTenant(),
		Enabled:                    true,
		ProviderID:                 w.candidate.ID,
		Percent:                    10,
		MaxAccuracyDropPoints:      current.MaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: current.MaxRejectionIncreasePoints,
		Version:                    current.Version,
	}
	if mutate != nil {
		mutate(req)
	}

	return w.svc.Update(t.Context(), req, testActor())
}

func TestUpdateStartsTheRolloutWithACapableCandidate(t *testing.T) {
	t.Parallel()

	w := newWorld()
	initial, err := w.svc.Get(t.Context(), testTenant())
	require.NoError(t, err)
	assert.False(t, initial.Enabled)
	assert.Equal(t, extractionrollout.DefaultPercent, initial.Percent)

	saved, err := w.update(t, nil)
	require.NoError(t, err)
	assert.Equal(t, w.candidate.ID, saved.CandidateID())
	require.NotNil(t, saved.StartedAt)
	assert.Equal(t, testNow, *saved.StartedAt)
	assert.Equal(t, testUser, *saved.UpdatedByID)
}

func TestUpdateRefusesACandidateThatCannotExtract(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.candidate.Enabled = false
	_, err := w.update(t, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot run document extraction")

	_, err = w.update(t, func(req *services.UpdateExtractionRolloutRequest) {
		req.ProviderID = pulid.MustNew("aip_")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
	assert.Zero(t, w.rollouts.saves)
}

func TestUpdateRefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	w := newWorld()
	_, err := w.update(t, nil)
	require.NoError(t, err)
	saved, err := w.update(t, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.Version)

	_, err = w.update(t, func(req *services.UpdateExtractionRolloutRequest) { req.Version = 0 })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Someone else changed the rollout")
}

func TestUpdateRefusesOutOfRangeSettings(t *testing.T) {
	t.Parallel()

	w := newWorld()
	_, err := w.update(t, func(req *services.UpdateExtractionRolloutRequest) { req.Percent = 0 })
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "percent", multiErr.Errors[0].Field)
}

func TestAssignSendsTheCandidatesShareToTheCandidate(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)

	candidateSide, controlSide := 0, 0
	for range 400 {
		document := pulid.MustNew("doc_")
		preferred, err := w.svc.AssignExtraction(t.Context(), &services.AssignExtractionRolloutRequest{
			TenantInfo:  testTenant(),
			DocumentID:  document,
			ExtractedAt: testNow,
		})
		require.NoError(t, err)
		if preferred.IsNil() {
			controlSide++
			continue
		}
		assert.Equal(t, w.candidate.ID, preferred)
		candidateSide++
	}

	assert.InDelta(t, 200, candidateSide, 60)
	assert.Equal(t, 400, candidateSide+controlSide)
	assert.Len(t, w.assignments.items, 400, "both sides are recorded for comparison")
}

func TestAssignIsStableAcrossRetries(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(100)
	req := &services.AssignExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: testNow,
	}

	first, err := w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)
	second, err := w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, w.candidate.ID, first)
	assert.Equal(t, first, second)
	assert.Len(t, w.assignments.items, 1)
}

func TestAssignLeavesProductionAloneWhenTheRolloutIsOff(t *testing.T) {
	t.Parallel()

	w := newWorld()
	req := &services.AssignExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: testNow,
	}
	preferred, err := w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, preferred.IsNil())

	w.start(100)
	w.rollouts.rollout.Halt(extractionrollout.Breach{Reason: extractionrollout.HaltReasonRejections}, testNow)
	preferred, err = w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, preferred.IsNil(), "a stopped rollout serves nothing")
	assert.Empty(t, w.assignments.items)
}

func TestAssignDoesNotPreferACandidateThatWasReplaced(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(100)
	req := &services.AssignExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: testNow,
	}
	_, err := w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)

	next := w.production.ID
	w.rollouts.rollout.ProviderID = &next
	preferred, err := w.svc.AssignExtraction(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, preferred.IsNil())
}

func TestSettleRecordsTheOutcomeOnce(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(100)
	document := pulid.MustNew("doc_")
	_, err := w.svc.AssignExtraction(t.Context(), &services.AssignExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  document,
		ExtractedAt: testNow,
	})
	require.NoError(t, err)

	settle := &services.SettleExtractionRolloutRequest{
		TenantInfo:       testTenant(),
		DocumentID:       document,
		ExtractedAt:      testNow,
		Outcome:          extractionrollout.OutcomeAccepted,
		ServedProviderID: w.candidate.ID,
		ServedModel:      "trenova-extract",
	}
	require.NoError(t, w.svc.SettleExtraction(t.Context(), settle))
	require.NoError(t, w.svc.SettleExtraction(t.Context(), settle))

	assignment := w.assignments.items[0]
	assert.Equal(t, extractionrollout.OutcomeAccepted, assignment.Outcome)
	assert.True(t, assignment.ServedByCandidate())
	assert.Equal(t, "trenova-extract", assignment.ServedModel)
	assert.Equal(t, int64(1), assignment.Version, "a repeated settle writes nothing")
}

func TestSettleIgnoresAnExtractionOutsideTheRollout(t *testing.T) {
	t.Parallel()

	w := newWorld()
	require.NoError(t, w.svc.SettleExtraction(t.Context(), &services.SettleExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: testNow,
		Outcome:     extractionrollout.OutcomeAccepted,
	}))

	err := w.svc.SettleExtraction(t.Context(), &services.SettleExtractionRolloutRequest{
		TenantInfo: testTenant(),
		Outcome:    extractionrollout.OutcomePending,
	})
	require.Error(t, err)
}

func TestTooManyRejectedAnswersStopTheRollout(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeAccepted, 45)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeRejected, 5)
	w.settled(extractionrollout.ArmCandidate, w.candidate.ID, extractionrollout.OutcomeAccepted, 20)
	w.settled(extractionrollout.ArmCandidate, w.candidate.ID, extractionrollout.OutcomeRejected, 9)

	document := pulid.MustNew("doc_")
	w.assignments.items = append(w.assignments.items, &extractionrollout.RolloutAssignment{
		OrganizationID:      testOrg,
		BusinessUnitID:      testBU,
		DocumentID:          document,
		ExtractedAt:         testNow,
		Arm:                 extractionrollout.ArmCandidate,
		CandidateProviderID: w.candidate.ID,
		Outcome:             extractionrollout.OutcomePending,
		CreatedAt:           testNow,
	})
	require.NoError(t, w.svc.SettleExtraction(t.Context(), &services.SettleExtractionRolloutRequest{
		TenantInfo:       testTenant(),
		DocumentID:       document,
		ExtractedAt:      testNow,
		Outcome:          extractionrollout.OutcomeFailed,
		ServedProviderID: w.candidate.ID,
	}))

	rollout := w.rollouts.rollout
	require.True(t, rollout.IsHalted())
	assert.Equal(t, extractionrollout.HaltReasonRejections, rollout.HaltReason)
	assert.InDelta(t, 10.0/30, rollout.HaltCandidateRate, 0.0001)
	assert.InDelta(t, 0.1, rollout.HaltBaselineRate, 0.0001)
	assert.True(t, rollout.Enabled, "the setting stays on so the reason stays visible")

	require.Len(t, w.notifier.sent, 1)
	notice := w.notifier.sent[0].Notification
	assert.Equal(t, services.ExtractionRolloutHaltedEvent, notice.EventType)
	assert.Contains(t, notice.Message, "33.3%")
	assert.Contains(t, notice.Message, "10.0%")

	preferred, err := w.svc.AssignExtraction(t.Context(), &services.AssignExtractionRolloutRequest{
		TenantInfo:  testTenant(),
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: testNow,
	})
	require.NoError(t, err)
	assert.True(t, preferred.IsNil(), "everything goes back to production")
}

func TestFallbacksDoNotCountAgainstTheCandidate(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeAccepted, 50)
	w.settled(extractionrollout.ArmCandidate, w.candidate.ID, extractionrollout.OutcomeAccepted, 30)
	w.settled(extractionrollout.ArmCandidate, w.production.ID, extractionrollout.OutcomeRejected, 30)

	require.NoError(t, w.svc.enforceGuards(t.Context(), testTenant()))
	assert.False(t, w.rollouts.rollout.IsHalted())

	report, err := w.svc.Report(t.Context(), testTenant())
	require.NoError(t, err)
	assert.Equal(t, 30, report.Candidate.FellBack)
	assert.InDelta(t, 0, report.Candidate.RejectionRate, 0.0001)
}

func TestAnAccuracyDropStopsTheRollout(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.correction(w.production.ID, 300, 285)
	correction := w.correction(w.candidate.ID, 250, 200)

	require.NoError(t, w.svc.ObserveCorrection(t.Context(), correction))

	rollout := w.rollouts.rollout
	require.True(t, rollout.IsHalted())
	assert.Equal(t, extractionrollout.HaltReasonAccuracyDrop, rollout.HaltReason)
	assert.InDelta(t, 0.8, rollout.HaltCandidateRate, 0.0001)
	assert.InDelta(t, 0.95, rollout.HaltBaselineRate, 0.0001)
	require.Len(t, w.notifier.sent, 1)
	assert.Contains(t, w.notifier.sent[0].Notification.Title, "accuracy dropped")
}

func TestAccuracyWithinTheAllowanceKeepsServing(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.correction(w.production.ID, 300, 285)
	correction := w.correction(w.candidate.ID, 250, 230)

	require.NoError(t, w.svc.ObserveCorrection(t.Context(), correction))
	assert.False(t, w.rollouts.rollout.IsHalted())
	assert.Empty(t, w.notifier.sent)
}

func TestHaltRetriesAStaleSave(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.rollouts.staleSaves = 1
	w.correction(w.production.ID, 300, 285)
	correction := w.correction(w.candidate.ID, 250, 200)

	require.NoError(t, w.svc.ObserveCorrection(t.Context(), correction))
	assert.True(t, w.rollouts.rollout.IsHalted())
}

func TestHaltStandsDownWhenSomeoneStoppedTheRolloutFirst(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.rollouts.staleSaves = 1
	w.rollouts.onStaleSave = func(store *rolloutStore) {
		store.rollout.Enabled = false
		store.rollout.Version++
	}
	w.correction(w.production.ID, 300, 285)
	correction := w.correction(w.candidate.ID, 250, 200)

	require.NoError(t, w.svc.ObserveCorrection(t.Context(), correction))
	assert.False(t, w.rollouts.rollout.IsHalted())
	assert.Empty(t, w.notifier.sent)
}

func TestStartingAgainAfterAHaltMeasuresFromNow(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.rollouts.rollout.Halt(extractionrollout.Breach{Reason: extractionrollout.HaltReasonAccuracyDrop}, testNow)

	saved, err := w.update(t, func(req *services.UpdateExtractionRolloutRequest) { req.Percent = 50 })
	require.NoError(t, err)
	assert.False(t, saved.IsHalted())
	assert.Equal(t, testNow, *saved.StartedAt)
	assert.Equal(t, w.candidate.ID, saved.CandidateID())
}

func TestReportComparesBothSides(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeAccepted, 8)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeRejected, 2)
	w.settled(extractionrollout.ArmControl, pulid.Nil, extractionrollout.OutcomePending, 1)
	w.settled(extractionrollout.ArmCandidate, w.candidate.ID, extractionrollout.OutcomeAccepted, 6)
	w.settled(extractionrollout.ArmCandidate, w.candidate.ID, extractionrollout.OutcomeFailed, 1)
	w.settled(extractionrollout.ArmCandidate, w.production.ID, extractionrollout.OutcomeAccepted, 2)
	w.settled(extractionrollout.ArmCandidate, pulid.Nil, extractionrollout.OutcomeSuperseded, 1)
	w.correction(w.production.ID, 10, 8)
	w.correction(w.candidate.ID, 10, 9)

	report, err := w.svc.Report(t.Context(), testTenant())
	require.NoError(t, err)

	assert.Equal(t, "Fine-tuned Qwen", report.ProviderName)
	assert.Equal(t, 11, report.Control.Assigned)
	assert.Equal(t, 1, report.Control.Pending)
	assert.InDelta(t, 0.2, report.Control.RejectionRate, 0.0001)
	assert.Equal(t, 10, report.Candidate.Assigned)
	assert.Equal(t, 8, report.Candidate.Accepted)
	assert.Equal(t, 1, report.Candidate.Failed)
	assert.Equal(t, 1, report.Candidate.Superseded)
	assert.Equal(t, 2, report.Candidate.FellBack)
	assert.InDelta(t, 1.0/7, report.Candidate.RejectionRate, 0.0001)
	assert.InDelta(t, 0.9, report.CandidateAccuracy.Accuracy, 0.0001)
	assert.InDelta(t, 0.8, report.ProductionAccuracy.Accuracy, 0.0001)
	require.Len(t, report.Fields, 1)
	assert.Equal(t, "rate", report.Fields[0].Key)
	assert.InDelta(t, 0.9, report.Fields[0].CandidateAccuracy, 0.0001)
	assert.InDelta(t, 0.8, report.Fields[0].ProductionAccuracy, 0.0001)
}

func TestReportWithoutARolloutIsEmpty(t *testing.T) {
	t.Parallel()

	w := newWorld()
	report, err := w.svc.Report(t.Context(), testTenant())
	require.NoError(t, err)
	assert.False(t, report.Rollout.Enabled)
	assert.Zero(t, report.Candidate.Assigned)
	assert.Empty(t, report.Fields)
}

func TestPurgeRemovesExpiredAssignments(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.start(50)
	w.settled(extractionrollout.ArmControl, w.production.ID, extractionrollout.OutcomeAccepted, 3)
	w.assignments.items[0].CreatedAt = 1

	purged, err := w.svc.PurgeExpiredAssignments(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: testTenant(),
		Now:        testNow,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), purged)
	assert.Len(t, w.assignments.items, 2)
}
