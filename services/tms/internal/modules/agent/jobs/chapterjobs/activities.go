package chapterjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Chapters   agentports.AssistantChapterService
	Completion agentports.CompletionService
}

type Activities struct {
	chapters   agentports.AssistantChapterService
	completion agentports.CompletionService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{chapters: p.Chapters, completion: p.Completion}
}

var Module = fx.Module("assistant-chapter-jobs", fx.Provide(NewActivities))

func (a *Activities) PlanChapterActivity(
	ctx context.Context,
	req *agentports.ChapterTurnRequest,
) (*agentports.ChapterPlan, error) {
	plan, err := a.chapters.Plan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("plan the turn's chapter: %w", err)
	}

	return plan, nil
}

func (a *Activities) ChapterModelActivity(
	ctx context.Context,
	plan *agentports.ChapterPlan,
) (*agentports.StructuredCompletionResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.completion.CompleteStructured(ctx, plan.Request)
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

func (a *Activities) ApplyChapterActivity(
	ctx context.Context,
	in *ApplyInput,
) (*agentports.ChapterOutcome, error) {
	outcome, err := a.chapters.Apply(ctx, in.Plan, in.Result)
	if err != nil {
		return nil, fmt.Errorf("keep the turn's chapter: %w", err)
	}

	return outcome, nil
}
