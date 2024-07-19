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
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jordan-wright/email"
	"github.com/rs/zerolog"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type KafkaListener struct {
	dataConsumer        *kafka.Consumer
	alertUpdateConsumer *kafka.Consumer
	running             bool
	mu                  sync.Mutex
	emailService        *EmailService
	subscriptionService *SubscriptionService
	logger              *zerolog.Logger
	batchEmailer        *BatchEmailer
}

// NewKafkaListener initializes a KafkaListener with a data consumer, an alert update consumer,
// and an email service. It subscribes the alert update consumer to the specified topic and
// starts listening for messages.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the KafkaListener
//	alertTopic - Kafka topic for alert updates
//	subService - SubscriptionService for managing subscriptions
//
// Returns:
//
//	KafkaListener - a new KafkaListener instance
//	error - an error if initialization fails
func NewKafkaListener(ctx context.Context, alertTopic string, subService *SubscriptionService, logger *zerolog.Logger) (*KafkaListener, error) {
	dataConsumer, err := createConsumer("localhost:9092", "trenova-data-alert-1")
	if err != nil {
		logger.Err(err).Msg("Failed to create data consumer")
		return nil, fmt.Errorf("failed to create data consumer: %w", err)
	}

	alertUpdateConsumer, err := createConsumer("localhost:9092", "trenova-table-change-alert-1")
	if err != nil {
		logger.Err(err).Msg("Failed to create alert update consumer")
		return nil, fmt.Errorf("failed to create alert update consumer: %w", err)
	}

	emailService := NewEmailService()

	// Ping the email service to ensure it is available
	if err := emailService.Ping(); err != nil {
		logger.Err(err).Msg("Failed to ping email service")
		return nil, fmt.Errorf("failed to ping email service: %w", err)
	}

	batchEmailer := NewBatchEmailer(emailService, logger, 30*time.Second) // Send emails every 30 seconds

	kl := &KafkaListener{
		dataConsumer:        dataConsumer,
		alertUpdateConsumer: alertUpdateConsumer,
		running:             true,
		emailService:        emailService,
		subscriptionService: subService,
		logger:              logger,
		batchEmailer:        batchEmailer,
	}

	if err := kl.alertUpdateConsumer.Subscribe(alertTopic, nil); err != nil {
		logger.Err(err).Msgf("Failed to subscribe alert update consumer to topic %s", alertTopic)
		return nil, fmt.Errorf("failed to subscribe alert update consumer to topic %s: %w", alertTopic, err)
	}

	go kl.listen(ctx)

	return kl, nil
}

// createConsumer creates a new Kafka consumer configured with the specified bootstrap servers
// and group ID. The consumer starts reading from the latest offset and auto-commit is disabled.
//
// Parameters:
//
//	bootstrapServers - comma-separated list of Kafka bootstrap servers
//	groupID - consumer group ID
//
// Returns:
//
//	*kafka.Consumer - a new Kafka consumer instance
//	error - an error if the consumer creation fails
func createConsumer(bootstrapServers, groupID string) (*kafka.Consumer, error) {
	return kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"group.id":           groupID,
		"auto.offset.reset":  "latest", // Start from the latest offset
		"enable.auto.commit": false,    // Disable auto commit to control offset commits manually
		// "debug":              "cgrp,topic,fetch", // Enable debug logging

	})
}

// subscribeToTopics retrieves the active subscriptions and subscribes the data consumer to
// the relevant Kafka topics. It periodically checks for updates to the subscriptions and
// retries if no active topics are found.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the subscription process
func (kl *KafkaListener) subscribeToTopics(ctx context.Context) {
	for kl.running {
		select {
		case <-ctx.Done():
			return
		default:
			subscriptions, err := kl.subscriptionService.GetActiveSubscriptions(ctx)
			if err != nil {
				kl.logger.Err(err).Msg("Failed to get active subscriptions")
				time.Sleep(5 * time.Minute)
				continue
			}

			topicsMap := make(map[string]struct{})
			for _, sub := range subscriptions {
				topicsMap[sub.TopicName] = struct{}{}
			}

			var topics []string
			for topic := range topicsMap {
				topics = append(topics, topic)
			}

			if len(topics) > 0 {
				kl.logger.Info().Msg("Retrieved active topics")
				for i, topic := range topics {
					kl.logger.Info().Msgf("Topic %d: '%s'", i, topic)
				}

				if err := kl.dataConsumer.SubscribeTopics(topics, nil); err != nil {
					kl.logger.Err(err).Msg("Failed to subscribe data consumer to topics")
				} else {
					kl.logger.Info().Msgf("Successfully subscribed to topics: %v", topics)
					return
				}
			} else {
				kl.logger.Info().Msg("No active topics found. Retrying in 5 minutes...")
			}
			time.Sleep(5 * time.Minute)
		}
	}
}

// listen starts goroutines to listen for data messages, alert updates, and topic subscriptions.
// It runs until the provided context is cancelled.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the listener
func (kl *KafkaListener) listen(ctx context.Context) {
	go kl.listenForDataMessages(ctx)
	go kl.listenForAlertUpdates(ctx)
	go kl.subscribeToTopics(ctx)

	<-ctx.Done()
	kl.shutdown()
}

// listenForDataMessages reads messages from the data consumer and processes valid JSON messages.
// It unmarshals the messages into DebeziumPayload structs and calls processDataMessage.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the message listening
func (kl *KafkaListener) listenForDataMessages(ctx context.Context) {
	for kl.running {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := kl.dataConsumer.ReadMessage(10 * time.Millisecond)
			if err == nil {
				if json.Valid(msg.Value) {
					var payload DebeziumPayload
					if err := json.Unmarshal(msg.Value, &payload); err != nil {
						kl.logger.Err(err).Msg("Failed to unmarshal JSON message")
						continue
					}
					kl.processDataMessage(ctx, msg, payload)
				} else {
					kl.logger.Info().Msgf("Received invalid JSON message: %s", string(msg.Value))
				}
			} else if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() != kafka.ErrTimedOut {
				kl.logger.Err(err).Msg("Error reading data message")
			}
		}
	}
}

// processDataMessage processes a Debezium payload message. It matches the message with active
// subscriptions and sends it if relevant to the subscription's topic and organization.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the message processing
//	msg - Kafka message containing the Debezium payload
//	payload - unmarshaled Debezium payload
func (kl *KafkaListener) processDataMessage(ctx context.Context, msg *kafka.Message, payload DebeziumPayload) {
	topicName := ""
	if msg.TopicPartition.Topic != nil {
		topicName = *msg.TopicPartition.Topic
	}

	subscriptions, err := kl.subscriptionService.GetActiveSubscriptions(ctx)
	if err != nil {
		kl.logger.Err(err).Msg("Failed to get active subscriptions")
		return
	}

	organizationID, ok := payload.After["organization_id"].(string)
	if !ok || organizationID == "" {
		kl.logger.Err(fmt.Errorf("organization ID not found in Debezium payload: %v", payload.After)).Msg("")
		return
	}

	for _, sub := range subscriptions {
		if kl.subscriptionService.MatchesSubscription(sub, payload) && sub.TopicName == topicName && sub.OrganizationID == organizationID {
			kl.sendMessage(sub, payload)
		}
	}
}

// loadEmailTemplate reads and parses the email template from a file.
// It returns the parsed template or an error if the template cannot be read or parsed.
//
// Returns:
//
//	*template.Template - parsed email template
//	error - an error if the template cannot be read or parsed
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

// sendMessage sends an email based on the provided subscription and Debezium payload.
// It uses the email template to format the message and sends it to the subscription's recipients.
//
// Parameters:
//
//	sub - subscription details for the message
//	payload - Debezium payload containing the data to send
func (kl *KafkaListener) sendMessage(sub Subscription, payload DebeziumPayload) {
	tmpl, err := kl.loadEmailTemplate()
	if err != nil {
		log.Printf("Failed to parse email template: %v", err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
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

	// TODO(WOLFRED): Add support for other delivery methods.
	switch sub.DeliveryMethod {
	case "Email":
		recipients := strings.Split(sub.EmailRecipients, ",")
		e.To = recipients
		kl.batchEmailer.AddEmail(e)
	case "Local":
		// We can probably transmit this via API to the local server and store it in the user inbox?
		// We could have it pass the notice via a kafka topic to the main application and then that manage the notification refresh?
		// Or we can use a local storage mechanism to store the notification and then have the main application poll for new notifications?
	case "Api":
		// We can batch API request similar to emails to whatever endpoint the user has configured.
	case "Sms":
		// This one is easy, we just need to find a way to send SMS messages.
	default:
		kl.logger.Info().Msgf("Unsupported delivery method: %s", sub.DeliveryMethod)
	}
}

// listenForAlertUpdates reads messages from the alert update consumer and processes them.
// It invalidates the cache and updates the subscriptions when an alert update is received.
//
// Parameters:
//
//	ctx - context for managing lifecycle of the alert update listening
func (kl *KafkaListener) listenForAlertUpdates(ctx context.Context) {
	for kl.running {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := kl.alertUpdateConsumer.ReadMessage(100 * time.Millisecond)
			if err == nil {
				kl.logger.Info().Msgf("Alert update message on %s: %s", msg.TopicPartition, msg.String())

				// Invalidate cache and re-fetch subscriptions
				kl.subscriptionService.InvalidateCache(ctx)

				subscriptions, err := kl.subscriptionService.GetActiveSubscriptions(ctx)
				if err != nil {
					kl.logger.Err(err).Msg("Failed to get active subscriptions")
					continue
				}

				topicsMap := make(map[string]struct{})
				for _, sub := range subscriptions {
					topicsMap[sub.TopicName] = struct{}{}
				}

				var topics []string
				for topic := range topicsMap {
					topics = append(topics, topic)
				}

				if len(topics) > 0 {
					kl.logger.Info().Msgf("Updating subscriptions to: %v", topics)
					if err := kl.dataConsumer.SubscribeTopics(topics, nil); err != nil {
						kl.logger.Info().Msgf("Failed to update topic subscriptions: %v", err)
					}
				}
			} else if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() != kafka.ErrTimedOut {
				kl.logger.Err(err).Msg("Error reading alert update message")
			}
		}
	}
}

// shutdown gracefully shuts down the KafkaListener by closing the Kafka consumers
// and stopping the listener. It ensures that the listener is no longer running and logs the shutdown status.
func (kl *KafkaListener) shutdown() {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	if !kl.running {
		return
	}
	log.Println("Shutting down Kafka listener...")
	kl.batchEmailer.Stop() // Ensure all batched emails are sent
	if err := kl.dataConsumer.Close(); err != nil {
		kl.logger.Printf("Error closing data consumer: %v\n", err)
	}
	if err := kl.alertUpdateConsumer.Close(); err != nil {
		kl.logger.Error().Err(err).Msg("Error closing alert update consumer")
	}
	kl.running = false
	kl.logger.Info().Msg("Kafka listener shutdown successfully")
}

// StartListener initializes and starts the KafkaListener with the provided subscription service.
// It runs until an interrupt signal is received, at which point it shuts down the listener.
//
// Parameters:
//
//	subService - SubscriptionService for managing subscriptions
func StartListener(subService *SubscriptionService, logger *zerolog.Logger) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	alertUpdateTopic := "trenova_app.public.table_change_alerts"
	listener, err := NewKafkaListener(ctx, alertUpdateTopic, subService, logger)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka listener: %v", err)
	}

	<-ctx.Done()
	listener.shutdown()
}
