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
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	supersededReason = "The document was extracted again before the shadow ran"
	deletedReason    = "The document was deleted before the shadow ran"
	budgetReason     = "The evaluation budget is spent: "
)

var _ services.ExtractionShadowRunner = (*Runner)(nil)

var errPredictorUnavailable = errors.New("extraction shadow predictor is not configured")

type RunnerParams struct {
	fx.In

	Logger    *zap.Logger
	Results   repositories.ExtractionShadowResultRepository
	Scorer    *Scorer
	Budget    services.EvaluationBudget          `optional:"true"`
	Predictor services.ExtractionShadowPredictor `optional:"true"`
}

type Runner struct {
	l         *zap.Logger
	results   repositories.ExtractionShadowResultRepository
	scorer    *Scorer
	budget    services.EvaluationBudget
	predictor services.ExtractionShadowPredictor
	now       func() int64
}

func NewRunner(p RunnerParams) *Runner {
	return &Runner{
		l:         p.Logger.Named("service.extractionshadow-runner"),
		results:   p.Results,
		scorer:    p.Scorer,
		budget:    p.Budget,
		predictor: p.Predictor,
		now:       timeutils.NowUnix,
	}
}

func AsRunner(r *Runner) services.ExtractionShadowRunner { return r }

func (r *Runner) RunShadow(
	ctx context.Context,
	req *services.RunExtractionShadowRequest,
) error {
	result, err := r.results.GetByID(ctx, repositories.GetExtractionShadowResultRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ResultID,
	})
	if err != nil {
		return err
	}
	if result.Status.IsSettled() {
		return nil
	}
	if r.predictor == nil {
		return errortypes.NewBusinessError(
			"Shadow extractions cannot run on this installation",
		).WithInternal(errPredictorUnavailable)
	}

	if stop, reason, budgetErr := r.budgetSpent(ctx, result); budgetErr != nil {
		return budgetErr
	} else if stop {
		return r.settle(ctx, result, extractionshadow.ResultStatusSkipped, reason)
	}

	started := r.now()
	if result.StartedAt == nil {
		result.StartedAt = &started
	}

	prediction, err := r.predictor.PredictShadowDraft(ctx, &services.PredictShadowDraftRequest{
		TenantInfo:  tenantOf(result),
		DocumentID:  result.DocumentID,
		ExtractedAt: result.ExtractedAt,
		ProviderID:  result.ProviderID,
	})
	switch {
	case err == nil:
	case errors.Is(err, services.ErrShadowSuperseded):
		return r.settle(ctx, result, extractionshadow.ResultStatusSkipped, supersededReason)
	case errortypes.IsNotFoundError(err):
		return r.settle(ctx, result, extractionshadow.ResultStatusSkipped, deletedReason)
	case extractionfailure.Retryable(err) && !req.FinalAttempt:
		return fmt.Errorf("predict shadow extraction %s: %w", result.ID, err)
	default:
		r.l.Warn("shadow extraction failed",
			zap.String("resultId", result.ID.String()),
			zap.Error(err),
		)
		return r.settle(
			ctx,
			result,
			extractionshadow.ResultStatusFailed,
			extractionfailure.Message(err),
		)
	}

	r.recordPrediction(result, prediction)
	result.Settle(extractionshadow.ResultStatusCompleted, "", r.now())
	saved, err := r.results.Save(ctx, result)
	if err != nil {
		return err
	}

	if err = r.scorer.scoreAgainstCorrection(ctx, saved); err != nil {
		r.l.Warn("failed to score a shadow extraction on completion",
			zap.String("resultId", saved.ID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (r *Runner) FailShadow(
	ctx context.Context,
	req repositories.GetExtractionShadowResultRequest,
	message string,
) error {
	result, err := r.results.GetByID(ctx, req)
	if err != nil {
		return err
	}
	if result.Status.IsSettled() {
		return nil
	}

	return r.settle(ctx, result, extractionshadow.ResultStatusFailed, message)
}

func (r *Runner) budgetSpent(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
) (stop bool, reason string, err error) {
	if r.budget == nil {
		return false, "", nil
	}

	decision, err := r.budget.CheckEvaluationBudget(ctx, tenantOf(result))
	if err != nil {
		return false, "", err
	}
	if !decision.Stop {
		return false, "", nil
	}

	return true, budgetReason + decision.Reason, nil
}

func (r *Runner) recordPrediction(
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

func (r *Runner) settle(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
	status extractionshadow.ResultStatus,
	reason string,
) error {
	result.Settle(
		status,
		stringutils.TruncateRunes(reason, extractionshadow.MaxReasonRunes),
		r.now(),
	)
	_, err := r.results.Save(ctx, result)

	return err
}
