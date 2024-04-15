package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Client struct {
	Config *kafka.ConfigMap
	Client *kafka.AdminClient
}

func NewKafkaClient(config *kafka.ConfigMap) *Client {
	client := &Client{
		Config: config,
	}
	client.init()

	return client
}

func (k *Client) init() {
	client, err := kafka.NewAdminClient(k.Config)
	if err != nil {
		panic(err)
	}

	k.Client = client
}

func (k *Client) Close() {
	if k.Client != nil {
		k.Client.Close()
	}
}

func (k *Client) GetKafkaTopics() ([]string, error) {
	meta, err := k.Client.GetMetadata(nil, true, 5000)
	if err != nil {
		return nil, err
	}

	var topics []string
	for topic := range meta.Topics {
		topics = append(topics, topic)
	}

	return topics, nil
}
