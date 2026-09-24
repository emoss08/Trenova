package agentqualityjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/services/agentqualityservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type qualityEngine interface {
	PlanSweep(
		ctx context.Context,
		req *agentqualityservice.SweepRequest,
	) (*agentqualityservice.SweepPlan, error)
	OpenSuite(
		ctx context.Context,
		req *agentqualityservice.OpenSuiteRequest,
	) (*agentqualityservice.OpenedSuite, error)
	RecordSkipped(
		ctx context.Context,
		req *agentqualityservice.RecordSkippedRequest,
	) (*agentqualityservice.OpenedSuite, error)
	ListSuiteCases(
		ctx context.Context,
		req *agentqualityservice.ListSuiteCasesRequest,
	) ([]agentqualityservice.SuiteCase, error)
	CheckBudget(
		ctx context.Context,
		req *agentqualityservice.CheckBudgetRequest,
	) (*agentquality.BudgetDecision, error)
	ScoreSuite(
		ctx context.Context,
		req *agentqualityservice.ScoreSuiteRequest,
	) (*agentqualityservice.ScoredSuite, error)
	JudgeCase(
		ctx context.Context,
		req *agentqualityservice.JudgeCaseRequest,
	) (*agentqualityservice.JudgedCase, error)
	FinalizeSuite(
		ctx context.Context,
		req *agentqualityservice.FinalizeSuiteRequest,
	) (*agentqualityservice.FinalizedSuite, error)
	FailSuite(ctx context.Context, req *agentqualityservice.FailSuiteRequest) error
}

type scheduleReconciler interface {
	Reconcile(ctx context.Context) (*ReconcileResult, error)
}

type ActivitiesParams struct {
	fx.In

	Quality   *agentqualityservice.Service
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
	req *agentqualityservice.SweepRequest,
) (*agentqualityservice.SweepPlan, error) {
	plan, err := a.quality.PlanSweep(ctx, req)

	return plan, nonRetryable(err)
}

func (a *Activities) OpenSuiteActivity(
	ctx context.Context,
	req *agentqualityservice.OpenSuiteRequest,
) (*agentqualityservice.OpenedSuite, error) {
	opened, err := a.quality.OpenSuite(ctx, req)

	return opened, nonRetryable(err)
}

func (a *Activities) RecordSkippedSuiteActivity(
	ctx context.Context,
	req *agentqualityservice.RecordSkippedRequest,
) (*agentqualityservice.OpenedSuite, error) {
	recorded, err := a.quality.RecordSkipped(ctx, req)

	return recorded, nonRetryable(err)
}

func (a *Activities) ListSuiteCasesActivity(
	ctx context.Context,
	req *agentqualityservice.ListSuiteCasesRequest,
) ([]agentqualityservice.SuiteCase, error) {
	cases, err := a.quality.ListSuiteCases(ctx, req)

	return cases, nonRetryable(err)
}

func (a *Activities) CheckEvalBudgetActivity(
	ctx context.Context,
	req *agentqualityservice.CheckBudgetRequest,
) (*agentquality.BudgetDecision, error) {
	decision, err := a.quality.CheckBudget(ctx, req)

	return decision, nonRetryable(err)
}

func (a *Activities) ScoreSuiteActivity(
	ctx context.Context,
	req *agentqualityservice.ScoreSuiteRequest,
) (*agentqualityservice.ScoredSuite, error) {
	scored, err := a.quality.ScoreSuite(ctx, req)

	return scored, nonRetryable(err)
}

func (a *Activities) JudgeCaseActivity(
	ctx context.Context,
	req *agentqualityservice.JudgeCaseRequest,
) (*agentqualityservice.JudgedCase, error) {
	judged, err := a.quality.JudgeCase(ctx, req)

	return judged, nonRetryable(err)
}

func (a *Activities) FinalizeSuiteActivity(
	ctx context.Context,
	req *agentqualityservice.FinalizeSuiteRequest,
) (*agentqualityservice.FinalizedSuite, error) {
	finalized, err := a.quality.FinalizeSuite(ctx, req)

	return finalized, nonRetryable(err)
}

func (a *Activities) FailSuiteActivity(
	ctx context.Context,
	req *agentqualityservice.FailSuiteRequest,
) error {
	return nonRetryable(a.quality.FailSuite(ctx, req))
}

func (a *Activities) ReconcileQualitySchedulesActivity(
	ctx context.Context,
) (*ReconcileResult, error) {
	return a.schedules.Reconcile(ctx)
}
