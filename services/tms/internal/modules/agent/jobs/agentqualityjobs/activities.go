package agentqualityjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type qualityEngine interface {
	PlanSweep(
		ctx context.Context,
		req *agentqualityengine.SweepRequest,
	) (*agentqualityengine.SweepPlan, error)
	OpenSuite(
		ctx context.Context,
		req *agentqualityengine.OpenSuiteRequest,
	) (*agentqualityengine.OpenedSuite, error)
	RecordSkipped(
		ctx context.Context,
		req *agentqualityengine.RecordSkippedRequest,
	) (*agentqualityengine.OpenedSuite, error)
	ListSuiteCases(
		ctx context.Context,
		req *agentqualityengine.ListSuiteCasesRequest,
	) ([]agentqualityengine.SuiteCase, error)
	CheckBudget(
		ctx context.Context,
		req *agentqualityengine.CheckBudgetRequest,
	) (*agentqualitysuite.BudgetDecision, error)
	ScoreSuite(
		ctx context.Context,
		req *agentqualityengine.ScoreSuiteRequest,
	) (*agentqualityengine.ScoredSuite, error)
	JudgeCase(
		ctx context.Context,
		req *agentqualityengine.JudgeCaseRequest,
	) (*agentqualityengine.JudgedCase, error)
	SubmitJudgeBatch(
		ctx context.Context,
		req *agentqualityengine.SubmitJudgeBatchRequest,
	) (*agentqualityengine.JudgeBatch, error)
	CheckJudgeBatch(
		ctx context.Context,
		req *agentqualityengine.CheckJudgeBatchRequest,
	) (*agentqualityengine.JudgeBatchStatus, error)
	CollectJudgeBatch(
		ctx context.Context,
		req *agentqualityengine.CollectJudgeBatchRequest,
	) (*agentqualityengine.CollectedJudgeBatch, error)
	FinalizeSuite(
		ctx context.Context,
		req *agentqualityengine.FinalizeSuiteRequest,
	) (*agentqualityengine.FinalizedSuite, error)
	FailSuite(ctx context.Context, req *agentqualityengine.FailSuiteRequest) error
}

type scheduleReconciler interface {
	Reconcile(ctx context.Context) (*ReconcileResult, error)
}

type ActivitiesParams struct {
	fx.In

	Quality   *agentqualityengine.Engine
	Schedules *QualitySchedules
}

type Activities struct {
	quality   qualityEngine
	schedules scheduleReconciler
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{quality: p.Quality, schedules: p.Schedules}
}

func nonRetryable(err error) error {
	if err == nil || errortypes.IsVersionMismatchError(err) {
		return err
	}
	if errortypes.IsError(err) || errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err) {
		return temporal.NewNonRetryableApplicationError(err.Error(), "AgentQuality", err)
	}

	return err
}

func (a *Activities) PlanSweepActivity(
	ctx context.Context,
	req *agentqualityengine.SweepRequest,
) (*agentqualityengine.SweepPlan, error) {
	plan, err := a.quality.PlanSweep(ctx, req)

	return plan, nonRetryable(err)
}

func (a *Activities) OpenSuiteActivity(
	ctx context.Context,
	req *agentqualityengine.OpenSuiteRequest,
) (*agentqualityengine.OpenedSuite, error) {
	opened, err := a.quality.OpenSuite(ctx, req)

	return opened, nonRetryable(err)
}

func (a *Activities) RecordSkippedSuiteActivity(
	ctx context.Context,
	req *agentqualityengine.RecordSkippedRequest,
) (*agentqualityengine.OpenedSuite, error) {
	recorded, err := a.quality.RecordSkipped(ctx, req)

	return recorded, nonRetryable(err)
}

func (a *Activities) ListSuiteCasesActivity(
	ctx context.Context,
	req *agentqualityengine.ListSuiteCasesRequest,
) ([]agentqualityengine.SuiteCase, error) {
	cases, err := a.quality.ListSuiteCases(ctx, req)

	return cases, nonRetryable(err)
}

func (a *Activities) CheckEvalBudgetActivity(
	ctx context.Context,
	req *agentqualityengine.CheckBudgetRequest,
) (*agentqualitysuite.BudgetDecision, error) {
	decision, err := a.quality.CheckBudget(ctx, req)

	return decision, nonRetryable(err)
}

func (a *Activities) ScoreSuiteActivity(
	ctx context.Context,
	req *agentqualityengine.ScoreSuiteRequest,
) (*agentqualityengine.ScoredSuite, error) {
	scored, err := a.quality.ScoreSuite(ctx, req)

	return scored, nonRetryable(err)
}

func (a *Activities) JudgeCaseActivity(
	ctx context.Context,
	req *agentqualityengine.JudgeCaseRequest,
) (*agentqualityengine.JudgedCase, error) {
	judged, err := a.quality.JudgeCase(ctx, req)

	return judged, nonRetryable(err)
}

func (a *Activities) SubmitJudgeBatchActivity(
	ctx context.Context,
	req *agentqualityengine.SubmitJudgeBatchRequest,
) (*agentqualityengine.JudgeBatch, error) {
	batch, err := a.quality.SubmitJudgeBatch(ctx, req)

	return batch, nonRetryable(err)
}

func (a *Activities) CheckJudgeBatchActivity(
	ctx context.Context,
	req *agentqualityengine.CheckJudgeBatchRequest,
) (*agentqualityengine.JudgeBatchStatus, error) {
	status, err := a.quality.CheckJudgeBatch(ctx, req)

	return status, nonRetryable(err)
}

func (a *Activities) CollectJudgeBatchActivity(
	ctx context.Context,
	req *agentqualityengine.CollectJudgeBatchRequest,
) (*agentqualityengine.CollectedJudgeBatch, error) {
	collected, err := a.quality.CollectJudgeBatch(ctx, req)

	return collected, nonRetryable(err)
}

func (a *Activities) FinalizeSuiteActivity(
	ctx context.Context,
	req *agentqualityengine.FinalizeSuiteRequest,
) (*agentqualityengine.FinalizedSuite, error) {
	finalized, err := a.quality.FinalizeSuite(ctx, req)

	return finalized, nonRetryable(err)
}

func (a *Activities) FailSuiteActivity(
	ctx context.Context,
	req *agentqualityengine.FailSuiteRequest,
) error {
	return nonRetryable(a.quality.FailSuite(ctx, req))
}

func (a *Activities) ReconcileQualitySchedulesActivity(
	ctx context.Context,
) (*ReconcileResult, error) {
	return a.schedules.Reconcile(ctx)
}
