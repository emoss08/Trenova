package resultcache

import (
	"context"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// StreamName is the redis stream GTC's reporting projections write to.
	// Registering a reportable table for cache invalidation is config-only on
	// the GTC side: add a redis_stream projection with `stream: reporting:cdc`.
	StreamName    = "reporting:cdc"
	consumerGroup = "report-datav"
	consumerName  = "report-datav-consumer"
	readCount     = 200
	blockTimeout  = 5 * time.Second
)

type ConsumerParams struct {
	fx.In

	LC     fx.Lifecycle
	Redis  *redis.Client
	Logger *zap.Logger
}

// Consumer tails GTC CDC events for reportable tables and advances per-org
// per-table data-version counters, invalidating cached report results.
type Consumer struct {
	redis  *redis.Client
	l      *zap.Logger
	cancel context.CancelFunc
	done   chan struct{}
}

func NewConsumer(p ConsumerParams) *Consumer {
	consumer := &Consumer{
		redis: p.Redis,
		l:     p.Logger.Named("reporting.datav-consumer"),
		done:  make(chan struct{}),
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return consumer.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			consumer.stop(ctx)
			return nil
		},
	})

	return consumer
}

func (c *Consumer) start(ctx context.Context) error {
	err := c.redis.XGroupCreateMkStream(ctx, StreamName, consumerGroup, "$").Err()
	if err != nil && !isBusyGroupError(err) {
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.consumeLoop(runCtx)

	return nil
}

func (c *Consumer) stop(ctx context.Context) {
	if c.cancel != nil {
		c.cancel()
	}
	select {
	case <-c.done:
	case <-ctx.Done():
	}
}

func (c *Consumer) consumeLoop(ctx context.Context) {
	defer close(c.done)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{StreamName, ">"},
			Count:    readCount,
			Block:    blockTimeout,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			c.l.Error("failed to read reporting CDC stream", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

type cdcPayload struct {
	Table   string         `json:"table"`
	NewData map[string]any `json:"new_data"`
	OldData map[string]any `json:"old_data"`
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	defer func() {
		if err := c.redis.XAck(ctx, StreamName, consumerGroup, msg.ID).Err(); err != nil {
			c.l.Warn("failed to ack reporting CDC message",
				zap.String("messageId", msg.ID), zap.Error(err))
		}
	}()

	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return
	}

	var payload cdcPayload
	if err := sonic.Unmarshal([]byte(raw), &payload); err != nil {
		c.l.Warn("malformed reporting CDC payload",
			zap.String("messageId", msg.ID), zap.Error(err))
		return
	}
	if payload.Table == "" {
		return
	}

	orgID := extractOrgID(payload.NewData)
	if orgID == "" {
		orgID = extractOrgID(payload.OldData)
	}
	if orgID == "" {
		return
	}

	if err := BumpDataVersion(ctx, c.redis, orgID, payload.Table); err != nil {
		c.l.Warn("failed to bump report data version",
			zap.String("table", payload.Table), zap.Error(err))
	}
}

func extractOrgID(data map[string]any) string {
	if data == nil {
		return ""
	}
	if orgID, ok := data["organization_id"].(string); ok {
		return orgID
	}
	return ""
}

func isBusyGroupError(err error) bool {
	return err != nil && errors.Is(err, redis.Nil) ||
		err != nil && containsBusyGroup(err.Error())
}

func containsBusyGroup(message string) bool {
	return len(message) >= 9 && message[:9] == "BUSYGROUP"
}

var Module = fx.Module("report-result-cache",
	fx.Provide(
		fx.Annotate(
			New,
			fx.As(new(services.ReportResultCache)),
		),
	),
	fx.Invoke(NewConsumer),
)
