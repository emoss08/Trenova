package extractionshadowservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const extractedAt = testNow - 600

func draftData(rate, reference string) map[string]any {
	return map[string]any{
		"kind": "RateConfirmation",
		"fields": map[string]any{
			"rate":            map[string]any{"value": rate, "source": "ai", "confidence": 0.9},
			"referenceNumber": map[string]any{"value": reference, "source": "ai", "confidence": 0.8},
		},
		"stops": []any{},
	}
}

func confirmed() *aicorrection.Snapshot {
	return &aicorrection.Snapshot{
		Fields: map[string]string{
			aicorrection.FieldRate:      "2563.12",
			aicorrection.FieldReference: "OUG-8393964",
		},
		Stops: []aicorrection.StopSnapshot{},
	}
}

func (w *world) correction(documentID pulid.ID, capturedAt int64, predicted map[string]any) *aicorrection.Correction {
	prediction := aicorrection.ReadPrediction(predicted)
	correction := &aicorrection.Correction{
		ID:             pulid.MustNew("aicr_"),
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		Task:           aicorrection.TaskShipmentDraftExtraction,
		DocumentID:     &documentID,
		Predicted:      prediction.Snapshot,
		Confirmed:      confirmed(),
		FieldResults:   aicorrection.Score(prediction, confirmed()),
		CapturedAt:     capturedAt,
	}
	correction.ApplyTally()
	w.corrections.items = append(w.corrections.items, correction)
	return correction
}

func (w *world) pending(documentID pulid.ID) *extractionshadow.ShadowResult {
	result := &extractionshadow.ShadowResult{
		ID:             pulid.MustNew("exsr_"),
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		DocumentID:     documentID,
		ExtractedAt:    extractedAt,
		Status:         extractionshadow.ResultStatusPending,
		ProviderID:     w.candidate.ID,
		ProviderName:   w.candidate.Name,
		CreatedAt:      testNow,
	}
	w.results.items = append(w.results.items, result)
	return result
}

func TestConsiderExtractionDoesNothingUntilAProviderIsChosen(t *testing.T) {
	t.Parallel()

	w := newWorld()
	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:  testTenant(),
		DocumentID:  w.document(extractedAt),
		ExtractedAt: extractedAt,
	})
	require.NoError(t, err)
	assert.False(t, decision.Sampled)
	assert.Equal(t, ReasonDisabled, decision.Reason)
	assert.Empty(t, w.results.items)
	assert.Empty(t, w.starter.started)
}

func TestConsiderExtractionSamplesAndStartsTheShadow(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)
	documentID := w.document(extractedAt)

	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           documentID,
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.production.ID,
		ProductionModel:      "frontier-large",
	})
	require.NoError(t, err)
	require.True(t, decision.Sampled)
	require.Len(t, w.results.items, 1)
	require.Len(t, w.starter.started, 1)

	stored := w.results.items[0]
	assert.Equal(t, decision.ResultID, stored.ID)
	assert.Equal(t, extractionshadow.ResultStatusPending, stored.Status)
	assert.Equal(t, w.candidate.ID, stored.ProviderID)
	assert.Equal(t, "Fine-tuned Qwen", stored.ProviderName)
	assert.Equal(t, w.production.ID, *stored.ProductionProviderID)
	assert.Equal(t, "frontier-large", stored.ProductionModel)
	assert.Equal(t, extractionshadow.WorkflowID(stored.ID), stored.WorkflowID)
	assert.Equal(t, decision.WorkflowID, stored.WorkflowID)
}

func TestConsiderExtractionIsIdempotentForOneExtraction(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)
	documentID := w.document(extractedAt)

	req := &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           documentID,
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.production.ID,
	}
	first, err := w.svc.ConsiderExtraction(t.Context(), req)
	require.NoError(t, err)
	second, err := w.svc.ConsiderExtraction(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, first.ResultID, second.ResultID)
	assert.Equal(t, ReasonAlreadyExists, second.Reason)
	assert.Len(t, w.results.items, 1)
	assert.Len(t, w.starter.started, 1, "a retried activity does not start a second workflow")
}

func TestConsiderExtractionRestartsARowWhoseWorkflowNeverStarted(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)
	documentID := w.document(extractedAt)
	w.pending(documentID)

	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           documentID,
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.production.ID,
	})
	require.NoError(t, err)
	assert.True(t, decision.Sampled)
	assert.Len(t, w.starter.started, 1)
	assert.NotEmpty(t, w.results.items[0].WorkflowID)
}

func TestConsiderExtractionSkipsWhenProductionAlreadyUsedTheCandidate(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)

	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           w.document(extractedAt),
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.candidate.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, ReasonSameProvider, decision.Reason)
	assert.Empty(t, w.results.items)
}

func TestConsiderExtractionFollowsTheSample(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(10)

	sampled, skipped := 0, 0
	for range 200 {
		documentID := w.document(extractedAt)
		decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
			TenantInfo:           testTenant(),
			DocumentID:           documentID,
			ExtractedAt:          extractedAt,
			ProductionProviderID: w.production.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, extractionshadow.Sampled(documentID, extractedAt, 10), decision.Sampled)
		if decision.Sampled {
			sampled++
		} else {
			assert.Equal(t, ReasonNotSampled, decision.Reason)
			skipped++
		}
	}
	assert.Equal(t, sampled, len(w.results.items))
	assert.Positive(t, skipped)
}

func TestConsiderExtractionStopsAtTheDailyLimit(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)
	w.settings.settings.DailyLimit = 2

	reasons := make([]string, 0, 3)
	for range 3 {
		decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
			TenantInfo:           testTenant(),
			DocumentID:           w.document(extractedAt),
			ExtractedAt:          extractedAt,
			ProductionProviderID: w.production.ID,
		})
		require.NoError(t, err)
		reasons = append(reasons, decision.Reason)
	}

	assert.Equal(t, []string{"", "", ReasonDailyLimit}, reasons)
	assert.Len(t, w.results.items, 2)
}

func TestConsiderExtractionIgnoresASupersededExtraction(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)

	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           w.document(extractedAt + 60),
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.production.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, ReasonSuperseded, decision.Reason)
	assert.Empty(t, w.results.items)
}

func TestConsiderExtractionRecordsAWorkflowThatCouldNotStart(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)
	w.starter.err = errUpstream

	decision, err := w.svc.ConsiderExtraction(t.Context(), &services.ConsiderExtractionShadowRequest{
		TenantInfo:           testTenant(),
		DocumentID:           w.document(extractedAt),
		ExtractedAt:          extractedAt,
		ProductionProviderID: w.production.ID,
	})
	require.NoError(t, err)
	assert.True(t, decision.Sampled)
	require.Len(t, w.results.items, 1)
	assert.Equal(t, extractionshadow.ResultStatusFailed, w.results.items[0].Status)
	assert.Equal(t, startFailedReason, w.results.items[0].StatusReason)
}

func TestRunShadowStoresTheWouldBeDraft(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := w.pending(documentID)
	cost := decimal.RequireFromString("0.0042")
	w.predictor.prediction = &services.ShadowDraftPrediction{
		DraftData:    draftData("2563.12", "OUG-8393964"),
		Accepted:     true,
		Model:        "trenova-extract-v3",
		ProviderID:   w.candidate.ID,
		InputTokens:  1200,
		OutputTokens: 650,
		LatencyMs:    2400,
		CostUSD:      &cost,
	}

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	}))

	stored := w.results.find(result.ID)
	assert.Equal(t, extractionshadow.ResultStatusCompleted, stored.Status)
	assert.True(t, stored.Accepted)
	assert.Equal(t, "trenova-extract-v3", stored.ServedModel)
	assert.Equal(t, "2563.12", stored.Predicted.Fields[aicorrection.FieldRate])
	assert.Equal(t, int64(2400), stored.LatencyMs)
	assert.Equal(t, int64(1200), stored.InputTokens)
	assert.True(t, stored.CostUSD.Equal(cost))
	assert.False(t, stored.IsScored(), "nothing is scored before a person confirms the document")

	require.Len(t, w.predictor.requests, 1)
	assert.Equal(t, w.candidate.ID, w.predictor.requests[0].ProviderID)
	assert.Equal(t, extractedAt, w.predictor.requests[0].ExtractedAt)
}

func TestRunShadowScoresAtOnceWhenThePersonConfirmedFirst(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := w.pending(documentID)
	correction := w.correction(documentID, extractedAt+120, draftData("2500.00", "OUG-8393964"))
	w.predictor.prediction = &services.ShadowDraftPrediction{
		DraftData: draftData("2563.12", "OUG-8393964"),
		Accepted:  true,
	}

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	}))

	stored := w.results.find(result.ID)
	require.True(t, stored.IsScored())
	assert.Equal(t, correction.ID, *stored.CorrectionID)
	assert.Equal(t, 2, stored.CorrectCount)
	assert.Equal(t, 1, stored.BaselineCorrectCount)
	assert.Equal(t, 1, stored.BaselineCorrectedCount)
	assert.Equal(t, extractionshadow.VerdictBetter, stored.Verdict)
}

func TestRunShadowDoesNotScoreAgainstAnEarlierConfirmation(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := w.pending(documentID)
	w.correction(documentID, extractedAt-3600, draftData("2500.00", "OUG-8393964"))
	w.predictor.prediction = &services.ShadowDraftPrediction{DraftData: draftData("2563.12", "X")}

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	}))

	stored := w.results.find(result.ID)
	assert.Equal(t, extractionshadow.ResultStatusCompleted, stored.Status)
	assert.False(t, stored.IsScored(), "a shipment confirmed before this extraction ran confirmed another draft")
}

func TestRunShadowSkipsWhenTheBudgetIsSpent(t *testing.T) {
	t.Parallel()

	w := newWorld()
	result := w.pending(w.document(extractedAt))
	w.budget.decision = &agentquality.BudgetDecision{Stop: true, Reason: "the nightly cap is reached"}

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	}))

	stored := w.results.find(result.ID)
	assert.Equal(t, extractionshadow.ResultStatusSkipped, stored.Status)
	assert.Equal(t, "The evaluation budget is spent: the nightly cap is reached", stored.StatusReason)
	assert.Empty(t, w.predictor.requests, "no model call is made past the budget")
}

func TestRunShadowSkipsASupersededOrDeletedDocument(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err    error
		reason string
	}{
		"superseded": {err: services.ErrShadowSuperseded, reason: supersededReason},
		"deleted":    {err: errortypes.NewNotFoundError("document not found"), reason: deletedReason},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			w := newWorld()
			result := w.pending(w.document(extractedAt))
			w.predictor.err = tc.err

			require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
				TenantInfo: testTenant(),
				ResultID:   result.ID,
			}))

			stored := w.results.find(result.ID)
			assert.Equal(t, extractionshadow.ResultStatusSkipped, stored.Status)
			assert.Equal(t, tc.reason, stored.StatusReason)
		})
	}
}

func TestRunShadowRetriesATransientFailureThenRecordsIt(t *testing.T) {
	t.Parallel()

	w := newWorld()
	result := w.pending(w.document(extractedAt))
	w.predictor.err = errUpstream

	err := w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	})
	require.ErrorIs(t, err, errUpstream)
	assert.Equal(t, extractionshadow.ResultStatusPending, w.results.find(result.ID).Status)

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo:   testTenant(),
		ResultID:     result.ID,
		FinalAttempt: true,
	}))
	stored := w.results.find(result.ID)
	assert.Equal(t, extractionshadow.ResultStatusFailed, stored.Status)
	assert.Equal(t, "The model could not be reached after several attempts", stored.StatusReason)
}

func TestRunShadowLeavesASettledResultAlone(t *testing.T) {
	t.Parallel()

	w := newWorld()
	result := w.pending(w.document(extractedAt))
	result.Status = extractionshadow.ResultStatusSkipped

	require.NoError(t, w.svc.RunShadow(t.Context(), &services.RunExtractionShadowRequest{
		TenantInfo: testTenant(),
		ResultID:   result.ID,
	}))
	assert.Empty(t, w.predictor.requests)
}

func TestFailShadowSettlesOnlyAPendingResult(t *testing.T) {
	t.Parallel()

	w := newWorld()
	result := w.pending(w.document(extractedAt))
	req := repositories.GetExtractionShadowResultRequest{TenantInfo: testTenant(), ID: result.ID}

	require.NoError(t, w.svc.FailShadow(t.Context(), req, "boom"))
	assert.Equal(t, extractionshadow.ResultStatusFailed, w.results.find(result.ID).Status)

	require.NoError(t, w.svc.FailShadow(t.Context(), req, "again"))
	assert.Equal(t, "boom", w.results.find(result.ID).StatusReason)
}

func completed(w *world, documentID pulid.ID, draft map[string]any) *extractionshadow.ShadowResult {
	result := w.pending(documentID)
	result.Status = extractionshadow.ResultStatusCompleted
	result.DraftData = draft
	result.Predicted = aicorrection.ReadPrediction(draft).Snapshot
	return result
}

func TestScoreCorrectionScoresTheShadowOfTheConfirmedExtraction(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := completed(w, documentID, draftData("2563.12", "WRONG"))
	correction := w.correction(documentID, testNow, draftData("2563.12", "OUG-8393964"))

	require.NoError(t, w.scorer.ScoreCorrection(t.Context(), correction))

	stored := w.results.find(result.ID)
	require.True(t, stored.IsScored())
	assert.Equal(t, 1, stored.CorrectCount)
	assert.Equal(t, 1, stored.CorrectedCount)
	assert.Equal(t, 2, stored.BaselineCorrectCount)
	assert.Equal(t, extractionshadow.VerdictWorse, stored.Verdict)
	assert.Equal(t, testNow, *stored.ScoredAt)
}

func TestScoreCorrectionLeavesAnUnfinishedShadowForItsCompletion(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := w.pending(documentID)
	correction := w.correction(documentID, testNow, draftData("2563.12", "OUG-8393964"))

	require.NoError(t, w.scorer.ScoreCorrection(t.Context(), correction))
	assert.False(t, w.results.find(result.ID).IsScored())
}

func TestScoreCorrectionRetriesAConcurrentWrite(t *testing.T) {
	t.Parallel()

	w := newWorld()
	documentID := w.document(extractedAt)
	result := completed(w, documentID, draftData("2563.12", "OUG-8393964"))
	correction := w.correction(documentID, testNow, draftData("2563.12", "OUG-8393964"))
	w.results.conflicts = 1

	require.NoError(t, w.scorer.ScoreCorrection(t.Context(), correction))
	assert.True(t, w.results.find(result.ID).IsScored())
}

func TestScoreCorrectionIgnoresCorrectionsWithoutADocument(t *testing.T) {
	t.Parallel()

	w := newWorld()
	require.NoError(t, w.scorer.ScoreCorrection(t.Context(), nil))
	require.NoError(t, w.scorer.ScoreCorrection(t.Context(), &aicorrection.Correction{
		Task: aicorrection.TaskShipmentDraftExtraction,
	}))
}

func TestReportComparesBothSidesOnTheSameDocuments(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(100)

	better := w.document(extractedAt)
	completed(w, better, draftData("2563.12", "OUG-8393964")).LatencyMs = 1000
	require.NoError(t, w.scorer.ScoreCorrection(t.Context(),
		w.correction(better, testNow, draftData("2500.00", "OUG-8393964"))))

	worse := w.document(extractedAt)
	completed(w, worse, draftData("1.00", "OUG-8393964")).LatencyMs = 3000
	require.NoError(t, w.scorer.ScoreCorrection(t.Context(),
		w.correction(worse, testNow, draftData("2563.12", "OUG-8393964"))))

	failed := w.pending(w.document(extractedAt))
	failed.Status = extractionshadow.ResultStatusFailed
	w.pending(w.document(extractedAt))

	report, err := w.svc.Report(t.Context(), &services.ExtractionShadowReportRequest{
		TenantInfo: testTenant(),
		WindowDays: 7,
	})
	require.NoError(t, err)

	assert.Equal(t, w.candidate.ID, *report.ProviderID)
	assert.Equal(t, "Fine-tuned Qwen", report.ProviderName)
	assert.Equal(t, 4, report.Sampled)
	assert.Equal(t, 2, report.Completed)
	assert.Equal(t, 1, report.Failed)
	assert.Equal(t, 1, report.Pending)
	assert.Equal(t, 2, report.Scored)
	assert.Equal(t, 1, report.Better)
	assert.Equal(t, 1, report.Worse)
	assert.Equal(t, int64(2000), report.AvgLatencyMs)
	assert.Equal(t, 4, report.Candidate.Scored)
	assert.Equal(t, 3, report.Candidate.Correct)
	assert.Equal(t, 4, report.Production.Scored)
	assert.Equal(t, 3, report.Production.Correct)
	assert.Equal(t, testNow-7*timeutils.SecondsPerDay, report.Since)

	require.NotEmpty(t, report.Fields)
	for _, field := range report.Fields {
		assert.Equal(t, 2, field.CandidateScored, field.Key)
		assert.Equal(t, 2, field.ProductionScored, field.Key)
	}
}

func TestReportLogsAProviderLookupFailure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		err    error
		logged int
	}{
		{name: "database failure", err: errors.New("connection refused"), logged: 1},
		{name: "provider removed", err: errortypes.NewNotFoundError("provider not found")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := newWorld()
			w.enable(100)
			core, logs := observer.New(zap.WarnLevel)
			w.svc.l = zap.New(core)
			w.providers.err = tc.err

			report, err := w.svc.Report(
				t.Context(),
				&services.ExtractionShadowReportRequest{TenantInfo: testTenant()},
			)
			require.NoError(t, err)
			assert.Empty(t, report.ProviderName)
			assert.Equal(t, tc.logged, logs.FilterMessage("failed to read shadow provider name").Len())
		})
	}
}

func TestReportWithoutACandidateIsEmpty(t *testing.T) {
	t.Parallel()

	w := newWorld()
	report, err := w.svc.Report(t.Context(), &services.ExtractionShadowReportRequest{TenantInfo: testTenant()})
	require.NoError(t, err)
	assert.Nil(t, report.ProviderID)
	assert.Equal(t, defaultReportWindowDays, report.WindowDays)
	assert.Zero(t, report.Sampled)

	_, err = w.svc.Report(t.Context(), &services.ExtractionShadowReportRequest{
		TenantInfo: testTenant(),
		WindowDays: 400,
	})
	require.Error(t, err)
}

func TestUpdateSettingsValidatesTheCandidate(t *testing.T) {
	t.Parallel()

	w := newWorld()
	settings, err := w.svc.GetSettings(t.Context(), testTenant())
	require.NoError(t, err)
	assert.False(t, settings.Enabled)
	assert.Equal(t, extractionshadow.DefaultSamplePercent, settings.SamplePercent)

	w.candidate.Enabled = false
	_, err = w.svc.UpdateSettings(t.Context(), &services.UpdateExtractionShadowSettingsRequest{
		TenantInfo:    testTenant(),
		Enabled:       true,
		ProviderID:    w.candidate.ID,
		SamplePercent: 25,
		DailyLimit:    100,
	}, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot run document extraction")
	assert.Zero(t, w.settings.saves)

	w.candidate.Enabled = true
	saved, err := w.svc.UpdateSettings(t.Context(), &services.UpdateExtractionShadowSettingsRequest{
		TenantInfo:    testTenant(),
		Enabled:       true,
		ProviderID:    w.candidate.ID,
		SamplePercent: 25,
		DailyLimit:    100,
	}, testActor())
	require.NoError(t, err)
	assert.True(t, saved.Enabled)
	assert.Equal(t, 25, saved.SamplePercent)
	assert.Equal(t, testUser, *saved.UpdatedByID)
	assert.Equal(t, w.candidate.ID, saved.CandidateID())
}

func TestUpdateSettingsRefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(10)
	w.settings.settings.Version = 3

	_, err := w.svc.UpdateSettings(t.Context(), &services.UpdateExtractionShadowSettingsRequest{
		TenantInfo:    testTenant(),
		SamplePercent: 10,
		DailyLimit:    10,
		Version:       2,
	}, testActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestUpdateSettingsCanTurnTheShadowOffWithoutAProvider(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.enable(10)

	saved, err := w.svc.UpdateSettings(t.Context(), &services.UpdateExtractionShadowSettingsRequest{
		TenantInfo:    testTenant(),
		Enabled:       false,
		SamplePercent: 10,
		DailyLimit:    10,
	}, testActor())
	require.NoError(t, err)
	assert.False(t, saved.Enabled)
	assert.True(t, saved.CandidateID().IsNil())
}

func TestPurgeExpiredShadowsFollowsCorrectionRetention(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.retention.settings = &tenant.DataRetention{AICorrectionRetentionPeriod: 30}
	old := w.pending(w.document(extractedAt))
	old.CreatedAt = testNow - 31*timeutils.SecondsPerDay
	recent := w.pending(w.document(extractedAt))

	purged, err := w.svc.PurgeExpiredShadows(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: testTenant(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), purged)
	require.Len(t, w.results.items, 1)
	assert.Equal(t, recent.ID, w.results.items[0].ID)
}
