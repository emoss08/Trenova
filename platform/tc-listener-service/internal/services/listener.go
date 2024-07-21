// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"sync"
	"time"

	"kafka/pkg"

	"github.com/jordan-wright/email"
	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kgo"
)

const globalAlertUpdateTopic = "trenova_app.public.table_change_alerts"

type KafkaListener struct {
	client              *kgo.Client
	running             bool
	mu                  sync.Mutex
	emailService        *EmailService
	subscriptionService *SubscriptionService
	logger              *zerolog.Logger
	batchEmailer        *BatchEmailer
	commitStyle         pkg.CommitStyle
}

func NewKafkaListener(ctx context.Context, brokers []string, group string, commitStyle pkg.CommitStyle, subService *SubscriptionService, emailService *EmailService, logger *zerolog.Logger) (*KafkaListener, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.WithLogger(kgo.BasicLogger(os.Stderr, kgo.LogLevelInfo, nil)),
	}

	if commitStyle != pkg.AutoCommit {
		opts = append(opts, kgo.DisableAutoCommit())
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		logger.Err(err).Msg("Failed to create Kafka client")
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	batchEmailer := NewBatchEmailer(emailService, logger, 30*time.Second)

	kl := &KafkaListener{
		client:              client,
		running:             true,
		emailService:        emailService,
		subscriptionService: subService,
		logger:              logger,
		batchEmailer:        batchEmailer,
		commitStyle:         commitStyle,
	}

	if err = kl.updateTopics(ctx); err != nil {
		return nil, err
	}

	go kl.consume(ctx)

	return kl, nil
}

func (kl *KafkaListener) updateTopics(ctx context.Context) error {
	subscriptions, err := kl.subscriptionService.GetActiveSubscriptions(ctx)
	if err != nil {
		kl.logger.Err(err).Msg("Failed to get active subscriptions")
		return fmt.Errorf("failed to get active subscriptions: %w", err)
	}

	topics := make([]string, 0, len(subscriptions))
	for _, sub := range subscriptions {
		topics = append(topics, sub.TopicName)
	}

	// Add the global alert update topic
	topics = append(topics, globalAlertUpdateTopic)

	kl.logger.Info().Msgf("Updating subscriptions to: %v", topics)
	kl.client.AddConsumeTopics(topics...)

	return nil
}

func (kl *KafkaListener) consume(ctx context.Context) {
	for kl.running {
		fetches := kl.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			kl.logger.Error().Err(err).Str("topic", t).Int32("partition", p).Msg("Fetch error")
		})

		var records []*kgo.Record
		fetches.EachRecord(func(record *kgo.Record) {
			records = append(records, record)
			kl.processRecord(ctx, record)
		})

		switch kl.commitStyle {
		case pkg.AutoCommit:
			kl.logger.Info().Msgf("Processed %d records - autocommit will handle offsets", len(records))
		case pkg.ManualCommitRecords:
			if err := kl.client.CommitRecords(ctx, records...); err != nil {
				kl.logger.Error().Err(err).Msg("Failed to commit records")
			} else {
				kl.logger.Info().Msgf("Committed %d records manually", len(records))
			}
		case pkg.ManualCommitUncommitted:
			if err := kl.client.CommitUncommittedOffsets(ctx); err != nil {
				kl.logger.Error().Err(err).Msg("Failed to commit uncommitted offsets")
			} else {
				kl.logger.Info().Msgf("Committed uncommitted offsets for %d records", len(records))
			}
		}
	}
}

func (kl *KafkaListener) processRecord(ctx context.Context, record *kgo.Record) {
	if record.Topic == globalAlertUpdateTopic {
		kl.handleAlertUpdate(ctx, record)
	} else {
		kl.handleDataMessage(ctx, record)
	}
}

func (kl *KafkaListener) handleDataMessage(ctx context.Context, record *kgo.Record) {
	if !json.Valid(record.Value) {
		kl.logger.Info().Msgf("Received invalid JSON message: %s", string(record.Value))
		return
	}

	payload, err := ParseDebeziumPayload(record.Value)
	if err != nil {
		kl.logger.Err(err).Msg("Failed to parse Debezium payload")
		return
	}

	kl.processDataMessage(ctx, record, *payload)
}

func (kl *KafkaListener) processDataMessage(ctx context.Context, record *kgo.Record, payload DebeziumPayload) {
	kl.logger.Info().Str("topicName", record.Topic).Msg("Processing message for topic")

	subscriptions, err := kl.subscriptionService.GetActiveSubscriptions(ctx)
	if err != nil {
		kl.logger.Err(err).Msg("Failed to get active subscriptions")
		return
	}

	organizationID, ok := payload.After["organization_id"].(string)
	if !ok || organizationID == "" {
		kl.logger.Error().Interface("payload", payload).Msg("Organization ID not found in Debezium payload")
		return
	}

	for _, sub := range subscriptions {
		if kl.subscriptionService.MatchesSubscription(sub, payload) &&
			sub.TopicName == record.Topic &&
			sub.OrganizationID == organizationID {
			kl.sendMessage(sub, payload)
		}
	}
}

func (kl *KafkaListener) sendMessage(sub Subscription, payload DebeziumPayload) {
	switch sub.DeliveryMethod {
	case Email:
		kl.logger.Info().Msgf("Sending email for subscription: %v", sub)
		kl.sendEmail(sub, payload)
	case Local, API, SMS:
		kl.logger.Info().Msgf("Delivery method not yet implemented: %s", sub.DeliveryMethod)
	default:
		kl.logger.Info().Msgf("Unsupported delivery method: %s", sub.DeliveryMethod)
	}
}

func (kl *KafkaListener) loadEmailTemplate() (*template.Template, error) {
	emailTemplate, err := os.ReadFile("web/templates/table_change_alert.html.tmpl")
	if err != nil {
		kl.logger.Err(err).Msg("Failed to read email template")
		return nil, fmt.Errorf("failed to read email template: %w", err)
	}

	tmpl, err := template.New("email").Parse(string(emailTemplate))
	if err != nil {
		kl.logger.Err(err).Msg("Failed to parse email template")
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}

	return tmpl, nil
}

func (kl *KafkaListener) sendEmail(sub Subscription, payload DebeziumPayload) {
	tmpl, err := kl.loadEmailTemplate()
	if err != nil {
		kl.logger.Err(err).Msg("Failed to load email template")
		return
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, payload); err != nil {
		kl.logger.Err(err).Msg("Failed to execute email template")
		return
	}

	subjectName := fmt.Sprintf("Table Change Alert: %s", sub.TopicName)

	if sub.CustomSubject != "" {
		subjectName = sub.CustomSubject
	}

	e := email.NewEmail()
	e.From = kl.emailService.From
	e.Subject = subjectName
	e.HTML = buf.Bytes()

	recipients := strings.Split(sub.EmailRecipients, ",")
	e.To = recipients
	kl.batchEmailer.AddEmail(e)
}

func (kl *KafkaListener) handleAlertUpdate(ctx context.Context, record *kgo.Record) {
	kl.logger.Info().Msgf("Alert update message on %s: %s", record.Topic, string(record.Value))

	kl.subscriptionService.InvalidateCache(ctx)

	if err := kl.updateTopics(ctx); err != nil {
		kl.logger.Err(err).Msg("Failed to update topics after alert update")
	}
}

func (kl *KafkaListener) shutdown() {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	if !kl.running {
		return
	}
	kl.logger.Info().Msg("Shutting down Kafka listener...")
	kl.batchEmailer.Stop()
	kl.client.Close()
	kl.running = false
	kl.logger.Info().Msg("Kafka listener shutdown successfully")
}

func StartListener(brokers []string, group string, commitStyle pkg.CommitStyle, subService *SubscriptionService, emailService *EmailService, logger *zerolog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := NewKafkaListener(ctx, brokers, group, commitStyle, subService, emailService, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Kafka listener")
	}

	// Wait for interrupt signal
	<-ctx.Done()
	listener.shutdown()
}
