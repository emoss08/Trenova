package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

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

func main() {
	// Define command-line flags
	toEmail := flag.String("to", "", "Recipient email address")
	template := flag.String("template", "welcome", "Email template to use (welcome, password-reset, etc.)")
	subject := flag.String("subject", "Test Email from Trenova", "Email subject")
	rabbitmqHost := flag.String("host", "localhost", "RabbitMQ host")
	rabbitmqPort := flag.Int("port", 5673, "RabbitMQ port")
	rabbitmqUser := flag.String("user", "guest", "RabbitMQ username")
	rabbitmqPass := flag.String("pass", "guest", "RabbitMQ password")
	rabbitmqExchange := flag.String("exchange", "trenova.events", "RabbitMQ exchange")
	flag.Parse()

	// Validate required parameters
	if *toEmail == "" {
		log.Fatal("Recipient email (-to) is required")
	}

	// Create RabbitMQ connection
	connURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", *rabbitmqUser, *rabbitmqPass, *rabbitmqHost, *rabbitmqPort)
	conn, err := amqp.Dial(connURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Create channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare exchange
	err = ch.ExchangeDeclare(
		*rabbitmqExchange, // name
		"direct",          // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare an exchange: %v", err)
	}

	// Prepare test data
	testUser := "Test User"
	testEmail := *toEmail
	loginURL := "https://app.trenova.io/login"
	currentYear := time.Now().Year()

	// Create email message
	message := EmailMessage{
		ID:          uuid.New().String(),
		Type:        "email.send",
		EntityID:    uuid.New().String(),
		EntityType:  "test",
		TenantID:    "test-tenant",
		RequestedAt: time.Now(),
		Payload: Payload{
			Template: *template,
			To:       []string{testEmail},
			Subject:  *subject,
			Data: map[string]any{
				"Name":     testUser,
				"Email":    testEmail,
				"Username": testUser,
				"LoginURL": loginURL,
				"Year":     currentYear,
			},
		},
	}

	// Convert message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	// Publish message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		*rabbitmqExchange, // exchange
		"email.send",      // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}

	fmt.Println("Test email sent successfully!")
	fmt.Println("Message ID:", message.ID)
	fmt.Println("Template:", message.Payload.Template)
	fmt.Println("To:", message.Payload.To)
	fmt.Println("Subject:", message.Payload.Subject)

	// If MailHog is being used, remind the user
	fmt.Println("\nIf you're using MailHog for testing, check http://localhost:8025 to view the email.")

	os.Exit(0)
}
