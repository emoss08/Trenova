package tablechangealertservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	notificationdomain "github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	tcaStreamName    = "tca:events"
	tcaConsumerGroup = "tca-consumer-group"
	tcaConsumerName  = "tca-consumer"
	tcaReadCount     = 10
	tcaBlockTimeout  = 5 * time.Second
	tcaTrimInterval  = 5 * time.Minute
	tcaMaxStreamLen  = 10000
	tcaMaxRetries    = 3
	tcaPendingIdle   = 30 * time.Second
	tcaRetryInterval = 10 * time.Second
)

type tcaEvent struct {
	Projection    string         `json:"projection"`
	Operation     string         `json:"operation"`
	Schema        string         `json:"schema"`
	Table         string         `json:"table"`
	NewData       map[string]any `json:"new_data"`
	OldData       map[string]any `json:"old_data"`
	ChangedFields []string       `json:"changed_fields"`
	Metadata      struct {
		LSN       string `json:"LSN"`
		Timestamp string `json:"Timestamp"`
	} `json:"metadata"`
}

type ConsumerParams struct {
	fx.In

	Logger       *zap.Logger
	Redis        *redis.Client
	SubRepo      repositories.TCASubscriptionRepository
	NotifService *notificationservice.Service
}

type Consumer struct {
	l            *zap.Logger
	redis        *redis.Client
	subRepo      repositories.TCASubscriptionRepository
	notifService *notificationservice.Service
	cancel       context.CancelFunc
}

func NewConsumer(p ConsumerParams) *Consumer {
	return &Consumer{
		l:            p.Logger.Named("tca.consumer"),
		redis:        p.Redis,
		subRepo:      p.SubRepo,
		notifService: p.NotifService,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.ensureConsumerGroup(ctx)

	go c.recoverPending(ctx)
	go c.consumeLoop(ctx)
	go c.retryLoop(ctx)
	go c.trimLoop(ctx)

	c.l.Info("TCA consumer started")
}

func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.l.Info("TCA consumer stopped")
}

func (c *Consumer) ensureConsumerGroup(ctx context.Context) {
	err := c.redis.XGroupCreateMkStream(ctx, tcaStreamName, tcaConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		c.l.Warn("failed to create consumer group", zap.Error(err))
	}
}

func (c *Consumer) recoverPending(ctx context.Context) {
	pending, err := c.redis.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: tcaStreamName,
		Group:  tcaConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	if err != nil {
		c.l.Warn("failed to check pending messages", zap.Error(err))
		return
	}

	if len(pending) == 0 {
		return
	}

	c.l.Info("recovering pending messages", zap.Int("count", len(pending)))

	ids := make([]string, 0, len(pending))
	for _, p := range pending {
		ids = append(ids, p.ID)
	}

	claimed, err := c.redis.XClaim(ctx, &redis.XClaimArgs{
		Stream:   tcaStreamName,
		Group:    tcaConsumerGroup,
		Consumer: tcaConsumerName,
		MinIdle:  0,
		Messages: ids,
	}).Result()
	if err != nil {
		c.l.Error("failed to claim pending messages", zap.Error(err))
		return
	}

	for _, msg := range claimed {
		if err := c.processMessage(ctx, msg); err != nil {
			c.l.Error("failed to process recovered message",
				zap.String("messageID", msg.ID),
				zap.Error(err),
			)
			continue
		}
		c.redis.XAck(ctx, tcaStreamName, tcaConsumerGroup, msg.ID)
	}

	c.l.Info("pending message recovery complete",
		zap.Int("claimed", len(claimed)),
	)
}

func (c *Consumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    tcaConsumerGroup,
			Consumer: tcaConsumerName,
			Streams:  []string{tcaStreamName, ">"},
			Count:    tcaReadCount,
			Block:    tcaBlockTimeout,
		}).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			c.l.Error("failed to read from stream", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				if err := c.processMessage(ctx, msg); err != nil {
					c.l.Error("failed to process message",
						zap.String("messageID", msg.ID),
						zap.Error(err),
					)
					continue
				}

				c.redis.XAck(ctx, tcaStreamName, tcaConsumerGroup, msg.ID)
			}
		}
	}
}

func (c *Consumer) retryLoop(ctx context.Context) {
	ticker := time.NewTicker(tcaRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.retryPending(ctx)
		}
	}
}

func (c *Consumer) retryPending(ctx context.Context) {
	pending, err := c.redis.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: tcaStreamName,
		Group:  tcaConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  50,
		Idle:   tcaPendingIdle,
	}).Result()
	if err != nil || len(pending) == 0 {
		return
	}

	for _, p := range pending {
		if p.RetryCount >= tcaMaxRetries {
			c.l.Error("dropping message after max retries",
				zap.String("messageID", p.ID),
				zap.Int64("retryCount", p.RetryCount),
			)
			c.redis.XAck(ctx, tcaStreamName, tcaConsumerGroup, p.ID)
			continue
		}

		claimed, err := c.redis.XClaim(ctx, &redis.XClaimArgs{
			Stream:   tcaStreamName,
			Group:    tcaConsumerGroup,
			Consumer: tcaConsumerName,
			MinIdle:  tcaPendingIdle,
			Messages: []string{p.ID},
		}).Result()
		if err != nil || len(claimed) == 0 {
			continue
		}

		for _, msg := range claimed {
			if err := c.processMessage(ctx, msg); err != nil {
				c.l.Warn("retry failed",
					zap.String("messageID", msg.ID),
					zap.Int64("attempt", p.RetryCount+1),
					zap.Error(err),
				)
				continue
			}
			c.redis.XAck(ctx, tcaStreamName, tcaConsumerGroup, msg.ID)
		}
	}
}

func (c *Consumer) trimLoop(ctx context.Context) {
	ticker := time.NewTicker(tcaTrimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.redis.XTrimMaxLenApprox(ctx, tcaStreamName, tcaMaxStreamLen, 0).Err(); err != nil {
				c.l.Error("failed to trim stream", zap.Error(err))
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) error {
	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return fmt.Errorf("message payload is not a string")
	}

	var event tcaEvent
	if err := sonic.UnmarshalString(payloadStr, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	orgID := extractStringField(event.NewData, event.OldData, "organization_id")
	buID := extractStringField(event.NewData, event.OldData, "business_unit_id")
	if orgID == "" || buID == "" {
		c.l.Warn("skipping event: missing tenant fields",
			zap.String("table", event.Table),
			zap.String("operation", event.Operation),
		)
		return nil
	}

	recordID := extractStringField(event.NewData, event.OldData, "id")

	eventOrgID := pulid.ID(orgID)
	eventBuID := pulid.ID(buID)

	c.l.Debug("processing event",
		zap.String("table", event.Table),
		zap.String("operation", event.Operation),
		zap.String("orgID", orgID),
		zap.String("recordID", recordID),
	)

	subs, err := c.subRepo.FindMatchingSubscriptions(ctx, repositories.FindMatchingTCASubscriptionsRequest{
		OrganizationID: eventOrgID,
		BusinessUnitID: eventBuID,
		TableName:      event.Table,
		Operation:      event.Operation,
		RecordID:       recordID,
	})
	if err != nil {
		return fmt.Errorf("find matching subscriptions: %w", err)
	}

	if len(subs) == 0 {
		c.l.Debug("no matching subscriptions",
			zap.String("table", event.Table),
			zap.String("operation", event.Operation),
		)
		return nil
	}

	c.l.Info("matched subscriptions",
		zap.String("table", event.Table),
		zap.String("operation", event.Operation),
		zap.Int("count", len(subs)),
	)

	summary := buildSummary(event.Table, event.Operation, recordID, event.ChangedFields)

	for _, sub := range subs {
		if sub.OrganizationID != eventOrgID || sub.BusinessUnitID != eventBuID {
			c.l.Error("tenant mismatch in subscription match — skipping",
				zap.String("subscriptionID", sub.ID.String()),
				zap.String("eventOrg", orgID),
				zap.String("subOrg", sub.OrganizationID.String()),
			)
			continue
		}

		if len(sub.WatchedColumns) > 0 && event.Operation == "UPDATE" {
			if !hasWatchedColumnChanged(sub.WatchedColumns, event.ChangedFields) {
				continue
			}
		}

		if len(sub.Conditions) > 0 {
			if !EvaluateConditions(sub.Conditions, sub.ConditionMatch, event.NewData, event.OldData, event.ChangedFields) {
				continue
			}
		}

		title := summary
		message := summary
		if sub.CustomTitle != "" {
			title = RenderTemplate(sub.CustomTitle, event.Table, event.Operation, recordID, event.NewData, event.OldData, event.ChangedFields)
		}
		if sub.CustomMessage != "" {
			message = RenderTemplate(sub.CustomMessage, event.Table, event.Operation, recordID, event.NewData, event.OldData, event.ChangedFields)
		}

		priority := notificationdomain.PriorityMedium
		if sub.Priority != "" {
			priority = notificationdomain.Priority(sub.Priority)
		}

		buID := sub.BusinessUnitID
		notif := &notificationdomain.Notification{
			OrganizationID: sub.OrganizationID,
			BusinessUnitID: &buID,
			TargetUserID:   &sub.UserID,
			EventType:      "tca." + strings.ToLower(event.Operation),
			Priority:       priority,
			Channel:        notificationdomain.ChannelUser,
			Title:          title,
			Message:        message,
			Source:         "table_change_alert",
			DeliveryStatus: notificationdomain.DeliveryStatusPending,
			Data: map[string]any{
				"subscriptionId":   sub.ID.String(),
				"subscriptionName": sub.Name,
				"tableName":        event.Table,
				"operation":        event.Operation,
				"changedFields":    event.ChangedFields,
				"recordId":         recordID,
				"topic":            sub.Topic,
				"priority":         sub.Priority,
			},
		}

		if _, err := c.notifService.Create(ctx, notif); err != nil {
			c.l.Error("failed to create notification",
				zap.String("subscriptionID", sub.ID.String()),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

func extractStringField(newData, oldData map[string]any, field string) string {
	if v, ok := newData[field]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v, ok := oldData[field]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func buildSummary(table, operation, recordID string, changedFields []string) string {
	var sb strings.Builder
	caser := cases.Title(language.English)
	sb.WriteString(caser.String(strings.ToLower(table)))

	if recordID != "" {
		sb.WriteString(" ")
		sb.WriteString(recordID)
	}

	sb.WriteString(" was ")

	switch operation {
	case "INSERT":
		sb.WriteString("created")
	case "UPDATE":
		sb.WriteString("updated")
	case "DELETE":
		sb.WriteString("deleted")
	default:
		sb.WriteString(strings.ToLower(operation))
	}

	if operation == "UPDATE" && len(changedFields) > 0 {
		sb.WriteString(" (")
		sb.WriteString(strings.Join(changedFields, ", "))
		sb.WriteString(")")
	}

	return sb.String()
}

func hasWatchedColumnChanged(watched, changed []string) bool {
	set := make(map[string]struct{}, len(watched))
	for _, w := range watched {
		set[w] = struct{}{}
	}
	for _, c := range changed {
		if _, ok := set[c]; ok {
			return true
		}
	}
	return false
}
