package assistantjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// turnTimeout bounds one answer. It is generous because a reasoning model
	// with a few tool calls legitimately takes minutes, and severing one of
	// those is the fault this work exists to stop repeating.
	turnTimeout = 15 * time.Minute
	// heartbeatTimeout is how long a turn may go without reporting progress.
	// The activity beats on every event, and the longest ordinary gap is the
	// silence before a model's first token.
	heartbeatTimeout = 2 * time.Minute
)

// turnRetry is deliberately narrow.
//
// A retried turn does not resume: it would ask the model again, and the claim
// the activity takes means the second attempt declines to answer rather than
// producing a duplicate reply. So a retry is only worth anything for a failure
// that happened before any of that — a worker that never picked the activity
// up, a transient fault reaching the database. Two attempts covers those and
// nothing more.
var turnRetry = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    10 * time.Second,
	MaximumAttempts:    2,
	NonRetryableErrorTypes: []string{
		errTypeTurnNotFound,
		errTypeNoActor,
		errTypeRefused,
	},
}

var turnActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: turnTimeout,
	HeartbeatTimeout:    heartbeatTimeout,
	RetryPolicy:         turnRetry,
	// A turn is stopped by cancelling its workflow, and the activity has to be
	// told rather than merely abandoned: it is holding a model connection and
	// writing a transcript, and both want closing properly.
	WaitForCancellation: true,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AssistantTurnWorkflowName,
			Fn:          AssistantTurnWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgentChat.String(),
			Description: "Answer one question in a conversation, publishing the reply as it is written",
		},
	}
}

// AssistantTurnWorkflow answers one question.
//
// It is one activity, not a step per tool call. Every activity boundary costs a
// Temporal round trip and a payload through the encrypting data converter, and
// a model's loop is not made more deterministic by being cut into pieces: the
// same prompt produces different calls on a replay regardless. The durability
// that matters is at the turn's edges, and the writes inside it are guarded by
// the step ledger rather than by the workflow's history.
func AssistantTurnWorkflow(
	ctx workflow.Context,
	payload *AssistantTurnPayload,
) (*AssistantTurnResult, error) {
	var a *Activities

	runCtx := workflow.WithActivityOptions(ctx, turnActivityOptions)

	var result AssistantTurnResult
	err := workflow.ExecuteActivity(runCtx, a.AssistantTurnActivity, payload).
		Get(runCtx, &result)
	if err != nil {
		// The activity closes the turn's record and its stream on every path
		// it controls, including cancellation, so there is nothing to undo
		// here. Returning the error is what marks the execution failed for
		// anybody reading Temporal rather than the database.
		workflow.GetLogger(ctx).Error("assistant turn failed",
			"turnId", payload.TurnID.String(),
			"error", err.Error(),
		)

		return nil, err
	}

	return &result, nil
}
