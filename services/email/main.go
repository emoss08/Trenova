/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/emoss08/trenova/microservices/email/internal/config"
	"github.com/emoss08/trenova/microservices/email/internal/consumer"
	"github.com/emoss08/trenova/microservices/email/internal/email"
	"github.com/emoss08/trenova/microservices/email/internal/model"
	"github.com/emoss08/trenova/microservices/email/internal/provider"
	"github.com/emoss08/trenova/microservices/email/internal/server"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

// SMTPProviderConstructor creates a new SMTP provider with the template service
func SMTPProviderConstructor(
	cfg *config.AppConfig,
	templateService *email.TemplateService,
) *provider.SMTPProvider {
	return provider.NewSMTPProvider(cfg, templateService)
}

// SendGridProviderConstructor creates a new SendGrid provider with the template service
func SendGridProviderConstructor(
	cfg *config.AppConfig,
	templateService *email.TemplateService,
) *provider.SendGridProvider {
	return provider.NewSendGridProvider(cfg, templateService)
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set up providers and configuration
	providers := []any{
		config.LoadConfig,
		email.NewTemplateService,
		SMTPProviderConstructor,
		SendGridProviderConstructor,
		provider.NewFactory,
		email.NewSenderService,
		consumer.NewRabbitMQConsumer,
	}

	// Create invokers for lifecycle hooks
	invokers := []any{
		setupConsumer,
	}

	// Add template management server in development mode
	if os.Getenv("EMAIL_ENV") == "development" || os.Getenv("EMAIL_ENV") == "" {
		invokers = append(invokers, setupTemplateServer)
	}

	// Use fx for dependency injection
	app := fx.New(
		// Provide all the constructors
		fx.Provide(providers...),
		// Register lifecycle hooks
		fx.Invoke(invokers...),
		// Disable verbose logging
		fx.NopLogger,
	)

	// Start the application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Set up signal handling for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-signalCh
	log.Println("Received termination signal, shutting down...")

	// Stop the application
	if err := app.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}

	log.Println("Email service shutdown complete")
}

// setupConsumer registers the RabbitMQ consumer and handlers
func setupConsumer(
	lc fx.Lifecycle,
	cfg *config.AppConfig,
	consumer *consumer.RabbitMQConsumer,
	sender *email.SenderService,
) {
	// Create email handler
	handler := consumer.NewEmailHandler(sender)

	// Register handlers for different message types
	consumer.RegisterHandler(model.TypeEmailSend, handler.HandleEmailSendMessage)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("Starting email service. Environment: %s", cfg.Environment)
			return consumer.Start(ctx)
		},
		OnStop: func(context.Context) error {
			log.Println("Stopping email service")
			return consumer.Stop()
		},
	})
}

// setupTemplateServer configures and starts the template management server (development only)
func setupTemplateServer(
	lc fx.Lifecycle,
	templateService *email.TemplateService,
) {
	// Create the template server
	srv := server.NewServer(":3002", templateService, "templates")

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start the template server in a separate goroutine
			go func() {
				log.Printf(
					"Starting template management server on http://localhost:3002 (DEV MODE ONLY)",
				)
				if err := srv.Start(); err != nil {
					log.Printf("Template server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			log.Println("Stopping template management server")
			return nil // No explicit stop needed for the HTTP server
		},
	})
}
