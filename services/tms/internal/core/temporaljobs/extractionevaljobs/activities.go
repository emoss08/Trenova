package extractionevaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Runner services.ExtractionEvalRunner
	Logger *zap.Logger
}

type Activities struct {
	runner services.ExtractionEvalRunner
	l      *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runner: p.Runner,
		l:      p.Logger.Named("job.extraction-eval"),
	}
}

func (a *Activities) BeginExtractionEvalRunActivity(ctx context.Context, input *RunInput) error {
	if _, err := a.runner.BeginRun(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: input.tenant(),
		ID:         input.RunID,
	}); err != nil {
		return fmt.Errorf("begin extraction evaluation run: %w", err)
	}

	return nil
}

func (a *Activities) ListPendingExtractionEvalCasesActivity(
	ctx context.Context,
	input *ListPendingInput,
) ([]PendingCase, error) {
	results, err := a.runner.ListPending(ctx, repositories.ListPendingExtractionEvalResultsRequest{
		TenantInfo:   input.tenant(),
		RunID:        input.RunID,
		AfterOrdinal: input.AfterOrdinal,
		Limit:        input.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list pending extraction evaluation cases: %w", err)
	}

	pending := make([]PendingCase, 0, len(results))
	for _, result := range results {
		pending = append(pending, PendingCase{ResultID: result.ID, Ordinal: result.Ordinal})
	}

	return pending, nil
}

func (a *Activities) CheckExtractionEvalBudgetActivity(
	ctx context.Context,
	input *RunInput,
) (*ContinueDecision, error) {
	decision, err := a.runner.CheckContinue(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: input.tenant(),
		ID:         input.RunID,
	})
	if err != nil {
		return nil, fmt.Errorf("check extraction evaluation budget: %w", err)
	}

	return &ContinueDecision{
		Stop:   decision.Stop,
		Status: decision.Status.String(),
		Reason: decision.Reason,
	}, nil
}

func (a *Activities) EvaluateExtractionEvalCaseActivity(ctx context.Context, input *EvaluateInput) error {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.runner.EvaluateResult(ctx, &services.EvaluateExtractionResultRequest{
		TenantInfo:   input.tenant(),
		ResultID:     input.ResultID,
		FinalAttempt: modelcall.FinalAttempt(ctx, evaluateAttempts),
	})
}

func (a *Activities) FinishExtractionEvalRunActivity(ctx context.Context, input *FinishInput) (*RunOutcome, error) {
	run, err := a.runner.FinishRun(ctx, &services.FinishExtractionEvalRunRequest{
		TenantInfo: input.tenant(),
		RunID:      input.RunID,
		Status:     extractioneval.RunStatus(input.Status),
		Reason:     input.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("finish extraction evaluation run: %w", err)
	}

	a.l.Info("extraction evaluation finished",
		zap.String("runId", run.ID.String()),
		zap.String("status", run.Status.String()),
		zap.Int("completed", run.CasesCompleted),
		zap.Float64("accuracy", run.Accuracy),
	)

	return &RunOutcome{RunID: run.ID, Status: run.Status.String()}, nil
}

func (a *Activities) FailExtractionEvalRunActivity(ctx context.Context, input *FailInput) error {
	if err := a.runner.FailRun(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: input.tenant(),
		ID:         input.RunID,
	}, input.Message); err != nil {
		return fmt.Errorf("fail extraction evaluation run: %w", err)
	}

	return nil
}
