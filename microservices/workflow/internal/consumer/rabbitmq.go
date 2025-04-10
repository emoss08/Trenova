package consumer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/workflow/internal/config"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rotisserie/eris"
)

type MessageHandler func(ctx context.Context, msg *model.Message) error

type RabbitMQConsumer struct {
	config         *config.RabbitMQConfig
	conn           *amqp.Connection
	ch             *amqp.Channel
	handlers       map[model.WorkflowType]MessageHandler
	reconnectDelay time.Duration
	mu             sync.RWMutex
	done           chan struct{}
}

func NewRabbitMQConsumer(cfg *config.RabbitMQConfig) (*RabbitMQConsumer, error) {
	consumer := &RabbitMQConsumer{
		config:         cfg,
		handlers:       make(map[model.WorkflowType]MessageHandler),
		reconnectDelay: 5 * time.Second,
		done:           make(chan struct{}),
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) RegisterHandler(workflowType model.WorkflowType, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[workflowType] = handler
	log.Printf("Registered handler for workflow type: %s", workflowType)
}

// Start begins consuming messages from RabbitMQ
func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	var err error

	// Connect to RabbitMQ
	if err = c.connect(); err != nil {
		return eris.Wrap(err, "failed to connect to RabbitMQ")
	}

	// Set up the necessary exchanges, queues, and bindings
	if err = c.setupTopology(); err != nil {
		return eris.Wrap(err, "failed to set up RabbitMQ topology")
	}

	// Register workflow type handlers
	c.registerWorkflowBindings()

	// Start consuming messages
	go c.consumeMessages(ctx)

	return nil
}

func (c *RabbitMQConsumer) connect() error {
	var err error

	// Try to connect to RabbitMQ with retries
	for range make([]struct{}, 5) {
		c.conn, err = amqp.Dial(c.config.URL())
		if err == nil {
			log.Printf("Connected to RabbitMQ")
			break
		}

		log.Printf("Failed to connect to RabbitMQ: %v", err)
		time.Sleep(c.reconnectDelay)
	}

	if err != nil {
		return eris.Wrap(err, "failed to connect to RabbitMQ")
	}

	// Create a channel AFTER successful connection
	c.ch, err = c.conn.Channel()
	if err != nil {
		return eris.Wrap(err, "failed to open channel")
	}

	// Set QoS prefetch AFTER creating the channel
	err = c.ch.Qos(
		c.config.PrefetchCount,
		0,
		false,
	)
	if err != nil {
		return eris.Wrap(err, "failed to set QoS")
	}

	log.Printf("Connected to RabbitMQ and set QoS at %v", c.config.Host)
	return nil
}

func (c *RabbitMQConsumer) setupTopology() error {
	// * Declare exchange with type "direct" to match existing exchange
	err := c.ch.ExchangeDeclare(
		c.config.ExchangeName,
		"direct", // Changed from "topic" to match existing exchange
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare exchange")
	}

	// Declare queue if not already done
	_, err = c.ch.QueueDeclare(
		c.config.QueueName, // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare queue")
	}

	log.Printf("Setup RabbitMQ topology with exchange %v and queue %v", c.config.ExchangeName, c.config.QueueName)
	return nil
}

func (c *RabbitMQConsumer) registerWorkflowBindings() {
	workflowTypes := []model.WorkflowType{
		model.WorkflowTypeShipmentUpdated,
	}

	for _, wType := range workflowTypes {
		// * Use the workflow type as the routing keys
		err := c.ch.QueueBind(
			c.config.QueueName,
			string(wType),
			c.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Warning: Failed to bind queue to exchange for workflow type %v: %v", wType, err)
			continue
		}
		log.Printf("Bound queue %v to exchange %v for workflow type %v", c.config.QueueName, c.config.ExchangeName, wType)
	}
}

// consumeMessages continuously consumes messages from the queue
func (c *RabbitMQConsumer) consumeMessages(ctx context.Context) {
	msgs, err := c.ch.Consume(
		c.config.QueueName, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		log.Printf("Failed to register consumer: %v", err)
		return
	}

	log.Printf("Started consuming messages from queue: %s", c.config.QueueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping consumer")
			return
		case <-c.done:
			log.Println("Consumer closed, stopping")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Channel closed, attempting to reconnect...")
				time.Sleep(c.reconnectDelay)

				if err = c.connect(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
					continue
				}

				if err = c.setupTopology(); err != nil {
					log.Printf("Failed to set up topology after reconnect: %v", err)
					continue
				}

				c.registerWorkflowBindings()

				msgs, err = c.ch.Consume(
					c.config.QueueName,
					"",
					false,
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					log.Printf("Failed to register consumer after reconnect: %v", err)
				}
				continue
			}

			// Process the message
			go c.processMessage(ctx, msg)
		}
	}
}

// processMessage handles an individual message
func (c *RabbitMQConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in message processing: %v", r)
			// Nack the message with requeue=true to try again later
			if err := msg.Nack(false, true); err != nil {
				log.Printf("Failed to nack message: %v", err)
			}
		}
	}()

	log.Printf("Received message with routing key: %s", msg.RoutingKey)

	// Parse the message
	message, err := deserializeMessage(msg.Body)
	if err != nil {
		log.Printf("Failed to deserialize message: %v", err)
		// Nack the message without requeue as it's malformed
		if err = msg.Nack(false, false); err != nil {
			log.Printf("Failed to nack message: %v", err)
		}
		return
	}

	// Find the appropriate handler
	c.mu.RLock()
	handler, exists := c.handlers[message.Type]
	c.mu.RUnlock()

	if !exists {
		log.Printf("No handler registered for workflow type: %s", message.Type)
		// Ack the message since we can't process it
		if err = msg.Ack(false); err != nil {
			log.Printf("Failed to ack message: %v", err)
		}
		return
	}

	// Execute the handler
	err = handler(ctx, message)
	if err != nil {
		log.Printf("Error handling message: %v", err)
		// Nack the message with requeue=true to try again later
		if err = msg.Nack(false, true); err != nil {
			log.Printf("Failed to nack message: %v", err)
		}
		return
	}

	// Acknowledge the message
	if err = msg.Ack(false); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}
}

// Stop gracefully shuts down the consumer
func (c *RabbitMQConsumer) Stop() error {
	close(c.done)

	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
			return eris.Wrap(err, "failed to close channel")
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return eris.Wrap(err, "failed to close connection")
		}
	}

	log.Println("RabbitMQ consumer stopped")
	return nil
}

// Helper functions
func deserializeMessage(data []byte) (*model.Message, error) {
	var message model.Message

	if err := sonic.Unmarshal(data, &message); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal message")
	}

	return &message, nil
}
