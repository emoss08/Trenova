package importassistantjobs

import (
	"context"
	"errors"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentimportassistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Assistant  *shipmentimportassistantservice.Service
	Completion serviceports.CompletionService
	Logger     *zap.Logger
}

type Activities struct {
	assistant  *shipmentimportassistantservice.Service
	completion serviceports.CompletionService
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		assistant:  p.Assistant,
		completion: p.Completion,
		l:          p.Logger.Named("import-assistant-jobs"),
	}
}

// PrepareImportTurnActivity opens the document's conversation and builds the
// turn's first request. A request that is wrong stays wrong however often it
// is sent, so it is not retried.
func (a *Activities) PrepareImportTurnActivity(
	ctx context.Context,
	payload *TurnPayload,
) (*shipmentimportassistantservice.PreparedTurn, error) {
	prepared, err := a.assistant.PrepareTurn(ctx, payload.request())
	if err != nil {
		var invalid *errortypes.Error
		if errortypes.IsBusinessError(err) || errors.As(err, &invalid) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(), errTypeTurnFailed, err,
			)
		}

		return nil, err
	}

	return prepared, nil
}

// ImportModelCallActivity asks the model once. A reply somebody is reading is
// published as it arrives; a retried attempt, or a reply the router starts
// over on another provider, opens a fresh message so the two do not run
// together.
func (a *Activities) ImportModelCallActivity(
	ctx context.Context,
	in *ModelInput,
) (*serviceports.ChatCompletionResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	req := *in.Request
	if !in.Stream {
		result, err := a.completion.CompleteChat(ctx, &req)
		if err != nil {
			return nil, modelcall.Classify(err)
		}

		return result, nil
	}

	stream, events, err := agentflow.OpenStream(ctx)
	if err != nil {
		return nil, err
	}
	defer agentflow.CloseStream(ctx, stream, a.l)

	startOver := func() {
		events.Publish(temporaltype.StreamItem{
			Event: shipmentimportassistantservice.EventNewMessage,
		}, true)
	}
	if activity.GetInfo(ctx).Attempt > 1 {
		startOver()
	}
	req.RetrySink = func(notice serviceports.ChatRetryNotice) {
		if notice.Kind == serviceports.RetryKindRestart {
			startOver()
		}
	}

	result, err := a.completion.StreamChat(ctx, &req, func(delta string) {
		events.Publish(temporaltype.StreamItem{
			Event: shipmentimportassistantservice.EventTextDelta,
			Data:  map[string]string{"delta": delta},
		}, false)
	})
	if err != nil {
		return nil, modelcall.Classify(err)
	}

	return result, nil
}

// ImportToolActivity runs one tool call. A tool reports its own failure in
// what it returns, so the activity fails only when it could not run at all.
func (a *Activities) ImportToolActivity(
	ctx context.Context,
	in *ToolInput,
) (*shipmentimportassistantservice.ToolOutcome, error) {
	outcome := a.assistant.RunTool(ctx, in.TenantInfo, &in.Call)

	return &outcome, nil
}

// FinishImportTurnActivity saves a turn that finished and returns the reply.
func (a *Activities) FinishImportTurnActivity(
	ctx context.Context,
	in *FinishInput,
) (*serviceports.ShipmentImportChatResponse, error) {
	return a.assistant.FinishTurn(ctx, in.Payload.request(), in.Record)
}

// FailImportTurnActivity saves a turn that did not finish and returns what the
// person is told.
func (a *Activities) FailImportTurnActivity(ctx context.Context, in *FinishInput) (string, error) {
	return a.assistant.FailTurn(ctx, in.Payload.request(), in.Record, in.Failure.Err()), nil
}
