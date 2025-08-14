<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Email Microservice

A microservice for handling email sending operations across the Trenova application ecosystem.

## Features

- RabbitMQ integration for receiving email requests from other services
- Template-based email composition
- Support for multiple email providers (SMTP, SendGrid, etc.)
- Uses [go-mail](https://github.com/wneessen/go-mail) for robust email delivery
- Tenant-specific email configurations
- Robust error handling and retry mechanisms
- Email delivery status tracking
- Rate limiting to avoid sending too many emails at once
- Supports HTML and plain text email formats
- Attachment support
- Configurable TLS policies (mandatory, opportunistic, none)
- Includes MailHog for local email testing
- Template management interface for development

## Architecture

The email microservice follows a simple but effective architecture:

1. **Consumers**: Listen for email requests on RabbitMQ queues
2. **Handlers**: Process email requests and validate them
3. **Templates**: Manage and render email templates
4. **Providers**: Abstract different email delivery mechanisms
5. **Sender**: Coordinate the email sending process

## Usage

Other services can send email requests via RabbitMQ with the appropriate routing key:

- `email.send` - For sending emails

## Message Structure

```json
{
  "id": "unique-message-id",
  "type": "email.send",
  "entityId": "related-entity-id",
  "entityType": "user|shipment|invoice|etc",
  "tenantId": "tenant-id",
  "requestedAt": "2023-07-21T12:00:00Z",
  "payload": {
    "template": "welcome|password-reset|invoice|etc",
    "to": ["recipient@example.com"],
    "cc": ["cc@example.com"],
    "bcc": ["bcc@example.com"],
    "subject": "Email Subject",
    "data": {
      "key1": "value1",
      "key2": "value2"
    },
    "attachments": [
      {
        "filename": "invoice.pdf",
        "content": "base64-encoded-content",
        "contentType": "application/pdf"
      }
    ]
  }
}
```

## Configuration

Configuration is loaded from environment variables and/or a .env file:

```env
# Server
EMAIL_ENV=development
EMAIL_PORT=8082

# Database
EMAIL_DB_HOST=localhost
EMAIL_DB_PORT=5432
EMAIL_DB_USER=postgres
EMAIL_DB_PASSWORD=postgres
EMAIL_DB_NAME=email_service
EMAIL_DB_SCHEMA=public
EMAIL_DB_SSL_MODE=disable
EMAIL_DB_MAX_CONNECTIONS=10
EMAIL_DB_MAX_IDLE_CONNS=5

# RabbitMQ
EMAIL_RABBITMQ_HOST=localhost
EMAIL_RABBITMQ_PORT=5672
EMAIL_RABBITMQ_USER=guest
EMAIL_RABBITMQ_PASSWORD=guest
EMAIL_RABBITMQ_VHOST=/
EMAIL_RABBITMQ_EXCHANGE=trenova.events
EMAIL_RABBITMQ_QUEUE=email.service
EMAIL_RABBITMQ_PREFETCH_COUNT=10
EMAIL_RABBITMQ_TIMEOUT=5s

# SMTP (Default Provider)
EMAIL_SMTP_HOST=smtp.example.com
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USER=user
EMAIL_SMTP_PASSWORD=password
EMAIL_SMTP_FROM=no-reply@example.com
EMAIL_SMTP_FROM_NAME=Trenova
EMAIL_SMTP_TLS_POLICY=mandatory
EMAIL_SMTP_TIMEOUT=30s

# SendGrid (Optional Provider)
EMAIL_SENDGRID_API_KEY=your-api-key
```

## Development

### Prerequisites

- Go 1.24.2 or higher
- PostgreSQL
- RabbitMQ
- Docker and Docker Compose (for local development with MailHog)

### Setup

1. Clone the repository
2. Create a `.env` file with the required configuration
3. Run `go mod download` to download dependencies
4. Run `go run main.go` to start the service

### Template Management (Development Only)

When running in development mode, the email service provides a web-based template management interface for editing and previewing email templates without sending actual emails.

To access this interface:

1. Ensure the service is running in development mode (`EMAIL_ENV=development` or unset)
2. Open a web browser and navigate to http://localhost:8083
3. From there, you can:
   - View a list of all available templates
   - Edit template HTML content
   - Preview templates with sample data
   - Save changes to template files

The interface uses a single consolidated template layout that provides:
- A sidebar template navigation menu
- Interactive editor with theme support
- Live preview capabilities
- Sample data management
- Real-time template editing

This simplifies the template development process by allowing immediate visual feedback without sending test emails.

### Local Testing with MailHog

MailHog is included in the Docker Compose configuration for easy email testing:

1. Start the services with `docker-compose up -d`
2. The email service will be configured to use MailHog as the SMTP server
3. Access the MailHog web interface at http://localhost:8025
4. All emails sent during local development will be captured by MailHog
5. No real emails will be sent, allowing safe testing

When the service is running with Docker Compose, the following MailHog settings are automatically applied:

```
EMAIL_SMTP_HOST=mailhog
EMAIL_SMTP_PORT=1025
EMAIL_SMTP_USER=
EMAIL_SMTP_PASSWORD=
EMAIL_SMTP_TLS_POLICY=none
```

This configuration allows emails to be sent without authentication and TLS, which is how MailHog operates by default. 