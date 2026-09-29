package extractionshadowjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const unfinishedMessage = "The shadow extraction could not be run or saved"

var runOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 20 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    runAttempts,
		MaximumInterval:    time.Minute,
		NonRetryableErrorTypes: []string{
			temporaltype.ErrorTypeInvalidInput.String(),
		},
	},
}

var failOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    5,
		MaximumInterval:    30 * time.Second,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ExtractionShadowWorkflowName,
			Fn:          ExtractionShadowWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Run a production document extraction again on the shadow AI provider and keep its answer for scoring",
		},
	}
}

func shadowPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflow.PriorityEvaluation,
		FairnessKey: orgID.String(),
	}
}

func withPriority(options *workflow.ActivityOptions, orgID pulid.ID) workflow.ActivityOptions {
	prioritized := *options
	prioritized.Priority = shadowPriority(orgID)
	return prioritized
}

func ExtractionShadowWorkflow(ctx workflow.Context, payload *ShadowPayload) error {
	runCtx := workflow.WithActivityOptions(ctx, withPriority(&runOptions, payload.OrganizationID))

	var a *Activities
	err := workflow.ExecuteActivity(runCtx, a.RunExtractionShadowActivity, payload).Get(runCtx, nil)
	if err == nil {
		return nil
	}

	failCtx := workflow.WithActivityOptions(ctx, withPriority(&failOptions, payload.OrganizationID))
	if failErr := workflow.ExecuteActivity(failCtx, a.FailExtractionShadowActivity, &FailInput{
		ShadowPayload: *payload,
		Message:       unfinishedMessage,
	}).Get(failCtx, nil); failErr != nil {
		workflow.GetLogger(ctx).Error("could not mark the shadow extraction failed",
			"resultId", payload.ResultID.String(),
			"error", failErr,
		)
	}

	return err
}
