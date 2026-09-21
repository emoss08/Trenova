package turnstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	fieldEvent = "event"
	fieldData  = "data"

	// startOfStream is the cursor a reader with no position uses. Redis reads
	// everything after it, which is everything.
	startOfStream = "0-0"

	// terminalTTL is how long a finished turn's events linger. Long enough for
	// a reader mid-reconnect to land and read the ending; short enough that
	// redis is not quietly storing conversations, which postgres already has.
	terminalTTL = 15 * time.Minute

	// readCount bounds one read. A turn that produced more frames than this
	// while a reader was away is drained over several reads rather than one
	// large one.
	readCount = 200
)

type Params struct {
	fx.In

	Redis  *redis.Client
	Config *config.Config
	Logger *zap.Logger
}

// Service publishes and replays a turn's events over one redis stream per
// turn.
//
// A stream rather than pub/sub, for one reason that decides everything else:
// its entry ids are monotonic and its ranges are exclusive, so the id handed
// to a browser as an SSE event id comes back as Last-Event-ID and is already a
// cursor this side understands. Pub/sub would deliver the same events and give
// a reconnecting reader nothing.
//
// No consumer group, either. Two people may watch one turn from a shared desk,
// and a group would hand each frame to exactly one of them.
type Service struct {
	redis  *redis.Client
	ai     *config.AIConfig
	logger *zap.Logger
}

var (
	_ serviceports.TurnStreamPublisher = (*Service)(nil)
	_ serviceports.TurnStreamReader    = (*Service)(nil)
)

func New(p Params) *Service {
	return &Service{
		redis:  p.Redis,
		ai:     p.Config.GetAIConfig(),
		logger: p.Logger.Named("turnstream"),
	}
}

func (s *Service) key(ref serviceports.TurnStreamRef) string {
	return fmt.Sprintf("%s:%s:%s",
		s.ai.GetTurnStreamKeyPrefix(),
		ref.TenantInfo.OrgID.String(),
		ref.TurnID.String(),
	)
}

func (s *Service) Publish(
	ctx context.Context,
	ref serviceports.TurnStreamRef,
	event serviceports.StreamEvent,
) error {
	return s.publish(ctx, ref, event, s.ai.GetTurnStreamTTL())
}

// Close writes the last frame and starts the clock on what is left.
func (s *Service) Close(
	ctx context.Context,
	ref serviceports.TurnStreamRef,
	event serviceports.StreamEvent,
) error {
	return s.publish(ctx, ref, event, terminalTTL)
}

func (s *Service) publish(
	ctx context.Context,
	ref serviceports.TurnStreamRef,
	event serviceports.StreamEvent,
	ttl time.Duration,
) error {
	encoded, err := sonic.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("encode %q event: %w", event.Event, err)
	}

	key := s.key(ref)
	// The add and the expiry go together so a stream cannot outlive its
	// window because the second call was the one that failed.
	_, err = s.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: key,
			MaxLen: s.ai.GetTurnStreamMaxLen(),
			Approx: true,
			Values: map[string]any{fieldEvent: event.Event, fieldData: encoded},
		})
		pipe.Expire(ctx, key, ttl)

		return nil
	})
	if err != nil {
		return fmt.Errorf("publish %q event: %w", event.Event, err)
	}

	return nil
}

func (s *Service) Exists(
	ctx context.Context,
	ref serviceports.TurnStreamRef,
) (bool, error) {
	count, err := s.redis.Exists(ctx, s.key(ref)).Result()
	if err != nil {
		return false, fmt.Errorf("check turn stream: %w", err)
	}

	return count > 0, nil
}

// Read replays what the reader missed and then follows the turn.
//
// It returns when the turn ends, when ctx is done, or when onFrame refuses a
// frame. A turn that goes quiet is not an ending: the read simply blocks
// again, and the caller decides how long silence is allowed to mean nothing —
// it is the one that knows whether the workflow is still alive.
func (s *Service) Read(
	ctx context.Context,
	req serviceports.ReadTurnStreamRequest,
) error {
	key := s.key(req.Ref)
	cursor := req.Cursor
	if cursor == "" {
		cursor = startOfStream
	}

	block := s.ai.GetTurnStreamBlockTimeout()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// The wait lives in BLOCK rather than in a sleep on purpose: go-redis
		// extends the connection's read timeout for a blocking command by the
		// block duration, and a hand-rolled wait would be severed by the
		// five-second default instead.
		streams, err := s.redis.XRead(ctx, &redis.XReadArgs{
			Streams: []string{key, cursor},
			Count:   readCount,
			Block:   block,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// Nothing arrived in the window. Handing control back lets the
				// caller write its keepalive and ask whether the turn is still
				// running, which is the only way a worker that died without
				// closing its stream is ever noticed.
				if req.OnIdle != nil {
					if idleErr := req.OnIdle(); idleErr != nil {
						return idleErr
					}
				}

				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return fmt.Errorf("read turn stream: %w", err)
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				frame := toFrame(message)
				cursor = frame.ID

				if err = req.OnFrame(frame); err != nil {
					return err
				}
				if frame.Terminal() {
					return nil
				}
			}
		}
	}
}

func toFrame(message redis.XMessage) serviceports.TurnStreamFrame {
	frame := serviceports.TurnStreamFrame{ID: message.ID}
	if name, ok := message.Values[fieldEvent].(string); ok {
		frame.Event = name
	}
	if data, ok := message.Values[fieldData].(string); ok {
		frame.Data = []byte(data)
	}

	return frame
}
