package extractionshadowservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/extractionfailure"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	supersededReason = "The document was extracted again before the shadow ran"
	deletedReason    = "The document was deleted before the shadow ran"
	budgetReason     = "The evaluation budget is spent: "
)

var errPredictorUnavailable = errors.New("extraction shadow predictor is not configured")

func (s *Service) RunShadow(
	ctx context.Context,
	req *services.RunExtractionShadowRequest,
) error {
	result, err := s.results.GetByID(ctx, repositories.GetExtractionShadowResultRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ResultID,
	})
	if err != nil {
		return err
	}
	if result.Status.IsSettled() {
		return nil
	}
	if s.predictor == nil {
		return errortypes.NewBusinessError(
			"Shadow extractions cannot run on this installation",
		).WithInternal(errPredictorUnavailable)
	}

	if stop, reason, budgetErr := s.budgetSpent(ctx, result); budgetErr != nil {
		return budgetErr
	} else if stop {
		return s.settle(ctx, result, extractionshadow.ResultStatusSkipped, reason)
	}

	started := s.now()
	if result.StartedAt == nil {
		result.StartedAt = &started
	}

	prediction, err := s.predictor.PredictShadowDraft(ctx, &services.PredictShadowDraftRequest{
		TenantInfo:  tenantOf(result),
		DocumentID:  result.DocumentID,
		ExtractedAt: result.ExtractedAt,
		ProviderID:  result.ProviderID,
	})
	switch {
	case err == nil:
	case errors.Is(err, services.ErrShadowSuperseded):
		return s.settle(ctx, result, extractionshadow.ResultStatusSkipped, supersededReason)
	case errortypes.IsNotFoundError(err):
		return s.settle(ctx, result, extractionshadow.ResultStatusSkipped, deletedReason)
	case extractionfailure.Retryable(err) && !req.FinalAttempt:
		return fmt.Errorf("predict shadow extraction %s: %w", result.ID, err)
	default:
		s.l.Warn("shadow extraction failed",
			zap.String("resultId", result.ID.String()),
			zap.Error(err),
		)
		return s.settle(
			ctx,
			result,
			extractionshadow.ResultStatusFailed,
			extractionfailure.Message(err),
		)
	}

	s.recordPrediction(result, prediction)
	result.Settle(extractionshadow.ResultStatusCompleted, "", s.now())
	saved, err := s.results.Save(ctx, result)
	if err != nil {
		return err
	}

	if err = s.scorer.scoreAgainstCorrection(ctx, saved); err != nil {
		s.l.Warn("failed to score a shadow extraction on completion",
			zap.String("resultId", saved.ID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (s *Service) FailShadow(
	ctx context.Context,
	req repositories.GetExtractionShadowResultRequest,
	message string,
) error {
	result, err := s.results.GetByID(ctx, req)
	if err != nil {
		return err
	}
	if result.Status.IsSettled() {
		return nil
	}

	return s.settle(ctx, result, extractionshadow.ResultStatusFailed, message)
}

func (s *Service) budgetSpent(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
) (stop bool, reason string, err error) {
	if s.budget == nil {
		return false, "", nil
	}

	decision, err := s.budget.CheckEvaluationBudget(ctx, tenantOf(result))
	if err != nil {
		return false, "", err
	}
	if !decision.Stop {
		return false, "", nil
	}

	return true, budgetReason + decision.Reason, nil
}

func (s *Service) recordPrediction(
	result *extractionshadow.ShadowResult,
	prediction *services.ShadowDraftPrediction,
) {
	result.DraftData = prediction.DraftData
	result.Predicted = aicorrection.ReadPrediction(prediction.DraftData).Snapshot
	result.Accepted = prediction.Accepted
	result.RejectionReason = stringutils.TruncateRunes(
		prediction.RejectionReason, extractionshadow.MaxReasonRunes,
	)
	result.ServedModel = stringutils.TruncateRunes(prediction.Model, extractionshadow.MaxModelRunes)
	result.LatencyMs = prediction.LatencyMs
	result.InputTokens = int64(prediction.InputTokens)
	result.OutputTokens = int64(prediction.OutputTokens)
	result.CostUSD = decimal.Zero
	if prediction.CostUSD != nil {
		result.CostUSD = *prediction.CostUSD
	}
}

func (s *Service) settle(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
	status extractionshadow.ResultStatus,
	reason string,
) error {
	result.Settle(
		status,
		stringutils.TruncateRunes(reason, extractionshadow.MaxReasonRunes),
		s.now(),
	)
	_, err := s.results.Save(ctx, result)

	return err
}
