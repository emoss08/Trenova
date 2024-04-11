package kafka

import (
	"context"
	"errors"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var (
	client *kafka.AdminClient
	once   sync.Once
)

// Initialize Kafka client in a thread-safe manner.
func initClient(config *kafka.ConfigMap) error {
	var err error
	once.Do(func() {
		client, err = kafka.NewAdminClient(config)
	})
	return err
}

func SetKafkaClient(newClient *kafka.AdminClient) {
	client = newClient
}

func GetKafkaClient() (*kafka.AdminClient, error) {
	if client == nil {
		return nil, errors.New("kafka client is not initialized")
	}
	return client, nil
}

func NewKafkaClient(config *kafka.ConfigMap) (*kafka.AdminClient, error) {
	if err := initClient(config); err != nil {
		return nil, err
	}
	return client, nil
}

func CloseKafkaClient() {
	if client != nil {
		client.Close()
	}
}

func GetKafkaTopics() ([]string, error) {
	if client == nil {
		return nil, errors.New("kafka client is not initialized")
	}
	meta, err := client.GetMetadata(nil, true, 5000)
	if err != nil {
		return nil, err
	}

	var topics []string
	for topic := range meta.Topics {
		topics = append(topics, topic)
	}

	return topics, nil
}

func CreateTopic(broker string, topic string, partitions int, replicationFactor int) error {
	config := &kafka.ConfigMap{
		"bootstrap.servers": broker,
	}

	newClient, err := NewKafkaClient(config)
	if err != nil {
		return err
	}
	defer newClient.Close()

	topicSpec := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	}

	_, err = client.CreateTopics(context.TODO(), []kafka.TopicSpecification{topicSpec})
	return err
}
