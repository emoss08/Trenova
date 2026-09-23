package importassistantjobs

import (
	"strings"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	assistant "github.com/emoss08/trenova/internal/core/services/shipmentimportassistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// failedMessage is what the person is told when even recording the failure
// failed.
const failedMessage = "AI assistant encountered an error. Please try again."

// ImportAssistantTurnWorkflow answers one message to the import assistant.
//
// The loop runs here, so a model call or a tool call that fails is retried on
// its own rather than taking the turn with it, and a lost worker resumes the
// turn from its last completed step. A reader follows the reply on the stream
// the turn hosts; a caller waiting for the whole reply gets it as the result.
func ImportAssistantTurnWorkflow(
	ctx workflow.Context,
	payload *TurnPayload,
) (*serviceports.ShipmentImportChatResponse, error) {
	stream, err := agentflow.HostStream(ctx)
	if err != nil {
		return nil, err
	}

	turn := &turn{ctx: ctx, stream: stream, payload: payload, record: &assistant.TurnRecord{}}
	response, failure := turn.converse()

	keep, _ := workflow.NewDisconnectedContext(ctx)
	if failure == nil {
		turn.emit(keep, assistant.EventDone, map[string]any{
			"conversationId": response.ConversationID,
			"actions":        response.Actions,
		})
		turn.close(keep)

		return response, nil
	}

	var a *Activities
	message := failedMessage
	saveCtx := workflow.WithActivityOptions(keep, turn.options(finishTimeout, saveAttempts))
	if err = workflow.ExecuteActivity(saveCtx, a.FailImportTurnActivity, &FinishInput{
		Payload: payload,
		Record:  turn.record,
		Failure: failure,
	}).Get(saveCtx, &message); err != nil {
		workflow.GetLogger(ctx).Error("a failed import turn could not be recorded",
			"documentId", payload.Request.DocumentID,
			"error", err.Error(),
		)
	}

	turn.emit(keep, assistant.EventError, map[string]string{"message": message})
	turn.close(keep)

	return nil, temporal.NewNonRetryableApplicationError(message, errTypeTurnFailed, nil)
}

type turn struct {
	ctx     workflow.Context
	stream  *agentflow.Stream
	payload *TurnPayload
	record  *assistant.TurnRecord
}

// converse runs the turn to its reply. Every failure is returned as data, for
// the turn's record and for the person.
func (t *turn) converse() (*serviceports.ShipmentImportChatResponse, *modelcall.Failure) {
	var a *Activities

	prepareCtx := workflow.WithActivityOptions(t.ctx, t.options(prepareTimeout, readAttempts))
	var prepared assistant.PreparedTurn
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareImportTurnActivity, t.payload).
		Get(prepareCtx, &prepared); err != nil {
		return nil, modelcall.FailureOf(err)
	}

	messages := prepared.Messages
	modelCtx := workflow.WithActivityOptions(t.ctx, t.modelOptions())
	for round := range assistant.MaxToolRounds {
		// A round after the first is a fresh answer, not a continuation of
		// the message the reader is already looking at.
		if round > 0 {
			t.emit(t.ctx, assistant.EventNewMessage, nil)
		}

		var result serviceports.ChatCompletionResult
		if err := workflow.ExecuteActivity(modelCtx, a.ImportModelCallActivity, &ModelInput{
			Request: &serviceports.ChatCompletionRequest{
				TenantInfo: t.payload.TenantInfo,
				System:     prepared.System,
				Messages:   messages,
				Tools:      prepared.Tools,
			},
			Stream: t.payload.Stream,
		}).Get(modelCtx, &result); err != nil {
			return nil, modelcall.FailureOf(err)
		}

		t.record.Model = result.ModelIdentifier
		if strings.TrimSpace(result.Text) != "" {
			t.record.Message += result.Text
		}
		if len(result.ToolCalls) == 0 {
			break
		}

		messages = append(
			messages,
			assistant.ToolRound(result.ToolCalls, t.runTools(result.ToolCalls))...)
	}

	if suggestions := assistant.NormalizeSuggestions(t.record.Suggestions); len(suggestions) > 0 {
		t.emit(t.ctx, assistant.EventSuggestions, map[string]any{"suggestions": suggestions})
	}

	finishCtx := workflow.WithActivityOptions(t.ctx, t.options(finishTimeout, saveAttempts))
	var response serviceports.ShipmentImportChatResponse
	if err := workflow.ExecuteActivity(finishCtx, a.FinishImportTurnActivity, &FinishInput{
		Payload: t.payload,
		Record:  t.record,
	}).Get(finishCtx, &response); err != nil {
		return nil, modelcall.FailureOf(err)
	}

	return &response, nil
}

// runTools runs one round's calls in the order the model made them, each an
// activity of its own. A call that could not be run at all is reported to the
// model as such, and the turn goes on.
func (t *turn) runTools(calls []serviceports.ToolCall) map[string]assistant.ToolOutcome {
	var a *Activities
	outcomes := make(map[string]assistant.ToolOutcome, len(calls))

	for _, call := range calls {
		if call.Name == assistant.SuggestQuickActionsTool {
			t.record.Suggestions = assistant.ReadSuggestions(call.Arguments)
			outcomes[call.ID] = assistant.SuggestionsOutcome()

			continue
		}

		t.emit(t.ctx, assistant.EventToolCallStart, map[string]string{
			"name":   call.Name,
			"callId": call.ID,
		})

		attempts := int32(readAttempts)
		if assistant.WritesTool(call.Name) {
			attempts = writeAttempts
		}
		toolCtx := workflow.WithActivityOptions(t.ctx,
			withSummary(t.options(toolTimeout, attempts), call.Name))

		var outcome assistant.ToolOutcome
		if err := workflow.ExecuteActivity(toolCtx, a.ImportToolActivity, &ToolInput{
			TenantInfo: t.payload.TenantInfo,
			Call:       call,
		}).Get(toolCtx, &outcome); err != nil {
			workflow.GetLogger(t.ctx).Warn("an import assistant tool could not be run",
				"tool", call.Name,
				"error", err.Error(),
			)
			outcome = assistant.UnavailableToolOutcome()
		}

		outcomes[call.ID] = outcome
		t.record.Actions = append(t.record.Actions, outcome.Actions...)
		t.record.ToolCalls = append(t.record.ToolCalls, assistant.ToolRecord(&call, outcome))

		t.emit(t.ctx, assistant.EventToolCallDone, map[string]any{
			"name":    call.Name,
			"callId":  call.ID,
			"status":  outcome.Status,
			"result":  outcome.Output,
			"actions": outcome.Actions,
		})
	}

	return outcomes
}

// emit publishes an event for a reader, when there is one. A turn nobody
// reads as it is written publishes nothing: every event is a signal in the
// turn's history.
func (t *turn) emit(ctx workflow.Context, event string, data any) {
	if t.payload.Stream {
		t.stream.Publish(ctx, temporaltype.StreamItem{Event: event, Data: data})
	}
}

// close hands the last events to the reader, when there is one. A turn nobody
// reads has nobody to wait for.
func (t *turn) close(ctx workflow.Context) {
	if t.payload.Stream {
		t.stream.Close(ctx)
	}
}

func (t *turn) options(timeout time.Duration, attempts int32) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        10 * time.Second,
			MaximumAttempts:        attempts,
			NonRetryableErrorTypes: []string{errTypeTurnFailed},
		},
		Priority: t.priority(),
	}
}

func (t *turn) modelOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: modelTimeout,
		HeartbeatTimeout:    modelcall.HeartbeatTimeout,
		RetryPolicy:         modelcall.RetryPolicy(modelAttempts),
		Summary:             "Ask the model",
		Priority:            t.priority(),
	}
}

func (t *turn) priority() temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflow.PriorityInteractive,
		FairnessKey: t.payload.TenantInfo.OrgID.String(),
	}
}

func withSummary(options workflow.ActivityOptions, summary string) workflow.ActivityOptions {
	options.Summary = summary

	return options
}
