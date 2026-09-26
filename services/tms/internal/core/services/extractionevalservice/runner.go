package extractionevalservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var errPredictorUnavailable = errors.New("extraction predictor is not configured")

func (s *Service) BeginRun(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	run, err := s.runs.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if run.Status != extractioneval.RunStatusQueued {
		return run, nil
	}

	now := s.now()
	run.Status = extractioneval.RunStatusRunning
	run.StartedAt = &now

	return s.runs.Update(ctx, run)
}

func (s *Service) ListPending(
	ctx context.Context,
	req repositories.ListPendingExtractionEvalResultsRequest,
) ([]*extractioneval.ExtractionResult, error) {
	return s.results.ListPending(ctx, req)
}

func (s *Service) CheckContinue(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
) (*services.ExtractionEvalDecision, error) {
	run, err := s.runs.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if !run.Status.IsActive() {
		return &services.ExtractionEvalDecision{Stop: true, Status: run.Status, Reason: run.StopReason}, nil
	}
	if s.budget == nil {
		return &services.ExtractionEvalDecision{}, nil
	}

	decision, err := s.budget.CheckEvaluationBudget(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if decision.Stop {
		return &services.ExtractionEvalDecision{
			Stop:   true,
			Status: extractioneval.RunStatusBudgetStopped,
			Reason: decision.Reason,
		}, nil
	}

	return &services.ExtractionEvalDecision{}, nil
}

func (s *Service) EvaluateResult(
	ctx context.Context,
	req *services.EvaluateExtractionResultRequest,
) error {
	if s.predictor == nil {
		return errortypes.NewBusinessError(
			"Extraction evaluations cannot run on this installation",
		).WithInternal(errPredictorUnavailable)
	}

	result, err := s.results.GetByID(ctx, repositories.GetExtractionEvalResultRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ResultID,
	})
	if err != nil {
		return err
	}
	if result.Status != extractioneval.ResultStatusPending {
		return nil
	}

	run, err := s.runs.GetByID(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: req.TenantInfo,
		ID:         result.RunID,
	})
	if err != nil {
		return err
	}
	evalCase, err := s.cases.GetByID(ctx, repositories.GetExtractionEvalCaseRequest{
		TenantInfo: req.TenantInfo,
		ID:         result.CaseID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return s.saveFailure(ctx, result, "The case was deleted before it ran")
		}
		return err
	}

	started := s.now()
	result.StartedAt = &started
	prediction, err := s.predictor.Predict(ctx, &services.PredictExtractionRequest{
		TenantInfo: req.TenantInfo,
		ProviderID: run.ProviderID,
		FileName:   evalCase.FileName,
		Pages:      evalCase.Pages,
	})
	if err != nil {
		if retryable(err) && !req.FinalAttempt {
			return fmt.Errorf("predict extraction case %s: %w", evalCase.ID, err)
		}
		s.l.Warn("extraction evaluation case failed",
			zap.String("runId", run.ID.String()),
			zap.String("caseId", evalCase.ID.String()),
			zap.Error(err),
		)

		return s.saveFailure(ctx, result, failureMessage(err))
	}

	predicted := aicorrection.ReadPrediction(prediction.DraftData)
	result.Predicted = predicted.Snapshot
	result.ApplyScore(aicorrection.Score(predicted, evalCase.Expected))
	result.Model = stringutils.TruncateRunes(prediction.Model, extractioneval.MaxModelRunes)
	result.ProviderID = pulid.PtrOrNil(prediction.ProviderID)
	result.LatencyMs = prediction.LatencyMs
	result.InputTokens = int64(prediction.InputTokens)
	result.OutputTokens = int64(prediction.OutputTokens)
	result.CostUSD = decimal.Zero
	if prediction.CostUSD != nil {
		result.CostUSD = *prediction.CostUSD
	}
	completed := s.now()
	result.CompletedAt = &completed
	result.Status = extractioneval.ResultStatusCompleted

	_, err = s.results.Save(ctx, result)
	return err
}

func (s *Service) saveFailure(
	ctx context.Context,
	result *extractioneval.ExtractionResult,
	message string,
) error {
	completed := s.now()
	result.Status = extractioneval.ResultStatusFailed
	result.ErrorMessage = stringutils.TruncateRunes(message, extractioneval.MaxFailureRunes)
	result.CompletedAt = &completed

	_, err := s.results.Save(ctx, result)
	return err
}

func (s *Service) FinishRun(
	ctx context.Context,
	req *services.FinishExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	run, err := s.runs.GetByID(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.RunID,
	})
	if err != nil {
		return nil, err
	}
	if !run.Status.IsActive() {
		return s.refreshTotals(ctx, run)
	}

	return s.finish(ctx, run, req.Status, req.Reason)
}

func (s *Service) FailRun(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
	message string,
) error {
	run, err := s.runs.GetByID(ctx, req)
	if err != nil {
		return err
	}
	if !run.Status.IsActive() {
		return nil
	}

	run.FailureMessage = stringutils.TruncateRunes(message, extractioneval.MaxFailureRunes)
	_, err = s.finish(ctx, run, extractioneval.RunStatusFailed, "")

	return err
}

func (s *Service) finish(
	ctx context.Context,
	run *extractioneval.ExtractionRun,
	status extractioneval.RunStatus,
	reason string,
) (*extractioneval.ExtractionRun, error) {
	tenant := tenantOf(run)
	if status != extractioneval.RunStatusCompleted {
		if _, err := s.results.SkipPending(ctx, tenant, run.ID); err != nil {
			return nil, err
		}
	}

	results, err := s.results.ListByRun(ctx, tenant, run.ID)
	if err != nil {
		return nil, err
	}

	finished := s.now()
	run.ApplyResults(results)
	run.Status = status
	run.StopReason = stringutils.TruncateRunes(reason, extractioneval.MaxStopReasonRunes)
	run.FinishedAt = &finished
	if run.StartedAt == nil {
		run.StartedAt = &finished
	}

	return s.runs.Update(ctx, run)
}

func (s *Service) refreshTotals(
	ctx context.Context,
	run *extractioneval.ExtractionRun,
) (*extractioneval.ExtractionRun, error) {
	results, err := s.results.ListByRun(ctx, tenantOf(run), run.ID)
	if err != nil {
		return nil, err
	}
	run.ApplyResults(results)

	return s.runs.Update(ctx, run)
}

func retryable(err error) bool {
	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, services.ErrModelSchemaValidation),
		errors.Is(err, services.ErrRequiredProviderUnavailable),
		errors.Is(err, services.ErrNoProviderConfigured),
		errortypes.IsBusinessError(err),
		errortypes.IsError(err),
		errortypes.IsMultiError(err),
		errortypes.IsNotFoundError(err):
		return false
	default:
		return true
	}
}

func failureMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrModelSchemaValidation):
		return "The model's answer did not match the extraction schema"
	case errors.Is(err, services.ErrRequiredProviderUnavailable):
		return "The provider being evaluated is no longer enabled for document extraction"
	case errortypes.IsBusinessError(err):
		return err.Error()
	default:
		return "The model could not be reached after several attempts"
	}
}
