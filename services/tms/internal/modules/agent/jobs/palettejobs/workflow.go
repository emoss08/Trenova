package palettejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/runstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	failedMessage  = "Nova couldn't answer that just now. Try again, or open it in the Desk."
	stoppedMessage = "Stopped."
)

var answerOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	WaitForCancellation: true,
	// The answer has already streamed to the person as it was written; a
	// second attempt would write it again under the first.
	RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
}

var finishOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
}

// FinishInput records how a question ended.
type FinishInput struct {
	palettecontract.AnswerInput

	Status palettequestion.Status `json:"status"`
}

func Workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{{
		Name:        palettecontract.AnswerWorkflowName,
		Fn:          PaletteAnswerWorkflow,
		Description: "Nova answers a question typed into the command palette",
	}}
}

// PaletteAnswerWorkflow hosts the answer's stream and answers the question.
// Its result is the answer, which "Open in Desk" carries into a conversation.
func PaletteAnswerWorkflow(
	ctx workflow.Context,
	input *palettecontract.AnswerInput,
) (*paletteports.Answer, error) {
	stream, err := runstream.Host(ctx)
	if err != nil {
		return nil, err
	}

	var a *Activities
	actx := workflow.WithActivityOptions(ctx, answerOptions)
	var answer paletteports.Answer
	err = workflow.ExecuteActivity(actx, a.AnswerActivity, input).Get(actx, &answer)

	// A stopped question still closes its stream, so the reader is told
	// rather than left waiting.
	closing, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	status := palettequestion.StatusAnswered
	if err != nil {
		message := failedMessage
		status = palettequestion.StatusFailed
		if temporal.IsCanceledError(err) {
			message = stoppedMessage
			status = palettequestion.StatusStopped
		}
		answer = paletteports.Answer{Question: input.Question, Failure: message}
		stream.Publish(closing, temporaltype.StreamItem{
			Event: assistantstreamports.AssistantEventError,
			Data:  map[string]string{"message": message},
			At:    workflow.Now(closing).Unix(),
		})
	} else if answer.Failure != "" {
		status = palettequestion.StatusFailed
	}

	finishCtx := workflow.WithActivityOptions(closing, finishOptions)
	if finishErr := workflow.ExecuteActivity(finishCtx, a.FinishAnswerActivity, &FinishInput{
		AnswerInput: *input,
		Status:      status,
	}).Get(finishCtx, nil); finishErr != nil {
		workflow.GetLogger(ctx).Warn("could not record how a palette question ended",
			"error", finishErr.Error())
	}
	stream.Close(closing)

	return &answer, nil
}
