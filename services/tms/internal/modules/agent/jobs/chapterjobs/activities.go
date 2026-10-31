package chapterjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Chapters assistantchapterports.AssistantChapterService
	Nova     llmports.TaskRunner
}

type Activities struct {
	chapters assistantchapterports.AssistantChapterService
	nova     llmports.TaskRunner
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{chapters: p.Chapters, nova: p.Nova}
}

var Module = fx.Module("assistant-chapter-jobs", fx.Provide(NewActivities))

func (a *Activities) PlanChapterActivity(
	ctx context.Context,
	req *assistantchapterports.ChapterTurnRequest,
) (*assistantchapterports.ChapterPlan, error) {
	plan, err := a.chapters.Plan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("plan the turn's chapter: %w", err)
	}

	return plan, nil
}

func (a *Activities) ChapterModelActivity(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
) (*llmports.StructuredCompletionResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	var answer map[string]any
	result, err := a.nova.Run(ctx, plan.Task, &answer)
	if err != nil && !errors.Is(err, llmports.ErrOutsideSchema) {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

func (a *Activities) ApplyChapterActivity(
	ctx context.Context,
	in *ApplyInput,
) (*assistantchapterports.ChapterOutcome, error) {
	outcome, err := a.chapters.Apply(ctx, in.Plan, in.Result)
	if err != nil {
		return nil, fmt.Errorf("keep the turn's chapter: %w", err)
	}

	return outcome, nil
}
