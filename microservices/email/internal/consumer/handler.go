package consumer

import (
	"context"
	"log"

	"github.com/emoss08/trenova/microservices/email/internal/email"
	"github.com/emoss08/trenova/microservices/email/internal/model"
)

// EmailHandler handles email messages
type EmailHandler struct {
	sender *email.SenderService
}

// NewEmailHandler creates a new email handler
func (c *RabbitMQConsumer) NewEmailHandler(sender *email.SenderService) *EmailHandler {
	return &EmailHandler{
		sender: sender,
	}
}

// HandleEmailSendMessage handles email.send messages
func (h *EmailHandler) HandleEmailSendMessage(ctx context.Context, msg *model.Message) error {
	log.Printf("Processing email message: %s, ID: %s", msg.Type, msg.ID)

	// Send the email
	if err := h.sender.SendEmailFromMessage(ctx, msg); err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	log.Printf("Successfully sent email for message ID: %s", msg.ID)
	return nil
}
