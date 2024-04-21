// Package kafka provides a high-level Kafka client abstraction.
package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/pkg/errors"
)

// Client encapsulates a Kafka admin client.
type Client struct {
	Config *kafka.ConfigMap   // Config is the configuration map for the Kafka client.
	Client *kafka.AdminClient // Client is the actual Kafka admin client.
}

// NewClient creates and initializes a new Kafka admin client using the specified configuration.
// It returns the initialized client or panics if creation fails.
func NewClient(config *kafka.ConfigMap) *Client {
	kClient := &Client{Config: config}
	kClient.initialize()
	return kClient
}

// initialize sets up the Kafka admin client. This method panics if the client cannot be created.
func (k *Client) initialize() {
	client, err := kafka.NewAdminClient(k.Config)
	if err != nil {
		panic(errors.Wrap(err, "failed to create Kafka admin client"))
	}
	k.Client = client
}

// Close terminates the connection to the Kafka broker. It does nothing if the client is nil.
func (k *Client) Close() {
	if k.Client != nil {
		k.Client.Close()
	}
}

// GetTopics retrieves a list of all topics from the Kafka broker.
// It returns a slice of topic names or an error if the operation fails.
func (k *Client) GetTopics() ([]string, error) {
	meta, err := k.Client.GetMetadata(nil, true, 5000) // 5000 is the timeout in milliseconds
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metadata from Kafka")
	}

	var topics []string
	for topic := range meta.Topics {
		topics = append(topics, topic)
	}
	return topics, nil
}
