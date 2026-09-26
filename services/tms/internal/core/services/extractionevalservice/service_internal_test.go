package extractionevalservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func confirmedSnapshot() *aicorrection.Snapshot {
	return &aicorrection.Snapshot{
		Fields: map[string]string{
			aicorrection.FieldReference: "LD-10442",
			aicorrection.FieldRate:      "2450.00",
		},
		Stops: []aicorrection.StopSnapshot{
			{
				Role:                 aicorrection.RolePickup,
				Name:                 "Northwind Foods",
				City:                 "Dallas",
				ScheduledWindowStart: time.Date(2026, time.March, 15, 14, 0, 0, 0, time.UTC).Unix(),
			},
		},
	}
}

func (w *world) seedCorrection() *aicorrection.Correction {
	documentID := pulid.MustNew("doc_")
	w.documents.docs[documentID] = &document.Document{ID: documentID, OriginalName: "ratecon-4471.pdf"}
	w.contents.pages = []*documentcontent.Page{
		{PageNumber: 1, ExtractedText: "Load LD-10442 Rate $2,450"},
		{PageNumber: 2, ExtractedText: "   "},
		{PageNumber: 3, ExtractedText: "Pickup Northwind Foods, Dallas TX"},
	}
	correction := &aicorrection.Correction{
		ID:                  pulid.MustNew("aicr_"),
		OrganizationID:      testOrg,
		BusinessUnitID:      testBU,
		Task:                aicorrection.TaskShipmentDraftExtraction,
		DocumentID:          &documentID,
		DocumentKind:        "RateConfirmation",
		DocumentFingerprint: "Acme Logistics",
		Confirmed:           confirmedSnapshot(),
	}
	w.corrections.items[correction.ID] = correction
	return correction
}

func (w *world) seedActiveCase(t *testing.T) *extractioneval.ExtractionCase {
	t.Helper()
	correction := w.seedCorrection()
	created, err := w.svc.PromoteCorrection(t.Context(), &services.PromoteCorrectionRequest{
		TenantInfo:   testTenant(),
		CorrectionID: correction.ID,
		Activate:     true,
	}, testActor())
	require.NoError(t, err)
	return created
}

func (w *world) seedProvider() *aiprovider.Provider {
	provider := &aiprovider.Provider{
		ID:      pulid.MustNew("aiprv_"),
		Name:    "Tuned extractor",
		Model:   "trenova-extract-8b",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}
	w.providers.providers[provider.ID] = provider
	return provider
}

func TestPromoteCorrection_FreezesTheDocumentAndConfirmedValues(t *testing.T) {
	t.Parallel()

	w := newWorld()
	correction := w.seedCorrection()

	created, err := w.svc.PromoteCorrection(t.Context(), &services.PromoteCorrectionRequest{
		TenantInfo:   testTenant(),
		CorrectionID: correction.ID,
	}, testActor())
	require.NoError(t, err)

	assert.Equal(t, extractioneval.CaseStatusCandidate, created.Status)
	assert.Equal(t, "Rate Confirmation · ratecon-4471.pdf", created.Title)
	assert.Equal(t, "ratecon-4471.pdf", created.FileName)
	require.Len(t, created.Pages, 2, "a page with no text is not frozen")
	assert.Equal(t, 3, created.Pages[1].Number)
	assert.Equal(t, 2, created.PageCount)
	assert.Equal(t, correction.Confirmed, created.Expected)
	assert.Equal(t, 2, created.ExpectedFieldCount)
	assert.Len(t, created.InputHash, extractioneval.InputHashLength)
	require.NotNil(t, created.SourceCorrectionID)
	assert.Equal(t, correction.ID, *created.SourceCorrectionID)
	assert.Equal(t, testUser, created.CreatedByID)
	assert.Equal(t, "Acme Logistics", created.DocumentFingerprint)
}

func TestPromoteCorrection_RefusesTwiceAndWithoutText(t *testing.T) {
	t.Parallel()

	w := newWorld()
	correction := w.seedCorrection()
	request := &services.PromoteCorrectionRequest{TenantInfo: testTenant(), CorrectionID: correction.ID}

	_, err := w.svc.PromoteCorrection(t.Context(), request, testActor())
	require.NoError(t, err)
	_, err = w.svc.PromoteCorrection(t.Context(), request, testActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	other := newWorld()
	second := other.seedCorrection()
	other.contents.pages = nil
	_, err = other.svc.PromoteCorrection(t.Context(), &services.PromoteCorrectionRequest{
		TenantInfo:   testTenant(),
		CorrectionID: second.ID,
	}, testActor())
	require.Error(t, err)
	assert.Empty(t, other.cases.items)

	deleted := newWorld()
	third := deleted.seedCorrection()
	clear(deleted.documents.docs)
	_, err = deleted.svc.PromoteCorrection(t.Context(), &services.PromoteCorrectionRequest{
		TenantInfo:   testTenant(),
		CorrectionID: third.ID,
	}, testActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestUpdateCase_FollowsTheStatusLifecycle(t *testing.T) {
	t.Parallel()

	w := newWorld()
	created := w.seedActiveCase(t)

	retired := extractioneval.CaseStatusRetired
	updated, err := w.svc.UpdateCase(t.Context(), &services.UpdateExtractionCaseRequest{
		TenantInfo: testTenant(),
		ID:         created.ID,
		Version:    created.Version,
		Status:     &retired,
		Notes:      new(string),
	}, testActor())
	require.NoError(t, err)
	assert.Equal(t, extractioneval.CaseStatusRetired, updated.Status)

	candidate := extractioneval.CaseStatusCandidate
	_, err = w.svc.UpdateCase(t.Context(), &services.UpdateExtractionCaseRequest{
		TenantInfo: testTenant(),
		ID:         created.ID,
		Version:    updated.Version,
		Status:     &candidate,
	}, testActor())
	require.Error(t, err, "a retired case cannot go back to candidate")

	_, err = w.svc.UpdateCase(t.Context(), &services.UpdateExtractionCaseRequest{
		TenantInfo: testTenant(),
		ID:         created.ID,
		Version:    created.Version,
	}, testActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
}

func TestStartRun_CreatesPendingResultsAndStartsTheWorkflow(t *testing.T) {
	t.Parallel()

	w := newWorld()
	evalCase := w.seedActiveCase(t)
	provider := w.seedProvider()

	run, err := w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.NoError(t, err)

	assert.Equal(t, extractioneval.RunStatusQueued, run.Status)
	assert.Equal(t, provider.Name, run.ProviderName)
	assert.Equal(t, provider.Model, run.ProviderModel)
	assert.Equal(t, extractioneval.DefaultCaseLimit, run.CaseLimit)
	assert.Equal(t, 1, run.CasesTotal)
	assert.Equal(t, extractioneval.RunWorkflowID(run.ID), run.WorkflowID)
	require.Len(t, w.starter.started, 1)
	require.Len(t, w.results.items, 1)
	assert.Equal(t, evalCase.ID, w.results.items[0].CaseID)
	assert.Equal(t, 1, w.results.items[0].Ordinal)
	assert.Equal(t, extractioneval.ResultStatusPending, w.results.items[0].Status)
}

func TestStartRun_RefusesWhatCannotRun(t *testing.T) {
	t.Parallel()

	w := newWorld()
	provider := w.seedProvider()

	_, err := w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.Error(t, err, "no active cases")
	assert.True(t, errortypes.IsBusinessError(err))

	w.seedActiveCase(t)
	_, err = w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
	}, testActor())
	require.Error(t, err, "no provider chosen")

	provider.Enabled = false
	_, err = w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.Error(t, err, "a disabled provider cannot be evaluated")

	provider.Enabled = true
	provider.Tasks = []aiprovider.Task{aiprovider.TaskGeneral}
	_, err = w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.Error(t, err, "a provider not assigned to extraction cannot be evaluated")
	assert.Empty(t, w.starter.started)
}

func TestStartRun_MarksTheRunFailedWhenTheWorkflowCannotStart(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.seedActiveCase(t)
	provider := w.seedProvider()
	w.starter.err = errUpstream

	_, err := w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.ErrorIs(t, err, errUpstream)

	require.Len(t, w.runs.runs, 1)
	for _, run := range w.runs.runs {
		assert.Equal(t, extractioneval.RunStatusFailed, run.Status)
		assert.NotEmpty(t, run.FailureMessage)
		assert.Equal(t, 1, run.CasesSkipped)
	}
}

func startedRun(t *testing.T, w *world) *extractioneval.ExtractionRun {
	t.Helper()
	w.seedActiveCase(t)
	provider := w.seedProvider()
	run, err := w.svc.StartRun(t.Context(), &services.StartExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ProviderID: provider.ID,
	}, testActor())
	require.NoError(t, err)
	_, err = w.svc.BeginRun(t.Context(), repositories.GetExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		ID:         run.ID,
	})
	require.NoError(t, err)
	return run
}

func TestEvaluateResult_ScoresThePredictionAgainstTheCase(t *testing.T) {
	t.Parallel()

	w := newWorld()
	run := startedRun(t, w)
	cost := decimal.RequireFromString("0.0031")
	w.predictor.prediction = &services.ExtractionPrediction{
		DraftData: map[string]any{
			"fields": map[string]any{
				"referenceNumber": map[string]any{"value": "LD-10442", "source": "ai", "confidence": 0.9},
				"rate":            map[string]any{"value": "$2,500.00", "source": "ai", "confidence": 0.8},
			},
			"stops": []map[string]any{
				{"role": "pickup", "name": "Northwind Foods", "city": "Dallas", "date": "03/15/2026"},
			},
		},
		Model:        "trenova-extract-8b-q4",
		ProviderID:   run.ProviderID,
		InputTokens:  900,
		OutputTokens: 140,
		LatencyMs:    2100,
		CostUSD:      &cost,
	}
	resultID := w.results.items[0].ID

	err := w.svc.EvaluateResult(t.Context(), &services.EvaluateExtractionResultRequest{
		TenantInfo: testTenant(),
		ResultID:   resultID,
	})
	require.NoError(t, err)

	require.Len(t, w.predictor.requests, 1)
	assert.Equal(t, run.ProviderID, w.predictor.requests[0].ProviderID)
	assert.Equal(t, "ratecon-4471.pdf", w.predictor.requests[0].FileName)

	result := w.results.items[0]
	assert.Equal(t, extractioneval.ResultStatusCompleted, result.Status)
	assert.Equal(t, "trenova-extract-8b-q4", result.Model)
	assert.Equal(t, 1, result.CorrectedCount, "the rate was wrong")
	assert.Positive(t, result.CorrectCount)
	assert.InDelta(t, float64(result.CorrectCount)/float64(result.ScoredCount), result.Accuracy, 1e-9)
	assert.True(t, cost.Equal(result.CostUSD))
	assert.Equal(t, int64(2100), result.LatencyMs)
	require.NotNil(t, result.Predicted)
	assert.Len(t, result.Predicted.Stops, 1)

	finished, err := w.svc.FinishRun(t.Context(), &services.FinishExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		RunID:      run.ID,
		Status:     extractioneval.RunStatusCompleted,
	})
	require.NoError(t, err)
	assert.Equal(t, extractioneval.RunStatusCompleted, finished.Status)
	assert.Equal(t, 1, finished.CasesCompleted)
	assert.Equal(t, "trenova-extract-8b-q4", finished.ServedModel)
	assert.Equal(t, result.ScoredCount, finished.ScoredCount)
	assert.InDelta(t, result.Accuracy, finished.Accuracy, 1e-9)
	assert.NotEmpty(t, finished.FieldAccuracy)
	assert.True(t, cost.Equal(finished.CostUSD))
	assert.Equal(t, int64(2100), finished.AvgLatencyMs)
	require.NotNil(t, finished.FinishedAt)
}

func TestEvaluateResult_RetriesTransientFailuresThenRecordsThem(t *testing.T) {
	t.Parallel()

	w := newWorld()
	startedRun(t, w)
	w.predictor.err = errUpstream
	resultID := w.results.items[0].ID

	err := w.svc.EvaluateResult(t.Context(), &services.EvaluateExtractionResultRequest{
		TenantInfo: testTenant(),
		ResultID:   resultID,
	})
	require.ErrorIs(t, err, errUpstream, "a transient failure is retried")
	assert.Equal(t, extractioneval.ResultStatusPending, w.results.items[0].Status)

	err = w.svc.EvaluateResult(t.Context(), &services.EvaluateExtractionResultRequest{
		TenantInfo:   testTenant(),
		ResultID:     resultID,
		FinalAttempt: true,
	})
	require.NoError(t, err, "the last attempt records the failure and lets the run go on")
	assert.Equal(t, extractioneval.ResultStatusFailed, w.results.items[0].Status)
	assert.NotEmpty(t, w.results.items[0].ErrorMessage)

	err = w.svc.EvaluateResult(t.Context(), &services.EvaluateExtractionResultRequest{
		TenantInfo: testTenant(),
		ResultID:   resultID,
	})
	require.NoError(t, err, "a result already evaluated is left alone")
	assert.Len(t, w.predictor.requests, 2)
}

func TestEvaluateResult_ASchemaFailureIsNotRetried(t *testing.T) {
	t.Parallel()

	w := newWorld()
	startedRun(t, w)
	w.predictor.err = services.ErrModelSchemaValidation

	err := w.svc.EvaluateResult(t.Context(), &services.EvaluateExtractionResultRequest{
		TenantInfo: testTenant(),
		ResultID:   w.results.items[0].ID,
	})
	require.NoError(t, err)
	assert.Equal(t, extractioneval.ResultStatusFailed, w.results.items[0].Status)
	assert.Equal(t, "The model's answer did not match the extraction schema", w.results.items[0].ErrorMessage)
}

func TestCheckContinue_StopsForTheBudgetAndForACancel(t *testing.T) {
	t.Parallel()

	w := newWorld()
	run := startedRun(t, w)
	request := repositories.GetExtractionEvalRunRequest{TenantInfo: testTenant(), ID: run.ID}

	decision, err := w.svc.CheckContinue(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Stop)

	w.budget.decision = &agentquality.BudgetDecision{Stop: true, Reason: "Tonight's budget is spent."}
	decision, err = w.svc.CheckContinue(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Stop)
	assert.Equal(t, extractioneval.RunStatusBudgetStopped, decision.Status)

	canceled, err := w.svc.CancelRun(t.Context(), request, testActor())
	require.NoError(t, err)
	assert.Equal(t, extractioneval.RunStatusCanceled, canceled.Status)
	assert.Equal(t, 1, canceled.CasesSkipped)

	decision, err = w.svc.CheckContinue(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Stop)
	assert.Equal(t, extractioneval.RunStatusCanceled, decision.Status)

	_, err = w.svc.CancelRun(t.Context(), request, testActor())
	require.Error(t, err, "a finished run cannot be canceled")

	finished, err := w.svc.FinishRun(t.Context(), &services.FinishExtractionEvalRunRequest{
		TenantInfo: testTenant(),
		RunID:      run.ID,
		Status:     extractioneval.RunStatusCanceled,
	})
	require.NoError(t, err)
	assert.Equal(t, extractioneval.RunStatusCanceled, finished.Status, "finishing a canceled run keeps it canceled")
}

func TestAccuracy_GroupsProductionCorrectionsByModelAndKind(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.seedActiveCase(t)
	w.corrections.rows = []*aicorrection.Correction{
		{
			ExtractionModel: "gpt-large", DocumentKind: "RateConfirmation",
			ScoredCount: 10, CorrectCount: 9, CorrectedCount: 1,
			FieldResults: []aicorrection.FieldResult{{Key: "rate", Outcome: aicorrection.OutcomeCorrect}},
		},
		{
			ExtractionModel: "gpt-large", DocumentKind: "BillOfLading",
			ScoredCount: 10, CorrectCount: 5, MissedCount: 5,
			FieldResults: []aicorrection.FieldResult{{Key: "rate", Outcome: aicorrection.OutcomeMissed}},
		},
		{
			DocumentKind: "RateConfirmation",
			ScoredCount:  4, CorrectCount: 2, CorrectedCount: 2,
		},
	}

	report, err := w.svc.Accuracy(t.Context(), &services.ExtractionAccuracyRequest{TenantInfo: testTenant()})
	require.NoError(t, err)

	assert.Equal(t, 30, report.WindowDays)
	assert.Equal(t, testNow-30*86400, w.corrections.since)
	assert.Equal(t, 3, report.Corrections)
	assert.Equal(t, 24, report.Scored)
	assert.Equal(t, 16, report.Correct)
	assert.InDelta(t, 16.0/24.0, report.Accuracy, 1e-9)
	require.Len(t, report.ByModel, 2)
	assert.Equal(t, "gpt-large", report.ByModel[0].Key)
	assert.Equal(t, 2, report.ByModel[0].Corrections)
	assert.InDelta(t, 14.0/20.0, report.ByModel[0].Accuracy, 1e-9)
	assert.Empty(t, report.ByModel[1].Key, "rules-only drafts have no model")
	require.Len(t, report.ByKind, 2)
	require.Len(t, report.Fields, 1)
	assert.InDelta(t, 0.5, report.Fields[0].Accuracy, 1e-9)
	assert.Equal(t, 1, report.Cases.Active)

	_, err = w.svc.Accuracy(t.Context(), &services.ExtractionAccuracyRequest{
		TenantInfo: testTenant(),
		WindowDays: 400,
	})
	require.Error(t, err)
}

func TestPurgeExpiredRuns_UsesTheCorrectionRetention(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.retention.settings = &tenant.DataRetention{AICorrectionRetentionPeriod: 90}

	purged, err := w.svc.PurgeExpiredRuns(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: testTenant(),
		Now:        testNow,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), purged)
	require.Len(t, w.runs.purged, 1)
	assert.Equal(t, testNow-90*86400, w.runs.purged[0].Before)
}

func TestDeleteCase_RemovesIt(t *testing.T) {
	t.Parallel()

	w := newWorld()
	created := w.seedActiveCase(t)

	require.NoError(t, w.svc.DeleteCase(t.Context(), repositories.GetExtractionEvalCaseRequest{
		TenantInfo: testTenant(),
		ID:         created.ID,
	}, testActor()))
	assert.Empty(t, w.cases.items)
}
