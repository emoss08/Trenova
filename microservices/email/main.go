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
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Use fx for dependency injection
	app := fx.New(
		// Provide all the constructors
		fx.Provide(
			config.LoadConfig,
			provider.NewSMTPProvider,
			provider.NewSendGridProvider,
			provider.NewFactory,
			email.NewTemplateService,
			email.NewSenderService,
			consumer.NewRabbitMQConsumer,
		),
		// Register lifecycle hooks
		fx.Invoke(setupConsumer),
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
