package email

import (
	"context"
	"log"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/workflow/internal/config"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rotisserie/eris"
)

// Client is a simple client for sending emails via RabbitMQ
type Client struct {
	config     *config.RabbitMQConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

// EmailMessage represents the email message structure
type EmailMessage struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	EntityID    string    `json:"entityId"`
	EntityType  string    `json:"entityType"`
	TenantID    string    `json:"tenantId"`
	RequestedAt time.Time `json:"requestedAt"`
	Payload     Payload   `json:"payload"`
}

// Payload represents the email payload
type Payload struct {
	Template    string            `json:"template"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Data        map[string]any    `json:"data,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // Base64 encoded content
	ContentType string `json:"contentType"`
}

// NewClient creates a new email client
func NewClient(cfg *config.RabbitMQConfig) (*Client, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.URL())
	if err != nil {
		return nil, eris.Wrap(err, "failed to connect to RabbitMQ")
	}

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, eris.Wrap(err, "failed to open a channel")
	}

	// Declare the exchange
	err = ch.ExchangeDeclare(
		cfg.ExchangeName, // name
		"direct",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, eris.Wrap(err, "failed to declare an exchange")
	}

	return &Client{
		config:     cfg,
		connection: conn,
		channel:    ch,
	}, nil
}

// SendEmail sends an email message to the email service
func (c *Client) SendEmail(ctx context.Context, tenantID, entityID, entityType, template, subject string, to []string, data map[string]any) error {
	// Validate that we have at least one valid email address
	if len(to) == 0 {
		return eris.New("no recipient email addresses provided")
	}

	// Filter out empty email addresses
	validEmails := make([]string, 0, len(to))
	for _, email := range to {
		if email != "" {
			validEmails = append(validEmails, email)
		}
	}

	// Check if we have any valid email addresses after filtering
	if len(validEmails) == 0 {
		return eris.New("no valid recipient email addresses provided")
	}

	// Create an email message
	message := EmailMessage{
		ID:          uuid.New().String(),
		Type:        "email.send",
		EntityID:    entityID,
		EntityType:  entityType,
		TenantID:    tenantID,
		RequestedAt: time.Now(),
		Payload: Payload{
			Template: template,
			To:       validEmails,
			Subject:  subject,
			Data:     data,
		},
	}

	// Convert message to JSON
	body, err := sonic.Marshal(message)
	if err != nil {
		return eris.Wrap(err, "failed to marshal message")
	}

	// Log message details for debugging
	log.Printf("Sending email message - ID: %s, To: %v, Subject: %s, Template: %s",
		message.ID, validEmails, subject, template)
	log.Printf("Email message JSON: %s", string(body))

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Publish the message to RabbitMQ
	err = c.channel.PublishWithContext(
		timeoutCtx,
		c.config.ExchangeName, // exchange
		"email.send",          // routing key
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return eris.Wrap(err, "failed to publish message")
	}

	log.Printf("Successfully published email message to RabbitMQ - ID: %s, Exchange: %s, RoutingKey: %s",
		message.ID, c.config.ExchangeName, "email.send")

	return nil
}

// Close closes the RabbitMQ connection
func (c *Client) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return eris.Wrap(err, "failed to close channel")
		}
	}

	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			return eris.Wrap(err, "failed to close connection")
		}
	}

	return nil
}
