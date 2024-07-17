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

package kfk

import (
	"github.com/pkg/errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog"
)

// Client encapsulates a Kafka admin and producer client.
type Client struct {
	Config   *kafka.ConfigMap   // Config is the configuration map for the Kafka client.
	Admin    *kafka.AdminClient // Admin is the actual Kafka admin client.
	Producer *kafka.Producer    // Producer is the Kafka producer for sending messages.
	Logger   *zerolog.Logger    // Logger is the logger for the Kafka client.
}

// ConfigMap is a type alias for the Kafka configuration map.
type ConfigMap = kafka.ConfigMap

// NewClient creates and initializes a new Kafka admin client using the specified configuration.
// It returns the initialized client or panics if creation fails.
func NewClient(config *kafka.ConfigMap, logger *zerolog.Logger) *Client {
	// Adding default settings for producer performance
	if err := config.SetKey("linger.ms", "5"); err != nil { // Delay in ms to allow messages to batch
		logger.Warn().Err(err).Msg("failed to set producer configuration")
	}
	if err := config.SetKey("batch.size", "16384"); err != nil { // Batch size in bytes
		logger.Warn().Err(err).Msg("failed to set producer configuration")
	}

	client := &Client{Config: config, Logger: logger}
	client.initialize()
	return client
}

// initialize sets up the Kafka admin and producer clients. This method panics if the clients cannot be created.
func (k *Client) initialize() {
	admin, err := kafka.NewAdminClient(k.Config)
	if err != nil {
		k.Logger.Fatal().Err(err).Msg("failed to create Kafka admin client")
	}

	k.Admin = admin

	producer, err := kafka.NewProducer(k.Config)
	if err != nil {
		k.Logger.Fatal().Err(err).Msg("failed to create Kafka producer")
	}
	k.Producer = producer
}

// Close terminates the connections to the Kafka broker. It closes both the admin and producer clients.
func (k *Client) Close() {
	if k.Admin != nil {
		k.Admin.Close()
	}
	if k.Producer != nil {
		k.Producer.Close()
	}
}

// GetTopics retrieves a list of all topics from the Kafka broker.
// It returns a slice of topic names or an error if the operation fails.
func (k *Client) GetTopics() ([]string, error) {
	meta, err := k.Admin.GetMetadata(nil, true, 5000) // 5000 is the timeout in milliseconds
	if err != nil {
		k.Logger.Error().Err(err).Msg("failed to get metadata from Kafka")
		return nil, errors.Wrap(err, "failed to get metadata from Kafka")
	}

	var topics []string
	for topic := range meta.Topics {
		topics = append(topics, topic)
	}
	return topics, nil
}

// SendMessage sends a message to the specified topic.
// It returns an error if the message cannot be delivered.
func (k *Client) SendMessage(topic, message string) error {
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}

	// Produce message asynchronously
	if err := k.Producer.Produce(msg, nil); err != nil {
		return errors.Wrap(err, "failed to produce message") // nil passed for the delivery channel for asynchronous processing
	}

	go func() {
		for e := range k.Producer.Events() {
			if ev, ok := e.(*kafka.Message); ok {
				if ev.TopicPartition.Error != nil {
					k.Logger.Error().Err(ev.TopicPartition.Error).Msg("failed to deliver message")
				} else {
					k.Logger.Info().Msgf("delivered message to %v", ev.TopicPartition)
				}
			}
		}
	}()

	return nil
}
