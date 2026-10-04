package reflectionjobs

import (
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	prepareTimeout = 2 * time.Minute
	modelTimeout   = 5 * time.Minute
	finishTimeout  = 2 * time.Minute
)

var prepareOptions = workflow.ActivityOptions{
	StartToCloseTimeout: prepareTimeout,
	Summary:             "Read the work to look back over",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    5,
	},
}

var modelOptions = workflow.ActivityOptions{
	StartToCloseTimeout: modelTimeout,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	Summary:             "Decide what the work taught",
	RetryPolicy:         modelcall.RetryPolicy(3),
}

var finishOptions = workflow.ActivityOptions{
	StartToCloseTimeout: finishTimeout,
	Summary:             "Keep what the work taught",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    10,
	},
}

var failOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Record a look back that failed",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    5,
	},
}

func Workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        ThreadReflectionWorkflowName,
			Fn:          ThreadReflectionWorkflow,
			Description: "Look back over a conversation once it goes quiet and keep what the agent learned",
		},
		{
			Name:        RunReflectionWorkflowName,
			Fn:          RunReflectionWorkflow,
			Description: "Look back over a settled background run and keep what the agent learned",
		},
	}
}

func ThreadReflectionWorkflow(
	ctx workflow.Context,
	input *ThreadReflectionInput,
) (*ReflectionResult, error) {
	signals := workflow.GetSignalChannel(ctx, TurnFinishedSignalName)
	result := &ReflectionResult{}

	for {
		waitForQuiet(ctx, signals)

		var a *Activities
		var plan serviceports.ReflectionPlan
		err := workflow.ExecuteActivity(
			withPriority(ctx, prepareOptions, input.TenantInfo),
			a.PrepareThreadReflectionActivity, input.request(),
		).Get(ctx, &plan)
		if err != nil {
			workflow.GetLogger(ctx).Error("could not prepare a look back over a conversation",
				"threadId", input.ThreadID.String(),
				"error", err.Error(),
			)
			result.Failed++
		} else {
			lookBack(ctx, &plan, result)
		}
		result.Rounds++

		if !drain(signals) {
			return result, nil
		}
		input.Rounds++
		if input.Rounds >= RoundsPerRun {
			next := *input
			next.Rounds = 0

			return result, workflow.NewContinueAsNewError(ctx, ThreadReflectionWorkflow, &next)
		}
	}
}

func RunReflectionWorkflow(
	ctx workflow.Context,
	input *RunReflectionInput,
) (*ReflectionResult, error) {
	result := &ReflectionResult{Rounds: 1}

	var a *Activities
	var plan serviceports.ReflectionPlan
	err := workflow.ExecuteActivity(
		withPriority(ctx, prepareOptions, input.TenantInfo),
		a.PrepareRunReflectionActivity, &serviceports.ReflectOnRunRequest{
			TenantInfo: input.TenantInfo,
			RunID:      input.RunID,
		},
	).Get(ctx, &plan)
	if err != nil {
		return result, err
	}
	lookBack(ctx, &plan, result)

	return result, nil
}

func lookBack(ctx workflow.Context, plan *serviceports.ReflectionPlan, result *ReflectionResult) {
	if !plan.Ready() {
		result.Skipped++

		return
	}

	var a *Activities
	var completion serviceports.StructuredCompletionResult
	err := workflow.ExecuteActivity(
		withPriority(ctx, modelOptions, plan.TenantInfo),
		a.ReflectionModelActivity, plan,
	).Get(ctx, &completion)
	if err != nil {
		result.Failed++
		failure := workflow.ExecuteActivity(
			withPriority(ctx, failOptions, plan.TenantInfo),
			a.FailReflectionActivity, &serviceports.FailReflectionRequest{
				Plan:    plan,
				Message: err.Error(),
			},
		).Get(ctx, nil)
		if failure != nil {
			workflow.GetLogger(ctx).Error("could not record a look back that failed",
				"reflectionId", plan.ReflectionID.String(),
				"error", failure.Error(),
			)
		}

		return
	}

	var outcome serviceports.ReflectionOutcome
	err = workflow.ExecuteActivity(
		withPriority(ctx, finishOptions, plan.TenantInfo),
		a.FinishReflectionActivity, &FinishReflectionInput{Plan: plan, Result: &completion},
	).Get(ctx, &outcome)
	if err != nil {
		result.Failed++
		workflow.GetLogger(ctx).Error("could not keep what a look back learned",
			"reflectionId", plan.ReflectionID.String(),
			"error", err.Error(),
		)

		return
	}
	result.absorb(&outcome)
}

func waitForQuiet(ctx workflow.Context, signals workflow.ReceiveChannel) {
	deadline := workflow.Now(ctx).Add(LongestWait)
	drain(signals)

	for {
		wait := min(QuietPeriod, deadline.Sub(workflow.Now(ctx)))
		if wait <= 0 {
			return
		}

		timerCtx, cancel := workflow.WithCancel(ctx)
		heard := false
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
			var signal TurnFinished
			channel.Receive(ctx, &signal)
			heard = true
		})
		selector.AddFuture(workflow.NewTimer(timerCtx, wait), func(workflow.Future) {})
		selector.Select(ctx)
		cancel()

		if !heard {
			return
		}
		drain(signals)
	}
}

func drain(signals workflow.ReceiveChannel) bool {
	heard := false
	for {
		var signal TurnFinished
		if !signals.ReceiveAsync(&signal) {
			return heard
		}
		heard = true
	}
}

func withPriority(
	ctx workflow.Context,
	options workflow.ActivityOptions,
	tenant pagination.TenantInfo,
) workflow.Context {
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityBackground,
		FairnessKey: tenant.OrgID.String(),
	}

	return workflow.WithActivityOptions(ctx, options)
}
