package chapterjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/chapterjobs/chaptercontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var planOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Read what a turn means for the conversation's chapters",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    500 * time.Millisecond,
		BackoffCoefficient: 2,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    4,
	},
}

var modelOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 45 * time.Second,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	Summary:             "Ask whether the question starts a chapter and what to call it",
	RetryPolicy:         modelcall.RetryPolicy(2),
}

var applyOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Keep the conversation's chapters",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    500 * time.Millisecond,
		BackoffCoefficient: 2,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    6,
	},
}

type ChaptersResult struct {
	Turns     int `json:"turns"`
	Opened    int `json:"opened"`
	Retitled  int `json:"retitled"`
	Removed   int `json:"removed"`
	Fallbacks int `json:"fallbacks"`
	Failed    int `json:"failed"`
}

type ApplyInput struct {
	Plan   *agentports.ChapterPlan                `json:"plan"`
	Result *agentports.StructuredCompletionResult `json:"result,omitempty"`
}

func Workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{{
		Name:        chaptercontract.ThreadChaptersWorkflowName,
		Fn:          ThreadChaptersWorkflow,
		Description: "Mark and name the chapters of a conversation as each question is asked and answered",
	}}
}

func ThreadChaptersWorkflow(
	ctx workflow.Context,
	input *chaptercontract.ThreadChaptersInput,
) (*ChaptersResult, error) {
	signals := workflow.GetSignalChannel(ctx, chaptercontract.TurnSignalName)
	result := &ChaptersResult{}

	for {
		var next agentports.ChapterTurnRequest
		if !signals.ReceiveAsync(&next) {
			received, _ := signals.ReceiveWithTimeout(ctx, chaptercontract.IdleWait, &next)
			if !received {
				if !signals.ReceiveAsync(&next) {
					return result, nil
				}
			}
		}

		handle(ctx, input.TenantInfo, &next, result)
		result.Turns++

		if result.Turns >= chaptercontract.TurnsPerRun && signals.Len() == 0 {
			return result, workflow.NewContinueAsNewError(ctx, ThreadChaptersWorkflow, input)
		}
	}
}

func handle(
	ctx workflow.Context,
	tenant pagination.TenantInfo,
	req *agentports.ChapterTurnRequest,
	result *ChaptersResult,
) {
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var plan agentports.ChapterPlan
	if err := workflow.ExecuteActivity(
		withPriority(ctx, planOptions, tenant), a.PlanChapterActivity, req,
	).Get(ctx, &plan); err != nil {
		logger.Error("could not read what a turn means for its chapters",
			"turnId", req.TurnID.String(), "error", err.Error())
		result.Failed++

		return
	}
	if plan.Skipped() {
		return
	}

	input := &ApplyInput{Plan: &plan}
	if plan.NeedsModel() {
		var completion agentports.StructuredCompletionResult
		if err := workflow.ExecuteActivity(
			withPriority(ctx, modelOptions, tenant), a.ChapterModelActivity, &plan,
		).Get(ctx, &completion); err != nil {
			logger.Warn("the chapter model did not answer; keeping what the turn gives",
				"turnId", req.TurnID.String(), "error", err.Error())
		} else {
			input.Result = &completion
		}
	}

	var outcome agentports.ChapterOutcome
	if err := workflow.ExecuteActivity(
		withPriority(ctx, applyOptions, tenant), a.ApplyChapterActivity, input,
	).Get(ctx, &outcome); err != nil {
		logger.Error("could not keep a conversation's chapters",
			"turnId", req.TurnID.String(), "error", err.Error())
		result.Failed++

		return
	}
	result.absorb(&outcome)
}

func (r *ChaptersResult) absorb(outcome *agentports.ChapterOutcome) {
	if outcome.Opened {
		r.Opened++
	}
	if outcome.Retitled {
		r.Retitled++
	}
	if outcome.Removed {
		r.Removed++
	}
	if outcome.Fallback {
		r.Fallbacks++
	}
}

func withPriority(
	ctx workflow.Context,
	options workflow.ActivityOptions,
	tenant pagination.TenantInfo,
) workflow.Context {
	options.Priority = temporal.Priority{
		PriorityKey: agentflowcontract.PriorityOneShot,
		FairnessKey: tenant.OrgID.String(),
	}

	return workflow.WithActivityOptions(ctx, options)
}
