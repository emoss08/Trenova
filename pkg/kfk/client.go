package kfk

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/twmb/franz-go/pkg/kmsg"

	"github.com/twmb/franz-go/pkg/kadm"
)

type KafkaClient struct {
	Seeds    []string
	Admin    *kadm.Client
	Logger   *zerolog.Logger
	Producer *kgo.Client
}

func NewKafkaClient(seeds []string, logger *zerolog.Logger) (*KafkaClient, error) {
	client := &KafkaClient{Seeds: seeds, Logger: logger}
	err := client.initialize()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// initialize sets up the Kafka client and admin.
func (k *KafkaClient) initialize() error {
	opts := []kgo.Opt{
		kgo.SeedBrokers(k.Seeds...),
		kgo.ProducerBatchMaxBytes(16384),
		kgo.ProducerLinger(5 * time.Millisecond),
		kgo.RetryBackoffFn(func(attempts int) time.Duration {
			return time.Second * time.Duration(attempts)
		}),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		k.Logger.Error().Err(err).Msg("failed to create Kafka client")
		return err
	}

	k.Producer = client
	k.Admin = kadm.NewClient(client)

	return nil
}

func (k *KafkaClient) Close() {
	if k.Producer != nil {
		k.Producer.Close()
	}
}

// GetTopics retrieves a list of all topics from the Kafka broker.
func (k *KafkaClient) GetTopics() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := kmsg.NewMetadataRequest()
	resp, err := req.RequestWith(ctx, k.Producer)
	if err != nil {
		k.Logger.Error().Err(err).Msg("failed to get topics from Kafka")
		return nil, err
	}

	topics := make([]string, 0, len(resp.Topics))
	for _, topic := range resp.Topics {
		if topic.Topic == nil {
			k.Logger.Warn().Msg("encountered a nil topic name")
			continue
		}

		topicName := *topic.Topic
		k.Logger.Debug().Msgf("topic: %s", topicName)

		err = kerr.ErrorForCode(topic.ErrorCode)
		if err != nil {
			k.Logger.Error().Err(err).Msgf("topic %s response has errored: %v", topicName, err)
			continue
		}

		topics = append(topics, topicName)
	}

	if len(topics) == 0 {
		k.Logger.Warn().Msg("no valid topics found")
	}

	return topics, nil
}
