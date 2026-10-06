package turnstream

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// releaseTimeout bounds telling a turn its reader has the last event. It rides
// a context the reader leaving cannot cancel, so it needs a bound of its own.
const releaseTimeout = 5 * time.Second

// pollCooldown is how long the reader rests after a delivered batch before it
// asks again. The library's default is 100ms, which on top of the publisher's
// own batching turned a reply into clumps. A poll waits on the workflow until
// there is something to return, so a short rest costs no extra polls; zero
// would mean the default, so it is small rather than nothing.
const pollCooldown = 5 * time.Millisecond

type Params struct {
	fx.In

	Client client.Client
	Logger *zap.Logger
}

// Service reads a turn's events from the Workflow Stream its workflow hosts.
//
// Each event's offset in the stream is its SSE event id, so a browser's
// Last-Event-ID comes back as a position the stream already understands, and
// a reader who lost the connection resumes after the last event it applied.
// Any number of readers can follow one turn: reading takes nothing from the
// stream.
type Service struct {
	client client.Client
	logger *zap.Logger
}

var _ serviceports.TurnStreamReader = (*Service)(nil)

func New(p Params) *Service {
	return &Service{client: p.Client, logger: p.Logger.Named("turnstream")}
}

// frame is an event as the workflow published it. The payload is kept as it
// arrived: the relay is a pipe, and nothing between the worker and the reader
// needs to understand what a delta says.
type frame struct {
	Event string                 `json:"event"`
	Data  sonic.NoCopyRawMessage `json:"data"`
}

func (s *Service) Read(ctx context.Context, req serviceports.ReadTurnStreamRequest) error {
	stream := workflowstreams.NewClient(s.client, req.Ref.WorkflowID, workflowstreams.Options{})

	for item, err := range stream.Subscribe(ctx, subscription(req.Cursor)) {
		if err != nil {
			return err
		}

		var decoded frame
		if err = sonic.Unmarshal(item.Data.GetData(), &decoded); err != nil {
			// A frame this side cannot read would be one the reader cannot
			// either. Skipping it keeps the rest of the turn flowing.
			s.logger.Warn("skipped a turn event that could not be read",
				zap.String("workflow", req.Ref.WorkflowID),
				zap.Int64("offset", item.Offset),
				zap.Error(err),
			)
			continue
		}

		out := serviceports.TurnStreamFrame{
			ID:    strconv.FormatInt(item.Offset, 10),
			Event: decoded.Event,
			Data:  decoded.Data,
		}
		if err = req.OnFrame(out); err != nil {
			return err
		}
		if out.Terminal() {
			s.release(ctx, req.Ref.WorkflowID)

			return nil
		}
	}

	return nil
}

// subscription is how a reader follows a turn's events from its cursor.
func subscription(cursor string) workflowstreams.SubscribeOptions {
	return workflowstreams.SubscribeOptions{
		Topics:       []string{temporaltype.StreamEventsTopic},
		FromOffset:   after(cursor),
		PollCooldown: pollCooldown,
	}
}

// after is where a reader resumes: the event after the last one it applied.
// A cursor that is not an offset, such as one left by the stream this
// replaced, starts the turn over, which a reader handles by refetching.
func after(cursor string) int64 {
	if cursor == "" {
		return 0
	}

	offset, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || offset < 0 {
		return 0
	}

	return offset + 1
}

// release tells the turn its reader has the last event, so it can close now
// rather than at the end of its drain window.
func (s *Service) release(ctx context.Context, workflowID string) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
	defer cancel()

	err := s.client.SignalWorkflow(
		releaseCtx,
		workflowID,
		"",
		temporaltype.SignalStreamDrained,
		nil,
	)
	var gone *serviceerror.NotFound
	if err != nil && !errors.As(err, &gone) {
		s.logger.Warn("could not tell a turn its reader is done",
			zap.String("workflow", workflowID),
			zap.Error(err),
		)
	}
}
