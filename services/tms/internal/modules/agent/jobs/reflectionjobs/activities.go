package reflectionjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Reflections agentports.AgentReflectionService
	Completion  agentports.CompletionService
}

type Activities struct {
	reflections agentports.AgentReflectionService
	completion  agentports.CompletionService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		reflections: p.Reflections,
		completion:  p.Completion,
	}
}

func (a *Activities) PrepareThreadReflectionActivity(
	ctx context.Context,
	req *agentports.ReflectOnThreadRequest,
) (*agentports.ReflectionPlan, error) {
	plan, err := a.reflections.PrepareThread(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prepare a look back over the conversation: %w", err)
	}

	return plan, nil
}

func (a *Activities) PrepareRunReflectionActivity(
	ctx context.Context,
	req *agentports.ReflectOnRunRequest,
) (*agentports.ReflectionPlan, error) {
	plan, err := a.reflections.PrepareRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prepare a look back over the run: %w", err)
	}

	return plan, nil
}

func (a *Activities) ReflectionModelActivity(
	ctx context.Context,
	plan *agentports.ReflectionPlan,
) (*agentports.StructuredCompletionResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.completion.CompleteStructured(ctx, plan.Request)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

func (a *Activities) FinishReflectionActivity(
	ctx context.Context,
	in *FinishReflectionInput,
) (*agentports.ReflectionOutcome, error) {
	outcome, err := a.reflections.Finish(ctx, in.Plan, in.Result)
	if err != nil {
		return nil, fmt.Errorf("keep what the look back learned: %w", err)
	}

	return outcome, nil
}

func (a *Activities) FailReflectionActivity(
	ctx context.Context,
	req *agentports.FailReflectionRequest,
) error {
	if err := a.reflections.Fail(ctx, req); err != nil {
		return fmt.Errorf("record a look back that failed: %w", err)
	}

	return nil
}
