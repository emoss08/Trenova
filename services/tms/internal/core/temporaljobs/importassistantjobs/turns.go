package importassistantjobs

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentimportassistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type TurnsParams struct {
	fx.In

	Workflows serviceports.WorkflowStarter
	Reader    serviceports.TurnStreamReader
}

// Turns hands an import assistant message to a worker, and either waits for
// the whole reply or relays it to the reader as it is written.
type Turns struct {
	workflows serviceports.WorkflowStarter
	reader    serviceports.TurnStreamReader
}

var _ serviceports.ShipmentImportTurns = (*Turns)(nil)

func NewTurns(p TurnsParams) *Turns {
	return &Turns{workflows: p.Workflows, reader: p.Reader}
}

// WorkflowIDFor names a document's turn. One document answers one message at
// a time: a second sent while the first is answered is refused, as the
// conversation would otherwise interleave two replies.
func WorkflowIDFor(documentID string) string {
	return "import-assistant/" + documentID
}

// Chat answers one message and returns the whole reply.
func (t *Turns) Chat(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*serviceports.ShipmentImportChatResponse, error) {
	run, err := t.start(ctx, req, false)
	if err != nil {
		return nil, err
	}

	var response serviceports.ShipmentImportChatResponse
	if err = run.Get(ctx, &response); err != nil {
		return nil, turnError(err)
	}

	return &response, nil
}

// ChatStream answers one message and hands its events to emit as they are
// written, ending with done or error.
func (t *Turns) ChatStream(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	emit func(serviceports.StreamEvent),
) error {
	run, err := t.start(ctx, req, true)
	if err != nil {
		return err
	}

	ended := false
	if err = t.reader.Read(ctx, serviceports.ReadTurnStreamRequest{
		Ref: serviceports.TurnStreamRef{TenantInfo: req.TenantInfo, WorkflowID: run.GetID()},
		OnFrame: func(frame serviceports.TurnStreamFrame) error {
			ended = ended || frame.Terminal()
			emit(serviceports.StreamEvent{
				Event: frame.Event,
				Data:  sonic.NoCopyRawMessage(frame.Data),
			})

			return nil
		},
	}); err != nil || ended {
		return err
	}

	// The turn closed before the reader caught up with it. Its result is
	// how it ended.
	var response serviceports.ShipmentImportChatResponse
	if err = run.Get(ctx, &response); err != nil {
		return turnError(err)
	}
	emit(serviceports.StreamEvent{
		Event: shipmentimportassistantservice.EventDone,
		Data: map[string]any{
			"conversationId": response.ConversationID,
			"actions":        response.Actions,
		},
	})

	return nil
}

func (t *Turns) start(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	stream bool,
) (client.WorkflowRun, error) {
	run, err := t.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       WorkflowIDFor(req.DocumentID),
		TaskQueue:                                temporaltype.TaskQueueAgentChat.String(),
		WorkflowExecutionTimeout:                 turnTimeout,
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy:                 enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            "Import assistant turn",
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityInteractive,
			FairnessKey: req.TenantInfo.OrgID.String(),
		},
	}, ImportAssistantTurnWorkflowName, &TurnPayload{
		TenantInfo: req.TenantInfo,
		Request:    req,
		Stream:     stream,
	})
	if err != nil {
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &started) {
			return nil, errortypes.NewBusinessError(
				"The assistant is still answering the last message about this document.",
			)
		}

		return nil, err
	}

	return run, nil
}

// turnError is how a turn that ended without a reply is reported: in the
// words it wrote for the person, or as a timeout.
func turnError(err error) error {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == errTypeTurnFailed {
		return errortypes.NewBusinessError(appErr.Message())
	}

	var timeout *temporal.TimeoutError
	if errors.As(err, &timeout) {
		return errortypes.NewBusinessError("The AI request timed out. Please try again.")
	}

	return err
}
